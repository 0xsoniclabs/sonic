package evmcore

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"
)

type SonicSigner struct {
	NetworkID *big.Int
}

func NewSonicSigner(networkID *big.Int) *SonicSigner {
	return &SonicSigner{
		NetworkID: networkID,
	}
}

func (s *SonicSigner) ChainID() *big.Int {
	return s.NetworkID
}

func (s *SonicSigner) Equal(other types.Signer) bool {
	if SonicSigner, ok := other.(*SonicSigner); ok {
		return s.NetworkID == SonicSigner.NetworkID
	}
	return false
}

func (s *SonicSigner) Hash(tx *types.Transaction) common.Hash {
	signer := types.LatestSignerForChainID(s.ChainID())
	return signer.Hash(tx)
}

func (s *SonicSigner) Sender(tx *types.Transaction) (common.Address, error) {

	if found, ok := senderCache.Get(tx.Hash()); ok {
		return found.(common.Address), nil
	}
	key := string(tx.Hash().Bytes())

	if found, ok := senderCache.Get(key); ok {
		return found.(common.Address), nil
	}

	signer := types.LatestSignerForChainID(s.ChainID())
	if addr, err := signer.Sender(tx); err == nil {
		senderCache.Add(key, addr)
		return addr, nil
	} else {
		return common.Address{}, err
	}
}

func (s *SonicSigner) sender(tx *types.Transaction) (common.Address, error) {
	signer := types.LatestSignerForChainID(s.ChainID())
	return signer.Sender(tx)
}

func (*SonicSigner) SignatureValues(tx *types.Transaction, sig []byte) (r *big.Int, s *big.Int, v *big.Int, err error) {
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(0))
	return signer.SignatureValues(tx, sig)
}

var _ types.Signer = (*SonicSigner)(nil)

var senderCache *lru.Cache

func init() {
	senderCache, _ = lru.New(100 * 1024)
}
