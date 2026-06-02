package execution_api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TestVector represents a single JSON-RPC request/response pair from an .io file.
type TestVector struct {
	Request  json.RawMessage
	Response json.RawMessage
	Method   string // extracted from request for routing
	File     string // source .io file path
	Index    int    // pair index within the file (0-based)
}

// jsonRPCRequest is used to extract the method name from a request.
type jsonRPCRequest struct {
	Method string `json:"method"`
}

// ParseIOFile parses a single .io file into a slice of TestVector.
// Format: lines starting with ">>" are requests, lines starting with "<<" are responses.
// Lines starting with "//" are comments and are ignored.
func ParseIOFile(path string) ([]TestVector, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening .io file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var (
		vectors  []TestVector
		requests []json.RawMessage
		scanner  = bufio.NewScanner(f)
	)

	// Increase scanner buffer for long lines (block responses can be large)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, ">>") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, ">>"))
			requests = append(requests, json.RawMessage(payload))
		} else if strings.HasPrefix(line, "<<") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "<<"))
			if len(requests) == 0 {
				return nil, fmt.Errorf("response without preceding request in %s", path)
			}
			req := requests[len(requests)-1]
			requests = requests[:len(requests)-1]

			var rpcReq jsonRPCRequest
			if err := json.Unmarshal(req, &rpcReq); err != nil {
				return nil, fmt.Errorf("parsing request JSON in %s: %w", path, err)
			}

			vectors = append(vectors, TestVector{
				Request:  req,
				Response: json.RawMessage(payload),
				Method:   rpcReq.Method,
				File:     path,
				Index:    len(vectors),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning .io file: %w", err)
	}

	if len(requests) > 0 {
		return nil, fmt.Errorf("unpaired request(s) in %s", path)
	}

	return vectors, nil
}

// DiscoverTestVectors walks the tests directory and returns all test vectors
// grouped by method name (directory name).
func DiscoverTestVectors(testsDir string) (map[string][]TestVector, error) {
	result := make(map[string][]TestVector)

	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil, fmt.Errorf("reading tests directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		methodDir := filepath.Join(testsDir, entry.Name())
		ioFiles, err := filepath.Glob(filepath.Join(methodDir, "*.io"))
		if err != nil {
			return nil, fmt.Errorf("globbing .io files in %s: %w", methodDir, err)
		}

		for _, ioFile := range ioFiles {
			vectors, err := ParseIOFile(ioFile)
			if err != nil {
				return nil, err
			}
			result[entry.Name()] = append(result[entry.Name()], vectors...)
		}
	}

	return result, nil
}
