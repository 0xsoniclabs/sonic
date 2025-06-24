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

package genesis

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func GetGenesisMetadata(header genesis.Header, genesisHashes genesis.Hashes) ([]byte, string, error) {
	// add section hashes in deterministic order
	sectionNames := make(sort.StringSlice, 0, len(genesisHashes))
	for sectionName := range genesisHashes {
		if sectionName == "signature" {
			continue
		}
		sectionNames = append(sectionNames, sectionName)
	}
	sectionNames.Sort()

	var sections = make([]interface{}, 0)
	for _, sectionName := range sectionNames {
		sections = append(sections, map[string]interface{}{
			"name": sectionName,
			"hash": genesisHashes[sectionName].Bytes(),
		})
	}

	eip712data := apitypes.TypedData{
		Types: map[string][]apitypes.Type{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"Genesis": {
				{Name: "sections", Type: "Section[]"},
			},
			"Section": {
				{Name: "name", Type: "string"},
				{Name: "hash", Type: "bytes"},
			},
		},
		PrimaryType: "Genesis",
		Domain: apitypes.TypedDataDomain{
			Name:    header.NetworkName,
			Version: header.GenesisID.String(),
			ChainId: math.NewHexOrDecimal256(int64(header.NetworkID)),
		},
		Message: map[string]interface{}{
			"sections": sections,
		},
	}

	return TypedDataAndHash(eip712data)
}

func CheckGenesisSignature(hash []byte, signature []byte) error {
	if len(signature) != 65 {
		return fmt.Errorf("invalid signature length")
	}
	// If V is on 27/28-form, convert to 0/1
	if signature[64] == 27 || signature[64] == 28 {
		signature[64] -= 27
	}
	recoveredPubKey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return err
	}
	address := crypto.PubkeyToAddress(*recoveredPubKey)
	for _, allowedSigner := range allowedGenesisSigners {
		if address == allowedSigner {
			return nil
		}
	}
	return fmt.Errorf("genesis signature does not match any trusted signer (signer: %x)", address)
}

func WriteSignatureIntoGenesisFile(header genesis.Header, signature []byte, file string) (err error) {
	out, err := os.OpenFile(file, os.O_RDWR, os.ModePerm) // avoid using O_APPEND for correct seek positions
	if err != nil {
		return fmt.Errorf("failed to open genesis file: %w", err)
	}
	_, err = out.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek genesis file: %w", err)
	}
	defer caution.CloseAndReportError(&err, out, "failed to close genesis file")

	tmpDir, err := os.MkdirTemp("", "signing-genesis-tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer caution.ExecuteAndReportError(&err, func() error { return os.RemoveAll(tmpDir) },
		"failed to remove temporary directory")

	writer := newUnitWriter(out)
	if err = writer.Start(header, "signature", tmpDir); err != nil {
		return fmt.Errorf("failed to write start to genesis file: %w", err)
	}
	_, err = writer.Write(signature)
	if err != nil {
		return fmt.Errorf("failed to write signature to genesis file: %w", err)
	}
	_, err = writer.Flush()
	return utils.AnnotateIfError(err, "failed to flush genesis file:")
}

// TypedDataAndHash is a helper function that calculates a hash for typed data conforming to EIP-712.
// This hash can then be safely used to calculate a signature.
//
// See https://eips.ethereum.org/EIPS/eip-712 for the full specification.
//
// This gives context to the signed typed data and prevents signing of transactions.
func TypedDataAndHash(typedData apitypes.TypedData) ([]byte, string, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, "", err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, "", err
	}
	rawData := fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash))
	return crypto.Keccak256([]byte(rawData)), rawData, nil
}
