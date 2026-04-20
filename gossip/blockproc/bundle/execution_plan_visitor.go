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

import "errors"

//go:generate mockgen -source=execution_plan_visitor.go -destination=execution_plan_visitor_mock.go -package=bundle

// ExecutionPlanVisitor is the visitor interface for traversing the execution plan.
// It allows keeping the internal implementation of the execution plan hidden
// from users.
type ExecutionPlanVisitor interface {
	Step(flags ExecutionFlags, txRef TxReference) error
	BeginGroup(oneOf bool, tolerateFailed bool)
	EndGroup()
}

// Accept accepts a visitor and traverses the execution plan,
// calling the appropriate methods on the visitor.
func (s *ExecutionStep) Accept(visitor ExecutionPlanVisitor) error {

	if !s.valid() {
		return errors.New("invalid execution plan")
	}

	var err error
	if s.single != nil {
		return visitor.Step(s.single.flags, s.single.txRef)
	} else if s.group != nil {
		visitor.BeginGroup(s.group.oneOf, s.group.tolerateFailed)
		for _, subStep := range s.group.steps {
			err = subStep.Accept(visitor)
			if err != nil {
				break
			}
		}
		visitor.EndGroup()
	}

	return err
}
