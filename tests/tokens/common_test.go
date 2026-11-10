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

package tokens

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/tokenreceiver"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// tokenActor names the parties of a token test case. Alice holds the tokens,
// Bob is the spender or operator, and Carol is the recipient. The deployer
// receives all tokens when the contract is created and funds Alice. The token
// contract itself serves as a recipient that implements no receiver hooks;
// the accepting and the rejecting receivers implement them, answering with
// the right and a wrong value respectively.
type tokenActor int

const (
	alice tokenActor = iota
	bob
	carol
	deployer
	tokenContract
	acceptingReceiver
	rejectingReceiver
	zeroAddress
)

func (a tokenActor) String() string {
	return [...]string{"alice", "bob", "carol", "deployer", "tokenContract", "acceptingReceiver", "rejectingReceiver", "zeroAddress"}[a]
}

// GoString makes failure messages show actors by name instead of number.
func (a tokenActor) GoString() string {
	return a.String()
}

type tokenActors [zeroAddress + 1]common.Address

func (a tokenActors) of(t *testing.T, address common.Address) tokenActor {
	t.Helper()
	for actor, candidate := range a {
		if candidate == address && (tokenActor(actor) == zeroAddress || address != (common.Address{})) {
			return tokenActor(actor)
		}
	}
	require.Fail(t, "address is none of the case's actors", address.Hex())
	return 0
}

// tokenAction issues a transaction on the token contract. The id is the token
// of the case, which only ERC-721 cases use.
type tokenAction[T any] func(o *bind.TransactOpts, token *T, a tokenActors, id *big.Int) (*types.Transaction, error)

// tokenStep is a setup transaction of a test case.
type tokenStep[T any] struct {
	sender tokenActor // Alice, Bob or Carol
	action tokenAction[T]
}

// tokenTest is a transaction sent by sender after the setup transactions. If
// want is nil, the transaction must revert, emitting no logs and leaving the
// state unchanged. Otherwise it must succeed, emit wantLogs, and result in the
// state want.
type tokenTest[T, S any] struct {
	setup    []tokenStep[T]
	sender   tokenActor
	action   tokenAction[T]
	wantLogs tokenLogs
	want     *S
}

// deployToken deploys the token contract. It returns the actors shared by all
// cases.
func deployToken[T any](
	t *testing.T,
	session tests.IntegrationTestNetSession,
	deploy tests.ContractDeployer[T],
) (*T, *types.Receipt, tokenActors) {
	t.Helper()
	token, deployment, err := tests.DeployContract(session, deploy)
	require.NoError(t, err, "deploying the token contract")
	require.Equal(t,
		types.ReceiptStatusSuccessful, deployment.Status,
		"deploying the token contract must succeed")
	return token, deployment, tokenActors{
		deployer:      session.GetSessionSponsor().Address(),
		tokenContract: deployment.ContractAddress,
	}
}

// addTokenReceivers deploys a pair of receivers, accepting and rejecting
// tokens, and adds them to the shared actors.
func addTokenReceivers(t *testing.T, session tests.IntegrationTestNetSession, shared tokenActors) tokenActors {
	t.Helper()
	shared[acceptingReceiver] = deployTokenReceiver(t, session, true)
	shared[rejectingReceiver] = deployTokenReceiver(t, session, false)
	return shared
}

func deployTokenReceiver(t *testing.T, session tests.IntegrationTestNetSession, accept bool) common.Address {
	t.Helper()
	_, receipt, err := tests.DeployContract(session, func(o *bind.TransactOpts, b bind.ContractBackend) (common.Address, *types.Transaction, *tokenreceiver.Tokenreceiver, error) {
		return tokenreceiver.DeployTokenreceiver(o, b, accept)
	})
	require.NoError(t, err, "deploying a token receiver")
	require.Equal(t,
		types.ReceiptStatusSuccessful, receipt.Status,
		"deploying a token receiver must succeed")
	return receipt.ContractAddress
}

