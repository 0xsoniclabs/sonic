// Copyright 2026 Sonic Operations Ltd
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

package core_types

// TransactionResult represents the result of executing a transaction within a
// bundle. It may be one of the following:
// - TransactionResultInvalid: The transaction is invalid (e.g., fails basic validation).
// - TransactionResultFailed: The transaction is valid but fails during execution (e.g., out of gas, revert).
// - TransactionResultSuccessful: The transaction is valid and executes successfully.
type TransactionResult int

const (
	TransactionResultInvalid TransactionResult = iota
	TransactionResultFailed
	TransactionResultSuccessful
)
