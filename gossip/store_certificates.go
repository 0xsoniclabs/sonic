package gossip

import (
	"encoding/binary"
	"fmt"
	"iter"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/result"
	"github.com/ethereum/go-ethereum/log"
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

// EnumerateCommitteeCertificates iterates over all committee certificates
// starting from the given period. The certificates are yielded in ascending
// order of period. If an error occurs during iteration, it is yielded as the
// last result.
func (s *Store) EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[CommitteeCertificate]] {
	return enumerateCertificates[cert.CommitteeStatement](
		getCommitteeCertificateKey(first),
		s.table.CommitteeCertificates,
		s.Log,
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
func (s *Store) GetBlockCertificate(block consensus.BlockID) (BlockCertificate, error) {
	return getCertificate[cert.BlockStatement](
		getBlockCertificateKey(block),
		s.table.BlockCertificates,
	)
}

// EnumerateBlockCertificates iterates over all block certificates starting from
// the given block number. The certificates are yielded in ascending order of
// block number. If an error occurs during iteration, it is yielded as the
// last result.
func (s *Store) EnumerateBlockCertificates(first consensus.BlockID) iter.Seq[result.T[BlockCertificate]] {
	return enumerateCertificates[cert.BlockStatement](
		getBlockCertificateKey(first),
		s.table.BlockCertificates,
		s.Log,
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
func getBlockCertificateKey(number consensus.BlockID) []byte {
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

// enumerateCertificates iterates over all certificates in the key/value store
// starting from the given key. The certificates are yielded in ascending order
// of the key. If an error occurs during iteration, it is yielded as the last
// result.
func enumerateCertificates[S cert.Statement](
	first []byte,
	table kvdb.Store,
	log log.Logger,
) iter.Seq[result.T[cert.Certificate[S]]] {
	return func(yield func(result.T[cert.Certificate[S]]) bool) {
		it := table.NewIterator(nil, first)
		defer it.Release()
		var res cert.Certificate[S]
		for it.Next() {
			// stop iteration if there is an error in the DB iterator
			if it.Error() != nil {
				log.Warn("Failed to iterate over certificates", "err", it.Error())
				yield(result.Error[cert.Certificate[S]](it.Error()))
				return
			}
			data := it.Value()
			if err := res.Deserialize(data); err != nil {
				log.Warn("Failed to deserialize certificate", "err", err)
				yield(result.Error[cert.Certificate[S]](err))
				return
			}
			if !yield(result.New(res)) {
				return
			}
		}
		// check for errors after the loop to catch any errors that may have
		// occurred after the last successful iteration
		if it.Error() != nil {
			log.Warn("Failed to iterate over certificates", "err", it.Error())
			yield(result.Error[cert.Certificate[S]](it.Error()))
		}
	}
}
