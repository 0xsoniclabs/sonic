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

package gossip

import (
	"bytes"
	"io"

	"github.com/golang/snappy"

	"github.com/ethereum/go-ethereum/p2p"
)

// sendBytes writes a raw byte payload (e.g. proto.Marshal output) as a
// snappy-compressed p2p message.
func sendBytes(w p2p.MsgWriter, msgcode uint64, data []byte) error {
	compressed := snappy.Encode(nil, data)
	return w.WriteMsg(p2p.Msg{Code: msgcode, Size: uint32(len(compressed)), Payload: bytes.NewReader(compressed)})
}

// decodeBytes reads the full message payload and decompresses it from snappy
// format (for proto.Unmarshal).
func decodeBytes(msg p2p.Msg) ([]byte, error) {
	compressed, err := io.ReadAll(io.LimitReader(msg.Payload, int64(msg.Size)))
	if err != nil {
		return nil, err
	}
	return snappy.Decode(nil, compressed)
}
