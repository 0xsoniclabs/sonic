package cert

import (
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/stretchr/testify/require"
)

func TestAggregatedSignature_Add_SignaturesCanBeAggregated(t *testing.T) {
	require := require.New(t)
	key1 := bls.NewPrivateKey()
	key2 := bls.NewPrivateKey()
	key3 := bls.NewPrivateKey()

	stmt := testStatement(1)
	sig1 := Sign(stmt, key1)
	sig2 := Sign(stmt, key2)
	sig3 := Sign(stmt, key3)

	agg := AggregatedSignature[testStatement]{}
	require.NoError(agg.Add(1, sig1))
	require.NoError(agg.Add(2, sig2))
	require.NoError(agg.Add(3, sig3))
}

func TestAggregatedSignature_Add_DuplicatesAreRejected(t *testing.T) {
	require := require.New(t)
	key := bls.NewPrivateKey()

	stmt := testStatement(1)
	sig := Sign(stmt, key)

	agg := AggregatedSignature[testStatement]{}
	require.NoError(agg.Add(1, sig))
	require.Error(agg.Add(1, sig))
}

func TestAggregatedSignature_Verify_CanVerifyValidSignature(t *testing.T) {
	require := require.New(t)
	key0 := bls.NewPrivateKey()
	key1 := bls.NewPrivateKey()
	key2 := bls.NewPrivateKey()

	stmt := testStatement(1)
	sig0 := Sign(stmt, key0)
	sig1 := Sign(stmt, key1)
	sig2 := Sign(stmt, key2)

	agg := AggregatedSignature[testStatement]{}
	require.NoError(agg.Add(0, sig0))
	require.NoError(agg.Add(1, sig1))
	require.NoError(agg.Add(2, sig2))

	committee, err := scc.NewCommittee(
		newMember(key0, 1),
		newMember(key1, 2),
		newMember(key2, 3),
	)
	require.NoError(err)
	require.NoError(agg.Verify(committee, committee, stmt))
}

func TestAggregatedSignature_Verify_CanVerifyAuthorityOfDifferentProducerCommittee(t *testing.T) {
	require := require.New(t)
	key0 := bls.NewPrivateKey()
	key1 := bls.NewPrivateKey()
	key2 := bls.NewPrivateKey()

	authority, err := scc.NewCommittee(
		newMember(key0, 1),
		newMember(key1, 2),
	)
	require.NoError(err)

	producer, err := scc.NewCommittee(
		newMember(key0, 1),
		newMember(key1, 2),
		newMember(key2, 3),
	)
	require.NoError(err)

	stmt := testStatement(1)

	// -- Valid for the Authority but not the Producer --

	agg := AggregatedSignature[testStatement]{}
	require.NoError(agg.Add(0, Sign(stmt, key0)))
	require.NoError(agg.Add(1, Sign(stmt, key1)))

	// This aggregated signature is not valid for the producer committee.
	require.Error(agg.Verify(producer, producer, stmt))

	// But the authority committee can verify it.
	require.NoError(agg.Verify(authority, producer, stmt))

	// --- Valid for the Producer but not the Authority ---

	agg = AggregatedSignature[testStatement]{}
	require.NoError(agg.Add(1, Sign(stmt, key1)))
	require.NoError(agg.Add(2, Sign(stmt, key2)))

	// This aggregated signature is valid for the producer committee.
	require.NoError(agg.Verify(producer, producer, stmt))

	// But the authority committee can not verify it.
	require.Error(agg.Verify(authority, producer, stmt))
}

func TestAggregatedSignature_Verify_DetectsInvalidProducerCommittee(t *testing.T) {
	require := require.New(t)
	agg := AggregatedSignature[testStatement]{}
	err := agg.Verify(scc.Committee{}, scc.Committee{}, testStatement(1))
	require.Error(err)
	require.ErrorContains(err, "invalid producer committee")
}

