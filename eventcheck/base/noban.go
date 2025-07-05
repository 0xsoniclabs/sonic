// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package eventcheck

import (
	"errors"
)

var (
	ErrAlreadyConnectedEvent = errors.New("event is connected already")
	ErrSpilledEvent          = errors.New("event is spilled")
	ErrDuplicateEvent        = errors.New("event is duplicated")
)
