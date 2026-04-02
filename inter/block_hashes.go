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

package inter

import (
	"crypto/sha256"

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/0xsoniclabs/sonic/utils/cser"
)

// BlockHashes contains hashes of recently processed blocks to be included
// in version 4 events. Each entry is the Ethereum block hash for a
// consecutive range of blocks within a single epoch.
type BlockHashes struct {
	Start  idx.Block
	Epoch  idx.Epoch
	Hashes []hash.Hash
}

// LastBlock returns the block number of the last block in the range.
func (bh BlockHashes) LastBlock() idx.Block {
	return bh.Start + idx.Block(len(bh.Hashes)) - 1
}

// Hash computes a deterministic hash over all block hashes fields.
func (bh BlockHashes) Hash() hash.Hash {
	hasher := sha256.New()
	hasher.Write(bh.Start.Bytes())
	hasher.Write(bh.Epoch.Bytes())
	hasher.Write(bigendian.Uint32ToBytes(uint32(len(bh.Hashes))))
	for _, h := range bh.Hashes {
		hasher.Write(h.Bytes())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

// MarshalCSER serializes block hashes in CSER format.
func (bh BlockHashes) MarshalCSER(w *cser.Writer) error {
	w.U64(uint64(bh.Start))
	w.U32(uint32(bh.Epoch))
	w.U32(uint32(len(bh.Hashes)))
	for _, h := range bh.Hashes {
		w.FixedBytes(h[:])
	}
	return nil
}

// UnmarshalCSER deserializes block hashes from CSER format.
func (bh *BlockHashes) UnmarshalCSER(r *cser.Reader) error {
	start := r.U64()
	epoch := r.U32()
	num := r.U32()
	if num > ProtocolMaxMsgSize/32 {
		return cser.ErrTooLargeAlloc
	}
	hashes := make([]hash.Hash, num)
	for i := range hashes {
		r.FixedBytes(hashes[i][:])
	}
	bh.Start = idx.Block(start)
	bh.Epoch = idx.Epoch(epoch)
	bh.Hashes = hashes
	return nil
}
