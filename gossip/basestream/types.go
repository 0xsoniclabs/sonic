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

package basestream

type Request struct {
	Session        Session
	Type           RequestType
	MaxPayloadNum  uint32
	MaxPayloadSize uint64
	MaxChunks      uint32
}

type Response struct {
	SessionID uint32
	Done      bool
	Payload   Payload
}

type Session struct {
	ID    uint32
	Start Locator
	Stop  Locator
}

type Locator interface {
	Compare(b Locator) int
	Inc() Locator
}

type Payload interface {
	Len() int
	TotalSize() uint64
	TotalMemSize() int
}

type RequestType uint8