// tokenParties are the funded accounts of Alice, Bob and Carol of one case,
// the addresses of all actors, and the token id of the case. Cases share the
// session but not its sponsor: each sends only from its own accounts, so
// cases can run in parallel without nonce conflicts and without observing
// each other's balances or approvals.
type tokenParties struct {
	accounts []*tests.Account
	actors   tokenActors
	tokenId  *big.Int
}

// newTokenParties creates the parties of n cases; case i uses token id i+1.
func newTokenParties(t *testing.T, session tests.IntegrationTestNetSession, shared tokenActors, n int) []tokenParties {
	t.Helper()
	accounts := tests.MakeAccountsWithBalance(t, session, 3*n, big.NewInt(1e17))
	parties := make([]tokenParties, n)
	for i := range parties {
		p := tokenParties{accounts: accounts[3*i : 3*i+3], actors: shared, tokenId: big.NewInt(int64(i + 1))}
		for actor, account := range p.accounts {
			p.actors[actor] = account.Address()
		}
		parties[i] = p
	}
	return parties
}

// fundTokenParties sends the fund action for each party from the session
// sponsor, the deployer, which holds all tokens after the creation.
func fundTokenParties[T any](
	t *testing.T,
	session tests.IntegrationTestNetSession,
	token *T,
	parties []tokenParties,
	fund tokenAction[T],
) {
	t.Helper()
	opts, err := session.GetTransactOptions(session.GetSessionSponsor())
	require.NoError(t, err, "transaction options of the deployer")
	hashes := make([]common.Hash, len(parties))
	for i, p := range parties {
		tx, err := issue(token, p, fund)(opts)
		require.NoError(t, err, "issuing the funding of case %d", i)
		hashes[i] = tx.Hash()
		opts.Nonce = new(big.Int).Add(opts.Nonce, big.NewInt(1))
	}
	receipts, err := session.GetReceipts(hashes)
	require.NoError(t, err, "receipts of the funding transactions")
	for i, receipt := range receipts {
		require.Equal(t,
			types.ReceiptStatusSuccessful, receipt.Status,
			"funding Alice of case %d must succeed", i)
	}
}

// issue binds the action to the token and the parties of a case.
func issue[T any](token *T, p tokenParties, action tokenAction[T]) func(*bind.TransactOpts) (*types.Transaction, error) {
	return func(o *bind.TransactOpts) (*types.Transaction, error) {
		return action(o, token, p.actors, p.tokenId)
	}
}

var (
	erc165InterfaceId  = [4]byte{0x01, 0xff, 0xc9, 0xa7}
	erc721InterfaceId  = [4]byte{0x80, 0xac, 0x58, 0xcd}
	erc1155InterfaceId = [4]byte{0xd9, 0xb6, 0x7a, 0x26}
)

// checkSupportedInterfaces checks supportsInterface for the given interface
// ids, and that it rejects 0xffffffff as ERC-165 requires.
func checkSupportedInterfaces(t *testing.T, supports func(*bind.CallOpts, [4]byte) (bool, error), want map[[4]byte]bool) {
	t.Helper()
	want[[4]byte{0xff, 0xff, 0xff, 0xff}] = false
	for id, expected := range want {
		got, err := supports(nil, id)
		require.NoError(t, err, "supportsInterface(%x)", id)
		require.Equal(t, expected, got, "supportsInterface(%x) must return %v", id, expected)
	}
}

// tokenLog is an emitted event in a form that can be written in a test table:
// addresses are replaced by actors, amounts and ids by ints, and the token id
// of an ERC-721 case by caseToken.
type tokenLog struct {
	event string
	args  []any
}

// GoString makes failure messages show events like the test tables do.
func (l tokenLog) GoString() string {
	args := make([]string, len(l.args))
	for i, arg := range l.args {
		args[i] = fmt.Sprint(arg)
	}
	return fmt.Sprintf("%s(%s)", l.event, strings.Join(args, ", "))
}

type tokenLogs []tokenLog

