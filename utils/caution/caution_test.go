package caution

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteAndReportError_ExecutesAndReturnsError(t *testing.T) {
	var err error
	ExecuteAndReportError(&err, func() error {
		return nil
	}, "message")
	require.NoError(t, err)
	someError := fmt.Errorf("someError")
	ExecuteAndReportError(&err, func() error {
		return someError
	}, "message")
	require.ErrorIs(t, err, someError)
}

func TestExecuteAndReportError_ExecutesAndCombinesErrors(t *testing.T) {
	firstError := fmt.Errorf("firstError")
	err := firstError
	ExecuteAndReportError(&err, func() error {
		return nil
	}, "message")
	require.ErrorIs(t, err, firstError)

	ExecuteAndReportError(&err, func() error {
		return fmt.Errorf("secondError")
	}, "message")
	require.ErrorContains(t, err, "firstError")
	require.ErrorContains(t, err, "secondError")
}

type closeMe struct {
	err error
}

func (c *closeMe) Close() error {
	return c.err
}

func TestCloseAndReportError_AddsMessageToError(t *testing.T) {
	file := &closeMe{}
	var err error
	CloseAndReportError(&err, file, "message")
	require.NoError(t, err)

	file.err = fmt.Errorf("someError")
	CloseAndReportError(&err, file, "message")
	require.ErrorContains(t, err, "message: someError")
}

func TestCloseAndReportError_UsagePatternPropagatesError(t *testing.T) {
	expectedError := fmt.Errorf("someError")

	testFun := func() (outErr error) {
		file := &closeMe{err: expectedError}
		defer CloseAndReportError(&outErr, file, "message")
		return
	}

	gotError := testFun()
	require.ErrorIs(t, gotError, expectedError)
}
