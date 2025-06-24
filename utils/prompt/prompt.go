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

package prompt

import geth_prompt "github.com/ethereum/go-ethereum/console/prompt"

//go:generate mockgen -source prompt.go -destination prompt_mock.go -package prompt

// UserPrompt is the default prompter used by the console to prompt the user for
// various types of inputs.
// In enables tests to replace the prompter with a mock.
var UserPrompt UserPrompter = geth_prompt.Stdin

// UserPrompter is a re-export of the geth_prompt.UserPrompter interface.
// It is used to generate mocks for tests.
type UserPrompter interface {
	PromptInput(prompt string) (string, error)
	PromptPassword(prompt string) (string, error)
	PromptConfirm(prompt string) (bool, error)
	SetHistory(history []string)
	AppendHistory(command string)
	ClearHistory()
	SetWordCompleter(completer geth_prompt.WordCompleter)
}

// static assert: user prompter declared in this file must implement the one in geth_prompt
var _ geth_prompt.UserPrompter = UserPrompt
