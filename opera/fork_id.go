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

package opera

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// ForkID derives the fork id of an upgrade, as eth_config reports it:
// CRC32(genesisId || bigEndian(upgrade.Height) || Rlp(upgrade)).
func ForkID(upgrade UpgradeHeight, genesisID common.Hash) ([4]byte, error) {
	encoded, err := rlp.EncodeToBytes(upgrade.Upgrades)
	if err != nil {
		return [4]byte{}, fmt.Errorf("could not encode upgrade to RLP, %v", err)
	}
	checksum := crc32.ChecksumIEEE(genesisID.Bytes())
	checksum = crc32.Update(checksum, crc32.IEEETable, binary.BigEndian.AppendUint64(nil, uint64(upgrade.Height)))
	checksum = crc32.Update(checksum, crc32.IEEETable, encoded)
	return [4]byte(binary.BigEndian.AppendUint32(nil, checksum)), nil
}