func TestAggregatedSignature_Verify_DetectsInvalidSigner(t *testing.T) {
	require := require.New(t)

	key0 := bls.NewPrivateKey()
	key1 := bls.NewPrivateKey()

	committee, err := scc.NewCommittee(newMember(key0, 1))
	require.NoError(err)

	stmt := testStatement(1)
	agg := AggregatedSignature[testStatement]{}
	require.NoError(agg.Add(0, Sign(stmt, key0)))
	require.NoError(agg.Add(1, Sign(stmt, key1))) // < signer not in committee

	err = agg.Verify(committee, committee, stmt)
	require.Error(err)
	require.ErrorContains(err, "signer 1 not found in producer committee")
}

func TestAggregatedSignature_Verify_DetectsOverflowInAuthorityCommitteeVotingPower(t *testing.T) {
	require := require.New(t)

	key0 := bls.NewPrivateKey()
	key1 := bls.NewPrivateKey()

	member0 := newMember(key0, 1)
	member1 := newMember(key1, 2)

	validCommittee, err := scc.NewCommittee(member0, member1)
	require.NoError(err)

	member0.VotingPower = math.MaxUint64
	invalidCommittee := scc.NewCommitteeForTests(member0, member1)
	require.Error(invalidCommittee.Validate())

	agg := AggregatedSignature[testStatement]{}
	err = agg.Verify(invalidCommittee, validCommittee, testStatement(1))
	require.Error(err)
	require.ErrorContains(err, "total voting power exceeds maximum")
}

func TestAggregatedSignature_Verify_DetectsInsufficientVotingPower(t *testing.T) {
	require := require.New(t)

	keys := make([]bls.PrivateKey, 6)
	members := make([]scc.Member, 6)
	for i := range keys {
		keys[i] = bls.NewPrivateKey()
		members[i] = newMember(keys[i], 1)
	}
	committee, err := scc.NewCommittee(members...)
	require.NoError(err)

	stmt := testStatement(1)
	agg := AggregatedSignature[testStatement]{}
	for i, key := range keys {
		sig := Sign(stmt, key)
		require.NoError(agg.Add(scc.MemberId(i), sig))
		err := agg.Verify(committee, committee, stmt)
		if i < 4 { // 4 signatures are not enough, 5 of 6 are required
			require.Error(err)
			require.ErrorContains(err, "insufficient voting power")
		} else {
			require.NoError(err)
		}
	}
}

func TestAggregatedSignature_Verify_DetectsWrongStatement(t *testing.T) {
	require := require.New(t)
	key := bls.NewPrivateKey()
	stmt := testStatement(1)

	committee, err := scc.NewCommittee(newMember(key, 1))
	require.NoError(err)

	tests := map[string]struct {
		sig  Signature[testStatement]
		stmt testStatement
	}{
		"wrong statement": {
			sig:  Sign(stmt, key),
			stmt: stmt + 1,
		},
		"wrong signature": {
			sig:  Sign(stmt+1, key),
			stmt: stmt,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			agg := AggregatedSignature[testStatement]{}
			require.NoError(agg.Add(0, test.sig))
			err := agg.Verify(committee, committee, test.stmt)
			require.ErrorContains(err, "invalid aggregated signature")
		})
	}
}

func TestAggregatedSignature_String_ListsKeyProperties(t *testing.T) {
	require := require.New(t)
	stmt := testStatement(1)

	agg := AggregatedSignature[testStatement]{}

	print := agg.String()
	require.Contains(print, "signers={}")
	require.Contains(print, "signature=0xc000..0000")

	require.NoError(agg.Add(1, Sign(stmt, bls.NewPrivateKeyForTests(1))))

	print = agg.String()
	require.Contains(print, "signers={1}")
	require.Contains(print, "signature=0xa9b5..ba22")

	agg.Add(7, Sign(stmt, bls.NewPrivateKeyForTests(2)))

	print = agg.String()
	require.Contains(print, "signers={1, 7}")
	require.Contains(print, "signature=0xb96b..b00c")
}

func newMember(key bls.PrivateKey, power uint64) scc.Member {
	return scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       power,
	}
}