func emits(event string, args ...any) tokenLogs {
	return tokenLogs{{event: event, args: args}}
}

func (l tokenLogs) then(event string, args ...any) tokenLogs {
	return append(l, tokenLog{event: event, args: args})
}

type caseTokenId struct{}

func (caseTokenId) String() string {
	return "caseToken"
}

var caseToken caseTokenId

// parseEvents parses the JSON ABI of event definitions.
func parseEvents(definition string) *abi.ABI {
	events, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		panic(err)
	}
	return &events
}

// decodeTokenLogs decodes the logs of the receipt, checking that each was
// emitted by the token contract or a receiver. Token logs are decoded with
// tokenEvents, the event layouts of the token standard, rather than with the
// ABI of the contract under test, so that an event with a wrong signature or
// wrong indexed parameters fails to decode. If caseTokenId is non-nil, its
// occurrences are reported as caseToken.
func decodeTokenLogs(
	t *testing.T,
	tokenEvents *abi.ABI,
	receipt *types.Receipt,
	actors tokenActors,
	caseTokenId *big.Int,
) tokenLogs {
	t.Helper()
	receiverAbi, err := tokenreceiver.TokenreceiverMetaData.GetAbi()
	require.NoError(t, err, "ABI of the token receiver")
	abis := map[common.Address]*abi.ABI{
		actors[tokenContract]:     tokenEvents,
		actors[acceptingReceiver]: receiverAbi,
		actors[rejectingReceiver]: receiverAbi,
	}

	var res tokenLogs
	for _, log := range receipt.Logs {
		contractAbi, found := abis[log.Address]
		require.True(t,
			found,
			"log emitted by %v, which is neither the token contract nor a receiver", log.Address)
		require.NotEmpty(t, log.Topics, "events must not be anonymous")
		event, err := contractAbi.EventByID(log.Topics[0])
		require.NoError(t,
			err,
			"event %v must be one of the events of its emitter's standard", log.Topics[0])
		values := map[string]any{}
		contract := bind.NewBoundContract(log.Address, *contractAbi, nil, nil, nil)
		require.NoError(t,
			contract.UnpackLogIntoMap(values, event.Name, *log),
			"event %s must have the parameters and indexing of its standard", event.Name)

		args := make([]any, len(event.Inputs))
		for i, input := range event.Inputs {
			switch v := values[input.Name].(type) {
			case common.Address:
				args[i] = actors.of(t, v)
			case *big.Int:
				require.True(t,
					v.IsInt64(),
					"value %v of %s.%s does not fit the expected ints", v, event.Name, input.Name)
				args[i] = int(v.Int64())
				if caseTokenId != nil && v.Cmp(caseTokenId) == 0 {
					args[i] = caseToken
				}
			case []*big.Int:
				ints := make([]int, len(v))
				for j, x := range v {
					require.True(t,
						x.IsInt64(),
						"value %v of %s.%s does not fit the expected ints", x, event.Name, input.Name)
					ints[j] = int(x.Int64())
				}
				args[i] = ints
			default:
				args[i] = v
			}
		}
		res = res.then(event.Name, args...)
	}
	return res
}

// applyAs sends the transaction issued by the given account, waits for its
// receipt, and checks that it succeeded or reverted as expected.
func applyAs(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	account *tests.Account,
	wantSuccess bool,
	issue func(*bind.TransactOpts) (*types.Transaction, error),
) *types.Receipt {
	t.Helper()
	opts, err := session.GetTransactOptions(account)
	require.NoError(t, err, "transaction options of %v", account.Address())
	tx, err := issue(opts)
	require.NoError(t, err, "issuing the transaction")
	receipt, err := session.GetReceipt(tx.Hash())
	require.NoError(t, err, "receipt of the transaction")
	if wantSuccess {
		require.Equal(t,
			types.ReceiptStatusSuccessful, receipt.Status,
			"the transaction must succeed")
	} else {
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "the transaction must revert")
	}
	return receipt
}
