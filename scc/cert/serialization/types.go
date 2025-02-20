package serialization

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type HexBytes []byte

func (h *HexBytes) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s == "null" {
		*h = nil
		return nil
	}
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 == 1 {
		s = "0" + s
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	*h = raw
	return nil
}

type PublicKey [48]byte

func (p *PublicKey) UnmarshalJSON(data []byte) error {
	hexBytes := HexBytes{}
	err := hexBytes.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	*p = PublicKey(hexBytes)
	return nil
}

func (p PublicKey) String() string {
	return fmt.Sprintf("0x%x", []byte(p[:]))
}

type Signature [96]byte

func (s *Signature) UnmarshalJSON(data []byte) error {
	hexBytes := HexBytes{}
	err := hexBytes.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	*s = Signature(hexBytes)
	return nil
}

func (h Signature) String() string {
	return fmt.Sprintf("0x%x", []byte(h[:]))
}
