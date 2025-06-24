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

package encryption

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

type encryptedAccountKeyJSONV3 struct {
	Address string              `json:"address"`
	Crypto  keystore.CryptoJSON `json:"crypto"`
	Id      string              `json:"id"`
	Version int                 `json:"version"`
}

func MigrateAccountToValidatorKey(acckeypath string, valkeypath string, pubkey validatorpk.PubKey) error {
	acckeyjson, err := os.ReadFile(acckeypath)
	if err != nil {
		return err
	}
	acck := new(encryptedAccountKeyJSONV3)
	if err := json.Unmarshal(acckeyjson, acck); err != nil {
		return err
	}

	valk := EncryptedKeyJSON{
		Type:      validatorpk.Types.Secp256k1,
		PublicKey: common.Bytes2Hex(pubkey.Raw),
		Crypto:    acck.Crypto,
	}
	valkeyjson, err := json.Marshal(valk)
	if err != nil {
		return err
	}
	tmpName, err := writeTemporaryKeyFile(valkeypath, valkeyjson)
	if err != nil {
		return err
	}
	return os.Rename(tmpName, valkeypath)
}
