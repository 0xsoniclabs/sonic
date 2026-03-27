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

package parentscheck

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/inter"
)

func TestValidate_NoParents(t *testing.T) {
	c := New()

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(1)
	me.SetSeq(1)
	me.SetLamport(1)
	me.SetCreationTime(1000)
	e := me.Build()

	// No parents, no self-parent - should pass.
	err := c.Validate(e, inter.EventIs{})
	if err != nil {
		t.Fatalf("expected no error for event without parents, got %v", err)
	}
}

func TestValidate_SelfParent_PastTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	_ = ctrl

	// Create a self-parent event.
	parentMe := &inter.MutableEventPayload{}
	parentMe.SetEpoch(1)
	parentMe.SetCreator(1)
	parentMe.SetSeq(1)
	parentMe.SetLamport(1)
	parentMe.SetCreationTime(2000) // parent creation time = 2000
	parent := parentMe.Build()

	// Create a child event with LOWER creation time.
	childMe := &inter.MutableEventPayload{}
	childMe.SetEpoch(1)
	childMe.SetCreator(1)
	childMe.SetSeq(2)
	childMe.SetLamport(2)
	childMe.SetCreationTime(1000) // lower than parent's 2000
	childMe.SetParents(hash.Events{parent.ID()})
	child := childMe.Build()

	c := New()
	err := c.Validate(child, inter.EventIs{parent})
	if err != ErrPastTime {
		t.Fatalf("expected ErrPastTime, got %v", err)
	}
}

func TestValidate_SelfParent_ValidTime(t *testing.T) {
	// Create a self-parent event.
	parentMe := &inter.MutableEventPayload{}
	parentMe.SetEpoch(1)
	parentMe.SetCreator(1)
	parentMe.SetSeq(1)
	parentMe.SetLamport(1)
	parentMe.SetCreationTime(1000)
	parent := parentMe.Build()

	// Create a child with HIGHER creation time.
	childMe := &inter.MutableEventPayload{}
	childMe.SetEpoch(1)
	childMe.SetCreator(1)
	childMe.SetSeq(2)
	childMe.SetLamport(2)
	childMe.SetCreationTime(2000) // higher than parent's 1000
	childMe.SetParents(hash.Events{parent.ID()})
	child := childMe.Build()

	c := New()
	err := c.Validate(child, inter.EventIs{parent})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_MultipleParents(t *testing.T) {
	// Self-parent.
	parentMe := &inter.MutableEventPayload{}
	parentMe.SetEpoch(1)
	parentMe.SetCreator(1)
	parentMe.SetSeq(1)
	parentMe.SetLamport(1)
	parentMe.SetCreationTime(1000)
	selfParent := parentMe.Build()

	// Other parent from a different creator.
	otherMe := &inter.MutableEventPayload{}
	otherMe.SetEpoch(1)
	otherMe.SetCreator(idx.ValidatorID(2))
	otherMe.SetSeq(1)
	otherMe.SetLamport(2)
	otherMe.SetCreationTime(1500)
	otherParent := otherMe.Build()

	// Child with self-parent + other parent.
	// Lamport must be max(parent Lamports) + 1 = max(1,2) + 1 = 3.
	childMe := &inter.MutableEventPayload{}
	childMe.SetEpoch(1)
	childMe.SetCreator(1)
	childMe.SetSeq(2)
	childMe.SetLamport(3)
	childMe.SetCreationTime(2000)
	childMe.SetParents(hash.Events{selfParent.ID(), otherParent.ID()})
	child := childMe.Build()

	c := New()
	err := c.Validate(child, inter.EventIs{selfParent, otherParent})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
