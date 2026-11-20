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
	"maps"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc3009"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

func TestTokenStandards_ERC3009(t *testing.T) {
	// Alice starts with 100 tokens and authorizes transfers others submit.
	cases := map[string]struct {
		setup    []erc3009Auth
		auth     erc3009Auth
		wantLogs tokenLogs
		want     *erc3009State // nil if the transaction must revert
	}{
		"transferWithAuthorization submitted by a relayer succeeds": {
			auth:     authorize(alice, carol, 30),
			wantLogs: emits("AuthorizationUsed", alice, nonce(1)).then("Transfer", alice, carol, 30),
			want:     &erc3009State{balances{70, 0, 30}, true},
		},
		"transferWithAuthorization submitted by the payee succeeds": {
			auth:     authorize(alice, carol, 30).submittedBy(carol),
			wantLogs: emits("AuthorizationUsed", alice, nonce(1)).then("Transfer", alice, carol, 30),
			want:     &erc3009State{balances{70, 0, 30}, true},
		},
		"transferWithAuthorization of the entire balance succeeds": {
			auth:     authorize(alice, carol, 100),
			wantLogs: emits("AuthorizationUsed", alice, nonce(1)).then("Transfer", alice, carol, 100),
			want:     &erc3009State{balances{0, 0, 100}, true},
		},
		"transferWithAuthorization with a fresh nonce after a used one succeeds": {
			setup:    []erc3009Auth{authorize(alice, carol, 30)},
			auth:     authorize(alice, carol, 30).withNonce(2),
			wantLogs: emits("AuthorizationUsed", alice, nonce(2)).then("Transfer", alice, carol, 30),
			want:     &erc3009State{balances{40, 0, 60}, true},
		},
		"transferWithAuthorization reusing a nonce fails": {
			setup: []erc3009Auth{authorize(alice, carol, 30)},
			auth:  authorize(alice, carol, 30),
		},
		"transferWithAuthorization with insufficient funds fails": {
			auth: authorize(alice, carol, 101),
		},
		"transferWithAuthorization signed by someone else than the holder fails": {
			auth: authorize(alice, carol, 30).signedBy(bob),
		},
		"transferWithAuthorization of another value than signed fails": {
			auth: authorize(alice, carol, 30).submittingValue(31),
		},
		"transferWithAuthorization before its validity fails": {
			auth: authorize(alice, carol, 30).validWithin(math.MaxBig256, math.MaxBig256),
		},
		"transferWithAuthorization after its validity fails": {
			auth: authorize(alice, carol, 30).validWithin(big.NewInt(0), big.NewInt(1)),
		},
		"transferWithAuthorization of a receive authorization fails": {
			auth: authorize(alice, carol, 30).signedFor(receiveWithAuthorization),
		},

		"receiveWithAuthorization submitted by the payee succeeds": {
			auth:     authorize(alice, carol, 30).receivedBy(carol),
			wantLogs: emits("AuthorizationUsed", alice, nonce(1)).then("Transfer", alice, carol, 30),
			want:     &erc3009State{balances{70, 0, 30}, true},
		},
		"receiveWithAuthorization submitted by someone else than the payee fails": {
			auth: authorize(alice, carol, 30).receivedBy(bob),
		},
		"receiveWithAuthorization of a transfer authorization fails": {
			auth: authorize(alice, carol, 30).receivedBy(carol).signedFor(transferWithAuthorization),
		},
	}

	net := tests.StartIntegrationTestNet(t)
	token, _, shared := deployToken(t, net, func(o *bind.TransactOpts, b bind.ContractBackend) (common.Address, *types.Transaction, *erc3009.Erc3009, error) {
		return erc3009.DeployErc3009(o, b, big.NewInt(int64(100*len(cases))))
	})

	names := slices.Sorted(maps.Keys(cases))
	parties := newTokenParties(t, net, shared, len(names))
	fundTokenParties(t, net, token, parties, func(o *bind.TransactOpts, c *erc3009.Erc3009, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.Transfer(o, a[alice], big.NewInt(100))
	})

	for i, name := range names {
		test, p := cases[name], parties[i]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, auth := range test.setup {
				applyAs(t, net, p.accounts[auth.submitter], true, auth.issue(t, net, token, p))
			}
			before := observeErc3009(t, token, p.actors, test.auth.nonce)
			receipt := applyAs(t, net, p.accounts[test.auth.submitter], test.want != nil, test.auth.issue(t, net, token, p))

			after := observeErc3009(t, token, p.actors, test.auth.nonce)
			require.Equal(t,
				test.wantLogs,
				decodeTokenLogs(t, erc3009Events, receipt, p.actors, nil),
				"the events emitted by the transaction; a reverted transaction emits none")
			if test.want == nil {
				require.Equal(t,
					before, after,
					"a reverted transaction must not change any balance or the state of the authorization")
			} else {
				require.Equal(t,
					*test.want, after,
					"the balances of Alice, Bob and Carol, and whether Alice's authorization is used, after the transaction")
			}
		})
	}
}

