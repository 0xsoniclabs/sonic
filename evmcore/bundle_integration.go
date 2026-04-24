// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

// bundlePoolStatus represents the status of a bundle in the transaction pool,
// which can be pending, queued, or rejected. This type is used to
// manage transitions from queued to pending, pending to queued, queued to
// rejected, and pending to rejected, based on the evaluation of the bundle's
// executability.
type bundlePoolStatus int

const (
	// bundlePending, the bundle hasn't yet been executed and trial-run is positive
	bundlePending bundlePoolStatus = iota
	// bundleQueued, the bundle hasn't yet been executed but trial-run is negative
	bundleQueued
	// bundleRejected, the bundle has been executed, or is not valid
	bundleRejected
)
