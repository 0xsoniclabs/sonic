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

package bundle

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type BundleUnitRlp interface {
	isBundleUnitRlp()
}

func (bt *BundleTransaction) isBundleUnitRlp() {}

func (bl *BundleLayerRlp) isBundleUnitRlp() {}

type BundleUnitWrapperRlp struct {
	Inner BundleUnitRlp
}

type BundleLayerRlp struct {
	Units []BundleUnitWrapperRlp
	Flags ExecutionFlag
}

type TransactionBundleRlp struct {
	Layer    BundleLayerRlp
	Earliest uint64
	Latest   uint64
}

func (bu *BundleUnitWrapperRlp) EncodeRLP(w io.Writer) error {
	switch bu.Inner.(type) {
	case *BundleTransaction:
		typeId := uint8(1)
		return rlp.Encode(w, []any{typeId, bu.Inner.(*BundleTransaction)})
	case *BundleLayerRlp:
		typeId := uint8(2)
		return rlp.Encode(w, []any{typeId, bu.Inner.(*BundleLayerRlp)})
	default:
		panic(fmt.Sprintf("invalid type: %T", bu.Inner))
		// return fmt.Errorf("invalid type: %T", bu.Inner)
	}
}

func (bu *BundleUnitWrapperRlp) DecodeRLP(s *rlp.Stream) error {
	var content struct {
		TypeId uint8
		Raw    rlp.RawValue
	}
	if err := s.Decode(&content); err != nil {
		return err
	}

	switch content.TypeId {
	case 1:
		var tx BundleTransaction
		if err := rlp.DecodeBytes(content.Raw, &tx); err != nil {
			return fmt.Errorf("failed to decode transaction: %v", err)
		}
		bu.Inner = &tx
		return nil
	case 2:
		var layer BundleLayerRlp
		if err := rlp.DecodeBytes(content.Raw, &layer); err != nil {
			return fmt.Errorf("failed to decode bundle layer: %v", err)
		}
		bu.Inner = &layer
		return nil
	default:
		panic(fmt.Sprintf("invalid type: %d", content.TypeId))
		// return fmt.Errorf("invalid type: %d", content.TypeId)
	}
}

func wrap(unit BundleUnit) BundleUnitWrapperRlp {
	switch u := unit.(type) {
	case *BundleTransaction:
		return BundleUnitWrapperRlp{Inner: u}
	case *BundleLayer:
		layerRlp := BundleLayerRlp{Flags: u.Flags}
		for _, innerUnit := range u.Units {
			layerRlp.Units = append(layerRlp.Units, wrap(innerUnit))
		}
		return BundleUnitWrapperRlp{Inner: &layerRlp}
	default:
		panic(fmt.Sprintf("invalid type: %T", unit))
	}
}

func wrapAll(units []BundleUnit) []BundleUnitWrapperRlp {
	wrapped := make([]BundleUnitWrapperRlp, len(units))
	for i, unit := range units {
		wrapped[i] = wrap(unit)
	}
	return wrapped
}

func unwrapAll(wrapped []BundleUnitWrapperRlp) []BundleUnit {
	unwrapped := make([]BundleUnit, len(wrapped))
	for i, w := range wrapped {
		switch u := w.Inner.(type) {
		case *BundleTransaction:
			unwrapped[i] = u
		case *BundleLayerRlp:
			unwrapped[i] = &BundleLayer{Flags: u.Flags, Units: unwrapAll(u.Units)}
		default:
			panic(fmt.Sprintf("invalid type: %T", w.Inner))
		}
	}
	return unwrapped
}
