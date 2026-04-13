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

package sccapi

import (
	"iter"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/result"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=sccapi

// SccApiBackend is the backend interface for the Sonic Certification Chain API.
// An implementation thereof provides access to the Sonic Certification Chain.
type SccApiBackend interface {
	GetLatestCommitteeCertificate() (cert.CommitteeCertificate, error)
	EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[cert.CommitteeCertificate]]

	GetLatestBlockCertificate() (cert.BlockCertificate, error)
	EnumerateBlockCertificates(first idx.Block) iter.Seq[result.T[cert.BlockCertificate]]
}
