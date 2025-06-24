// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

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
