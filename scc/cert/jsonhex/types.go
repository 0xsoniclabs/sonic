package jsonhex

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Bytes represents a byte slice that is serialized as a hexadecimal string with a "0x" prefix.
type Bytes []byte

// MarshalJSON converts the HexBytes into a JSON-compatible hex string with a "0x" prefix.
func (h Bytes) MarshalJSON() ([]byte, error) {
	return []byte(h.String()), nil
}

// UnmarshalJSON parses a JSON hex string into a HexBytes slice.
// The input string must have a "0x" prefix or be "null".
func (h *Bytes) UnmarshalJSON(data []byte) error {
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
func (h Bytes) String() string {
	if h == nil {
		return "null"
	}
	return fmt.Sprintf("0x%x", []byte(h))
}

// UnmarshalFixLengthHexBytes decodes a JSON hex string into a fixed-length HexBytes slice.
// Returns an error if the decoded length does not match the expected length.
func UnmarshalFixLengthHexBytes(data []byte, length int) (Bytes, error) {
	var h Bytes
	err := h.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	if len(h) != length {
		return nil, fmt.Errorf("invalid length %d, expected %d", len(h), length)
	}
	return h, nil
}

// Bytes48 is a fixed-size [48]byte array that serializes as a hex string with a "0x" prefix
type Bytes48 [48]byte

// MarshalJSON converts the HexBytes48 into a JSON-compatible hex string.
func (h *Bytes48) MarshalJSON() ([]byte, error) {
	return Bytes((*h)[:]).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a HexBytes48.
func (h *Bytes48) UnmarshalJSON(data []byte) error {
	hexBytes, err := UnmarshalFixLengthHexBytes(data, 48)
	if err != nil {
		return err
	}
	*h = Bytes48(hexBytes)
	return nil
}

// String returns the hex string representation of HexBytes48.
func (h Bytes48) String() string {
	return Bytes(h[:]).String()
}

// Bytes96 is a fixed-size [96]byte array that serializes as a hex string with a "0x" prefix.
type Bytes96 [96]byte

// MarshalJSON converts the HexBytes96 into a JSON-compatible hex string.
func (h *Bytes96) MarshalJSON() ([]byte, error) {
	return Bytes((*h)[:]).MarshalJSON()
}

// UnmarshalJSON parses a JSON hex string into a HexBytes96.
func (h *Bytes96) UnmarshalJSON(data []byte) error {
	hexBytes, err := UnmarshalFixLengthHexBytes(data, 96)
	if err != nil {
		return err
	}
	*h = Bytes96(hexBytes)
	return nil
}

// String returns the hex string representation of HexBytes96.
func (h Bytes96) String() string {
	return Bytes(h[:]).String()
}
