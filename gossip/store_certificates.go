package gossip

import (
	"encoding/binary"
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
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
	data, err := certificate.Serialize()
	if err != nil {
		return err
	}
	key := getCommitteeCertificateKey(certificate.Subject().Period)
	return s.table.CommitteeCertificates.Put(key, data)
}

// GetCommitteeCertificate retrieves the certificate for the given period.
// If no certificate is found, an error is returned.
func (s *Store) GetCommitteeCertificate(period scc.Period) (CommitteeCertificate, error) {
	var res CommitteeCertificate
	table := s.table.CommitteeCertificates
	data, err := table.Get(getCommitteeCertificateKey(period))
	if err != nil {
		return res, err
	}
	if data == nil {
		return res, fmt.Errorf("no certificate found for period %d", period)
	}
	return res, res.Deserialize(data)
}

// UpdateBlockCertificate adds or updates the certificate in the store.
// If a certificate for the same block is already present, it is overwritten.
func (s *Store) UpdateBlockCertificate(certificate BlockCertificate) error {
	data, err := certificate.Serialize()
	if err != nil {
		return err
	}
	key := getBlockCertificateKey(certificate.Subject().Number)
	return s.table.BlockCertificates.Put(key, data)
}

// GetBlockCertificate retrieves the certificate for the given block.
// If no certificate is found, an error is returned.
func (s *Store) GetBlockCertificate(block idx.Block) (BlockCertificate, error) {
	var res BlockCertificate
	table := s.table.BlockCertificates
	data, err := table.Get(getBlockCertificateKey(block))
	if err != nil {
		return res, err
	}
	if data == nil {
		return res, fmt.Errorf("no certificate found for block %d", block)
	}
	return res, res.Deserialize(data)
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
