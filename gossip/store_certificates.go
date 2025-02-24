package gossip

import (
	"encoding/binary"
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
)

// CommitteeCertificate is a certificate for a committee. This is an alias
// for cert.Certificate[cert.CommitteeStatement] to improve readability.
type CommitteeCertificate = cert.Certificate[cert.CommitteeStatement]

// CommitteeCertificate is a certificate for a block. This is an alias
// for cert.Certificate[cert.BlockStatement] to improve readability.
type BlockCertificate = cert.Certificate[cert.BlockStatement]

// UpdateCommitteeCertificate adds or updates the certificate in the store.
// If a certificate for the same period is already present, it is overwritten.
func (s *Store) UpdateCommitteeCertificate(certificate CommitteeCertificate) error {
	return updateCertificate(
		getCommitteeCertificateKey(certificate.Subject().Period),
		s.table.CommitteeCertificates,
		certificate,
	)
}

// GetCommitteeCertificate retrieves the certificate for the given period.
// If no certificate is found, an error is returned.
func (s *Store) GetCommitteeCertificate(period scc.Period) (CommitteeCertificate, error) {
	return getCertificate[cert.CommitteeStatement](
		getCommitteeCertificateKey(period),
		s.table.CommitteeCertificates,
	)
}

// UpdateBlockCertificate adds or updates the certificate in the store.
// If a certificate for the same block is already present, it is overwritten.
func (s *Store) UpdateBlockCertificate(certificate BlockCertificate) error {
	return updateCertificate(
		getBlockCertificateKey(certificate.Subject().Number),
		s.table.BlockCertificates,
		certificate,
	)
}

// GetBlockCertificate retrieves the certificate for the given block.
// If no certificate is found, an error is returned.
func (s *Store) GetBlockCertificate(block idx.Block) (BlockCertificate, error) {
	return getCertificate[cert.BlockStatement](
		getBlockCertificateKey(block),
		s.table.BlockCertificates,
	)
}

// getCommitteeCertificateKey returns the key used to store committee
// certificates in the key/value store.
func getCommitteeCertificateKey(period scc.Period) []byte {
	// big endian to sort entries in DB by period
	return binary.BigEndian.AppendUint64(nil, uint64(period))
}

// getBlockCertificateKey returns the key used to store block certificates
// in the key/value store.
func getBlockCertificateKey(number idx.Block) []byte {
	// big endian to sort entries in DB by block
	return binary.BigEndian.AppendUint64(nil, uint64(number))
}

// updateCertificate stores the certificate in the key/value store.
// If a certificate for the same key is already present, it is overwritten.
func updateCertificate[S cert.Statement](
	key []byte,
	table kvdb.Store,
	certificate cert.Certificate[S],
) error {
	data, err := certificate.Serialize()
	if err != nil {
		return err
	}
	return table.Put(key, data)
}

// getCertificate retrieves the certificate from the key/value store.
// If no certificate is found, an error is returned.
func getCertificate[S cert.Statement](
	key []byte,
	table kvdb.Store,
) (cert.Certificate[S], error) {
	var res cert.Certificate[S]
	data, err := table.Get(key)
	if err != nil {
		return res, err
	}
	if data == nil {
		return res, fmt.Errorf("no such certificate")
	}
	return res, res.Deserialize(data)
}
