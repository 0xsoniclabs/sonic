package utils

import "fmt"

// AnnotateIfError adds a message to an error, if the error is not nil.
func AnnotateIfError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
