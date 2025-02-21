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
	if !strings.HasPrefix(s, "0x") {
		return fmt.Errorf("invalid hex string %s", s)
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

func UnmarshallFixLenghtHexBytes(data []byte, length int) (HexBytes, error) {
	var h HexBytes
	err := h.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	if len(h) != length {
		return nil, fmt.Errorf("invalid length %d, expected %d", len(h), length)
	}
	return h, nil
}

type PublicKey [48]byte

func (p *PublicKey) UnmarshalJSON(data []byte) error {
	hexBytes, err := UnmarshallFixLenghtHexBytes(data, 48)
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
	hexBytes, err := UnmarshallFixLenghtHexBytes(data, 96)
	if err != nil {
		return err
	}
	*s = Signature(hexBytes)
	return nil
}

func (h Signature) String() string {
	return fmt.Sprintf("0x%x", []byte(h[:]))
}
