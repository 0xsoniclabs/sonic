package serialization

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// HexBytes represents a byte slice that is serialized as a hexadecimal string with a "0x" prefix.
type HexBytes []byte

// MarshalJSON converts the HexBytes into a JSON-compatible hex string with a "0x" prefix.
func (h HexBytes) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"0x%x\"", []byte(h))), nil
}

// UnmarshalJSON parses a JSON hex string into a HexBytes slice.
// The input string must have a "0x" prefix and be an even-length hex string.
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

// String returns the hex string representation of HexBytes.
func (h HexBytes) String() string {
	return fmt.Sprintf("0x%x", []byte(h))
}

// UnmarshalFixLengthHexBytes decodes a JSON hex string into a fixed-length HexBytes slice.
// Returns an error if the decoded length does not match the expected length.
func UnmarshalFixLengthHexBytes(data []byte, length int) (HexBytes, error) {
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

// HexBytes48 is a fixed-size [48]byte array that serializes as a hex string with a "0x" prefix
type HexBytes48 [48]byte

// MarshalJSON converts the HexBytes48 into a JSON-compatible hex string.
func (p *HexBytes48) MarshalJSON() ([]byte, error) {
	return HexBytes((*p)[:]).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a HexBytes48.
func (p *HexBytes48) UnmarshalJSON(data []byte) error {
	hexBytes, err := UnmarshalFixLengthHexBytes(data, 48)
	if err != nil {
		return err
	}
	*p = HexBytes48(hexBytes)
	return nil
}

// String returns the hex string representation of HexBytes48.
func (p HexBytes48) String() string {
	return fmt.Sprintf("0x%x", []byte(p[:]))
}

// HexBytes96 is a fixed-size [96]byte array that serializes as a hex string with a "0x" prefix.
type HexBytes96 [96]byte

// MarshalJSON converts the HexBytes96 into a JSON-compatible hex string.
func (s *HexBytes96) MarshalJSON() ([]byte, error) {
	return HexBytes((*s)[:]).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a HexBytes96.
func (s *HexBytes96) UnmarshalJSON(data []byte) error {
	hexBytes, err := UnmarshalFixLengthHexBytes(data, 96)
	if err != nil {
		return err
	}
	*s = HexBytes96(hexBytes)
	return nil
}

// String returns the hex string representation of HexBytes96.
func (h HexBytes96) String() string {
	return fmt.Sprintf("0x%x", []byte(h[:]))
}
