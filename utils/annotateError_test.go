package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnnotateIfError_PropagatesNil(t *testing.T) {
	if AnnotateIfError(nil, "message") != nil {
		t.Error("AnnotateIfError should return nil when err is nil")
	}
}

func TestAnnotateIfError_AddsContextToError(t *testing.T) {
	err := fmt.Errorf("someError")
	errWithContext := AnnotateIfError(err, "message")
	require.ErrorContains(t, errWithContext, "message: someError")
}
