package execution_apis

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// CompareJSON compares an actual JSON-RPC response against the expected one.
// It returns nil if they match, or an error describing the difference.
//
// Comparison rules:
//   - Objects: all keys in expected must be present in actual with matching values.
//     Extra keys in actual are tolerated (Sonic may add fields not in spec).
//   - Arrays: must have same length and elements must match positionally.
//   - Scalars: must be equal.
//   - null in expected must match null in actual.
func CompareJSON(expected, actual json.RawMessage) error {
	var exp, act interface{}

	if err := json.Unmarshal(expected, &exp); err != nil {
		return fmt.Errorf("unmarshaling expected: %w", err)
	}
	if err := json.Unmarshal(actual, &act); err != nil {
		return fmt.Errorf("unmarshaling actual: %w", err)
	}

	return compareValues("", exp, act)
}

func compareValues(path string, expected, actual interface{}) error {
	if expected == nil && actual == nil {
		return nil
	}
	if expected == nil && actual != nil {
		return fmt.Errorf("at %s: expected null, got %v", pathOrRoot(path), actual)
	}
	if expected != nil && actual == nil {
		return fmt.Errorf("at %s: expected %v, got null", pathOrRoot(path), expected)
	}

	switch exp := expected.(type) {
	case map[string]interface{}:
		act, ok := actual.(map[string]interface{})
		if !ok {
			return fmt.Errorf("at %s: expected object, got %T", pathOrRoot(path), actual)
		}
		return compareObjects(path, exp, act)

	case []interface{}:
		act, ok := actual.([]interface{})
		if !ok {
			return fmt.Errorf("at %s: expected array, got %T", pathOrRoot(path), actual)
		}
		return compareArrays(path, exp, act)

	default:
		// Scalar comparison (strings, numbers, bools)
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("at %s: expected %v, got %v", pathOrRoot(path), expected, actual)
		}
		return nil
	}
}

func compareObjects(path string, expected, actual map[string]interface{}) error {
	var errs []string

	// Check all expected keys are present and match
	for key, expVal := range expected {
		childPath := path + "." + key
		actVal, exists := actual[key]
		if !exists {
			errs = append(errs, fmt.Sprintf("at %s: missing key %q", pathOrRoot(path), key))
			continue
		}
		if err := compareValues(childPath, expVal, actVal); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func compareArrays(path string, expected, actual []interface{}) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("at %s: array length mismatch: expected %d, got %d",
			pathOrRoot(path), len(expected), len(actual))
	}

	var errs []string
	for i := range expected {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		if err := compareValues(childPath, expected[i], actual[i]); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func pathOrRoot(path string) string {
	if path == "" {
		return "$"
	}
	return path
}
