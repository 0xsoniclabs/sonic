package gossip

import (
	"encoding/binary"
	"fmt"
	"iter"
	"math"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/result"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/log"
)

// UpdateCommitteeCertificate adds or updates the certificate in the store.
// If a certificate for the same period is already present, it is overwritten.
func (s *Store) UpdateCommitteeCertificate(certificate cert.CommitteeCertificate) error {
	return updateCertificate(
		uint64(certificate.Subject().Period),
		s.table.CommitteeCertificates,
		certificate,
	)
}

// GetCommitteeCertificate retrieves the certificate for the given period.
// If no certificate is found, an error is returned.
func (s *Store) GetCommitteeCertificate(period scc.Period) (cert.CommitteeCertificate, error) {
	return getCertificate[cert.CommitteeStatement](
		uint64(period),
		s.table.CommitteeCertificates,
	)
}

// GetLatestCommitteeCertificate retrieves the latest committee certificate from
// the store. If no certificate is found, an error is returned.
func (s *Store) GetLatestCommitteeCertificate() (cert.CommitteeCertificate, error) {
	return getLatestCertificate[cert.CommitteeStatement](s.table.CommitteeCertificates)
}

// EnumerateCommitteeCertificates iterates over all committee certificates
// starting from the given period. The certificates are yielded in ascending
// order of period. If an error occurs during iteration, it is yielded as the
// last result.
func (s *Store) EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[cert.CommitteeCertificate]] {
	return enumerateCertificates[cert.CommitteeStatement](
		uint64(first),
		s.table.CommitteeCertificates,
		s.Log,
	)
}

// UpdateBlockCertificate adds or updates the certificate in the store.
// If a certificate for the same block is already present, it is overwritten.
func (s *Store) UpdateBlockCertificate(certificate cert.BlockCertificate) error {
	return updateCertificate(
		uint64(certificate.Subject().Number),
		s.table.BlockCertificates,
		certificate,
	)
}

// GetBlockCertificate retrieves the certificate for the given block.
// If no certificate is found, an error is returned.
func (s *Store) GetBlockCertificate(block idx.Block) (cert.BlockCertificate, error) {
	return getCertificate[cert.BlockStatement](
		uint64(block),
		s.table.BlockCertificates,
	)
}

// GetLatestBlockCertificate retrieves the latest block certificate from the store.
// If no certificate is found, an error is returned.
func (s *Store) GetLatestBlockCertificate() (cert.BlockCertificate, error) {
	return getLatestCertificate[cert.BlockStatement](s.table.BlockCertificates)
}

// EnumerateBlockCertificates iterates over all block certificates starting from
// the given block number. The certificates are yielded in ascending order of
// block number. If an error occurs during iteration, it is yielded as the
// last result.
func (s *Store) EnumerateBlockCertificates(first idx.Block) iter.Seq[result.T[cert.BlockCertificate]] {
	return enumerateCertificates[cert.BlockStatement](
		uint64(first),
		s.table.BlockCertificates,
		s.Log,
	)
}

// getKey returns the key used to store certificates in the key/value store.
func getKey(index uint64) []byte {
	// big endian to sort entries in DB by period
	return binary.BigEndian.AppendUint64(nil, index)
}

// updateCertificate stores the certificate in the key/value store.
// If a certificate for the same key is already present, it is overwritten.
func updateCertificate[S cert.Statement](
	key uint64,
	table kvdb.Store,
	certificate cert.Certificate[S],
) error {
	data, err := certificate.Serialize()
	if err != nil {
		return err
	}
	return table.Put(getKey(key), data)
}

// getCertificate retrieves the certificate from the key/value store.
// If no certificate is found, an error is returned.
func getCertificate[S cert.Statement](
	key uint64,
	table kvdb.Store,
) (cert.Certificate[S], error) {
	var res cert.Certificate[S]
	data, err := table.Get(getKey(key))
	if err != nil {
		return res, err
	}
	if data == nil {
		return res, fmt.Errorf("no such certificate")
	}
	return res, res.Deserialize(data)
}

// getLatestCertificate retrieves the latest certificate from the key/value store.
// If no certificate is found, an error is returned.
func getLatestCertificate[S cert.Statement](
	table kvdb.Store,
) (cert.Certificate[S], error) {
	var res cert.Certificate[S]
	highest, err := findHighestKey(table)
	if err != nil {
		return res, err
	}
	data, err := table.Get(getKey(highest))
	if err != nil {
		return res, err
	}
	if len(data) == 0 {
		return res, fmt.Errorf("no such certificate")
	}
	if err := res.Deserialize(data); err != nil {
		return res, err
	}
	return res, nil
}

// enumerateCertificates iterates over all certificates in the key/value store
// starting from the given key. The certificates are yielded in ascending order
// of the key. If an error occurs during iteration, it is yielded as the last
// result.
func enumerateCertificates[S cert.Statement](
	first uint64,
	table kvdb.Store,
	log log.Logger,
) iter.Seq[result.T[cert.Certificate[S]]] {
	return func(yield func(result.T[cert.Certificate[S]]) bool) {
		it := table.NewIterator(nil, getKey(first))
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

// findHighestKey returns the highest key in the key/value store.
// If the store is empty, an error is returned.
func findHighestKey(table kvdb.Store) (uint64, error) {
	// Since the Ethereum DB doesn't support reverse iteration, we need to
	// run a binary search to find the highest key.
	return binarySearch(func(x uint64) (bool, error) {
		it := table.NewIterator(nil, getKey(x))
		defer it.Release()
		return it.Next(), it.Error()
	})
}

// binarySearch runs a binary search to find the highest uint64 such that the
// given check function returns true. It assumes that the check function is
// monotonic. If the check function always returns false, the searched range
// is empty and an error is returned.
func binarySearch(check func(uint64) (bool, error)) (uint64, error) {
	low := uint64(0)
	high := uint64(math.MaxUint64)

	// If there is no entry at all, an error is to be returned.
	ok, err := check(low)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("no such element")
	}

	// If MaxUint64 is in the DB, return it.
	ok, err = check(high)
	if err != nil {
		return 0, err
	}
	if ok {
		return high, nil
	}

	// Otherwise, run the binary search.
	for {
		mid := low + (high-low)/2
		if mid == low {
			return mid, nil
		}

		ok, err := check(mid)
		if err != nil {
			return 0, err
		}
		if ok {
			low = mid
		} else {
			high = mid
		}
	}
}