// erc3009Events are the events of EIP-20 and the one EIP-3009 requires.
var erc3009Events = parseEvents(`[
	{"type": "event", "name": "Transfer", "inputs": [
		{"name": "from", "type": "address", "indexed": true},
		{"name": "to", "type": "address", "indexed": true},
		{"name": "value", "type": "uint256"}]},
	{"type": "event", "name": "Approval", "inputs": [
		{"name": "owner", "type": "address", "indexed": true},
		{"name": "spender", "type": "address", "indexed": true},
		{"name": "value", "type": "uint256"}]},
	{"type": "event", "name": "AuthorizationUsed", "inputs": [
		{"name": "authorizer", "type": "address", "indexed": true},
		{"name": "nonce", "type": "bytes32", "indexed": true}]}
]`)

const (
	transferWithAuthorization = "TransferWithAuthorization"
	receiveWithAuthorization  = "ReceiveWithAuthorization"
)

// erc3009Auth is an authorization signed by signer for the EIP-712 type
// signedType, and the transaction by submitter using it, calling
// receiveWithAuthorization if receive is set. The submitted value and method
// differ from the signed ones only in cases checking tampering.
type erc3009Auth struct {
	signedType              string
	receive                 bool
	signer, submitter       tokenActor
	from, to                tokenActor
	value, signedValue      int64
	validAfter, validBefore *big.Int
	nonce                   [32]byte
}

// authorize is from authorizing a transfer of value to to, valid at any time
// and submitted by Bob as a relayer.
func authorize(from, to tokenActor, value int64) erc3009Auth {
	return erc3009Auth{
		signedType: transferWithAuthorization,
		signer:     from, submitter: bob,
		from: from, to: to,
		value: value, signedValue: value,
		validAfter: big.NewInt(0), validBefore: math.MaxBig256,
		nonce: nonce(1),
	}
}

func nonce(n byte) [32]byte {
	return [32]byte{n}
}

func (a erc3009Auth) submittedBy(submitter tokenActor) erc3009Auth {
	a.submitter = submitter
	return a
}

// receivedBy has the submitter use receiveWithAuthorization.
func (a erc3009Auth) receivedBy(submitter tokenActor) erc3009Auth {
	a.signedType = receiveWithAuthorization
	a.receive = true
	a.submitter = submitter
	return a
}

func (a erc3009Auth) signedBy(signer tokenActor) erc3009Auth {
	a.signer = signer
	return a
}

func (a erc3009Auth) signedFor(signedType string) erc3009Auth {
	a.signedType = signedType
	return a
}

func (a erc3009Auth) submittingValue(value int64) erc3009Auth {
	a.value = value
	return a
}

func (a erc3009Auth) withNonce(n byte) erc3009Auth {
	a.nonce = nonce(n)
	return a
}

func (a erc3009Auth) validWithin(after, before *big.Int) erc3009Auth {
	a.validAfter, a.validBefore = after, before
	return a
}

// issue signs the authorization with the signer's key and returns the
// submission of it.
func (a erc3009Auth) issue(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	token *erc3009.Erc3009,
	p tokenParties,
) func(*bind.TransactOpts) (*types.Transaction, error) {
	t.Helper()
	tokenAddress := p.actors[tokenContract]
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			a.signedType: {
				{Name: "from", Type: "address"},
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "validAfter", Type: "uint256"},
				{Name: "validBefore", Type: "uint256"},
				{Name: "nonce", Type: "bytes32"},
			},
		},
		PrimaryType: a.signedType,
		Domain: apitypes.TypedDataDomain{
			Name:              "ERC3009",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(session.GetChainId()),
			VerifyingContract: tokenAddress.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"from":        p.actors[a.from].Hex(),
			"to":          p.actors[a.to].Hex(),
			"value":       big.NewInt(a.signedValue),
			"validAfter":  a.validAfter,
			"validBefore": a.validBefore,
			"nonce":       a.nonce[:],
		},
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	require.NoError(t, err, "EIP-712 digest of the authorization")
	sig, err := crypto.Sign(digest, p.accounts[a.signer].PrivateKey)
	require.NoError(t, err, "signing the authorization")
	v, r, s := sig[64]+27, [32]byte(sig[:32]), [32]byte(sig[32:64])

	submit := token.TransferWithAuthorization
	if a.receive {
		submit = token.ReceiveWithAuthorization
	}
	return func(o *bind.TransactOpts) (*types.Transaction, error) {
		return submit(o, p.actors[a.from], p.actors[a.to], big.NewInt(a.value), a.validAfter, a.validBefore, a.nonce, v, r, s)
	}
}

type erc3009State struct {
	balances balances
	used     bool // whether Alice's authorization with the case's nonce is used
}

func observeErc3009(t *testing.T, token *erc3009.Erc3009, a tokenActors, nonce [32]byte) erc3009State {
	t.Helper()
	var s erc3009State
	for i := range s.balances {
		balance, err := token.BalanceOf(nil, a[i])
		require.NoError(t, err, "balanceOf %v", tokenActor(i))
		require.True(t, balance.IsUint64(), "balanceOf %v exceeds uint64: %v", tokenActor(i), balance)
		s.balances[i] = balance.Uint64()
	}
	used, err := token.AuthorizationState(nil, a[alice], nonce)
	require.NoError(t, err, "authorizationState of Alice's nonce")
	s.used = used
	return s
}
