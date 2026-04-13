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

package sonicapi

import (
	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=sonicapi

// BundleApiBackend is an interface that abstracts the necessary BundleApiBackend functions for the
// sonic API.
type BundleApiBackend interface {
	ethapi.Backend

	// GetBundleExecutionInfo returns information about the execution of a transaction bundle, if available.
	// If the bundle is unknown, it returns nil.
	GetBundleExecutionInfo(common.Hash) *bundle.ExecutionInfo
}
