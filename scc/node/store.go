// Copyright 2025 Sonic Operations Ltd
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

package node

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=store.go -destination=store_mock.go -package=node

// Store is an interface for a persistent store of certificates. A store is
// required to facilitate the creation of certificates and the gradual
// aggregation of signatures.
type Store interface {
	// GetCommitteeCertificate retrieves the certificate for the given period.
	// If no certificate is found, an error is returned.
	GetCommitteeCertificate(scc.Period) (cert.CommitteeCertificate, error)

	// GetBlockCertificate retrieves the certificate for the given block.
	// If no certificate is found, an error is returned.
	GetBlockCertificate(idx.Block) (cert.BlockCertificate, error)

	// UpdateCommitteeCertificate adds or updates the certificate in the store.
	// If a certificate for the same period is already present, it is overwritten.
	UpdateCommitteeCertificate(cert.CommitteeCertificate) error

	// UpdateBlockCertificate adds or updates the certificate in the store.
	// If a certificate for the same block is already present, it is overwritten.
	UpdateBlockCertificate(cert.BlockCertificate) error
}
