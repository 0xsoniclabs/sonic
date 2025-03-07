package ethapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBlockCertificate_ToBlockCertificate_ConvertsToHealthyCertificate(t *testing.T) {
	require := require.New(t)
	bitSet := cert.BitSet[scc.MemberId]{}
	bitSet.Add(1)
	newPrivateKey := bls.NewPrivateKey()
	sig := newPrivateKey.GetProofOfPossession()

	b := BlockCertificate{
		ChainId:   123,
		Number:    456,
		Hash:      common.Hash{0x1},
		StateRoot: common.Hash{0x2},
		Signers:   bitSet,
		Signature: sig,
	}

	got := b.ToCertificate()
	aggregatedSignature := cert.NewAggregatedSignature[cert.BlockStatement](
		b.Signers,
		b.Signature,
	)
	want := cert.NewCertificateWithSignature(cert.NewBlockStatement(
		b.ChainId,
		idx.Block(b.Number),
		b.Hash,
		b.StateRoot),
		aggregatedSignature)

	require.Equal(want, got)
}

func TestBlockCertificate_CanBeJsonEncodedAndDecoded(t *testing.T) {
	require := require.New(t)

	// setup
	private := bls.NewPrivateKey()
	agg := cert.AggregatedSignature[cert.BlockStatement]{}
	err := agg.Add(1, cert.Signature[cert.BlockStatement]{Signature: private.Sign([]byte{1})})
	require.NoError(err)
	c := cert.NewCertificateWithSignature(
		cert.NewBlockStatement(123, 456, common.Hash{0x1}, common.Hash{0x2}),
		agg,
	)

	// encode
	certJson := toJsonBlockCertificate(c)
	data, err := json.Marshal(certJson)
	require.NoError(err)

	// decode
	var decoded BlockCertificate
	err = json.Unmarshal(data, &decoded)
	require.NoError(err)

	// check
	require.Equal(certJson, decoded)
	cert := decoded.ToCertificate()
	require.Equal(c, cert)
}

func TestBlockCertificateToJson(t *testing.T) {
	require := require.New(t)
	bitset := cert.BitSet[scc.MemberId]{}
	bitset.Add(1)
	sig := bls.NewPrivateKey().GetProofOfPossession()
	cert := cert.NewCertificateWithSignature(
		cert.NewBlockStatement(123, 456, common.Hash{0x1}, common.Hash{0x2}),
		cert.NewAggregatedSignature[cert.BlockStatement](bitset, sig),
	)

	json := toJsonBlockCertificate(cert)
	require.Equal(uint64(123), json.ChainId)
	require.Equal(uint64(456), json.Number)
	require.Equal(common.Hash{0x1}, json.Hash)
	require.Equal(common.Hash{0x2}, json.StateRoot)
	require.Equal(bitset, json.Signers)
	require.Equal(sig, json.Signature)
}
func TestBlockCertificate_JsonEncodingMatchesExpectedFormat(t *testing.T) {
	certs := map[string]cert.BlockCertificate{
		"empty":     cert.BlockCertificate{},
		"non-empty": makeTestBlockCert(t),
	}

	hashRegex := `"0x[0-9a-f]{64}"`
	regexes := map[string]string{
		"chainId":   `"chainId":\d+`,
		"number":    `"number":\d+`,
		"hash":      `"hash":` + hashRegex,
		"stateRoot": `"stateRoot":` + hashRegex,
		"signers":   `"signers":("0x[0-9a-f]+"|null)`,
		"signature": `"signature":"0x[0-9a-f]{192}"`,
	}

	for name, cert := range certs {
		t.Run(name, func(t *testing.T) {
			for name, regex := range regexes {
				t.Run(name, func(t *testing.T) {
					require := require.New(t)
					data, err := json.Marshal(toJsonBlockCertificate(cert))
					require.NoError(err)
					require.Regexp(regexp.MustCompile(regex), string(data))
				})
			}
		})
	}
}

func TestBlockCertificate_EmptyCertificate_ContainsExpectedValues(t *testing.T) {
	require := require.New(t)
	emptyCert := cert.BlockCertificate{}
	data, err := json.Marshal(toJsonBlockCertificate(emptyCert))
	require.NoError(err)
	require.Contains(string(data), `"chainId":0`)
	require.Contains(string(data), `"number":0`)
	require.Contains(string(data), fmt.Sprintf(`"hash":"%v"`, common.Hash{}))
	require.Contains(string(data), fmt.Sprintf(`"stateRoot":"%v"`, common.Hash{}))
	require.Contains(string(data), `"signers":null`)
	require.Contains(string(data), fmt.Sprintf(`"signature":"%v"`, bls.Signature{}))
}

func TestBlockCertificate_NonEmptyCertificate_ContainsExpectedValues(t *testing.T) {
	require := require.New(t)
	testCert := makeTestBlockCert(t)
	agg := testCert.Signature()
	signers, err := json.Marshal(agg.Signers())
	require.NoError(err)

	data, err := json.Marshal(toJsonBlockCertificate(testCert))
	require.NoError(err)

	require.Contains(string(data), `"chainId":123`)
	require.Contains(string(data), `"number":456`)
	require.Contains(string(data), fmt.Sprintf(`"hash":"%v"`, common.Hash{0x1}))
	require.Contains(string(data), fmt.Sprintf(`"stateRoot":"%v"`, common.Hash{0x2}))
	require.Contains(string(data), fmt.Sprintf(`"signers":%v`, string(signers)))
	require.Contains(string(data), fmt.Sprintf(`"signature":"%v"`, agg.Signature()))
}

func makeTestBlockCert(t *testing.T) cert.BlockCertificate {
	key := bls.NewPrivateKey()
	sig := key.GetProofOfPossession()
	bitset := cert.BitSet[scc.MemberId]{}
	agg := cert.NewAggregatedSignature[cert.BlockStatement](
		bitset,
		sig,
	)
	err := agg.Add(1, cert.Signature[cert.BlockStatement]{Signature: key.Sign([]byte{1})})
	require.NoError(t, err)

	return cert.NewCertificateWithSignature(
		cert.NewBlockStatement(123, 456, common.Hash{0x1}, common.Hash{0x2}),
		agg,
	)
}
