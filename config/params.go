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

package config

var (
	Bootnodes = map[string][]string{
		"sonic": {
			"enode://5facb14cfb4c5cb63f916f46e9689619f65040aa8efb7e9e9adb506db9fc006f042828ad7896467cf5eb1f3b74e53ad4fcd6bc4f15ae3acc3e8b7628e5446dad@boot-a.sonic.soniclabs.com:5050",
			"enode://04d51b2e172ed3722267bf26a39b55645ea220df7d35be7581a96409567cab5807eadd4c0b94f2c0c7128adece5ae64dd808dd9aa4543aa05c789bca02099731@boot-b.sonic.soniclabs.com:5050",
			"enode://ae3a88445aafb8cfff48bbcb1e7399c49d5d00d7c859f6dc0d3f8f2386019481f49cf1c32f2bce5a234de1311ebd8162858a4aa4c9dcc7f99b9b0c770044e7d5@boot-c.sonic.soniclabs.com:5050",
			"enode://979d4ed2a093324cafb413bd890f9e61d5ed1838215fe1704994e88d74e6067c294e03efd17e1186d8cafb70ca1ab05709765b1eb373928d7736319216cb97d0@boot-d.sonic.soniclabs.com:5050",
			"enode://22002ecd942431f5ba13c6873c32e5af5901eca9af170e263592a43e18dda774d7f986e6fd1e57c3f97d7566d7f713da30ed59746b4c52448e1e8c8fbcc7de5a@boot-e.sonic.soniclabs.com:5050",
			"enode://d85734f1fa2415ceffe56ae55f3f446fd722e3a5189af5753e4db3947efea2d11637912c010ab67d884cc8859f76bfd1e9ea47b613fff61f746b3aa0e0237837@boot-f.sonic.soniclabs.com:5050",
			"enode://9403ff58d782635f5a82974a774efdfa10ca15125c5c146426485dbe68305d52a39ae2af8739175b11cca7b3f42746ea9c51d1c19969cfd5ef96ec24b0fce093@boot-g.sonic.soniclabs.com:5050",
			"enode://e03d93283100cec25105da49bdaf34e38d5ae9ddcbc54114c9095471f2e9b3f6f1d190ef4cc5acdb12da4131722f9948c9af43ab64c6f0d8b7d05694cfc3c5b4@boot-h.sonic.soniclabs.com:5050",
			"enode://e6bb6a491256d67f6731a9b03af0426965449f205343605ad4ff075448a88624c30a0274aef29d9d44ce1bd94e2bec6fc158e076422545a20dbf6e272f1188c7@boot-i.sonic.soniclabs.com:5050",
			"enode://ecf6bc22d0293db8a1323dc525b69b3a5648692da6d64c612cacbec47b6139e18acf49a7edfb8298b7e4df9182b0da271c63a7d53f293fa9d035f2242bf554e2@boot-j.sonic.soniclabs.com:5050",
			"enode://06f542cb2377feb39d544cb7d6160ddcb682d4a93b8fbb15137b75558295cc31dfe8862b233d26c32a377a71d13539befeadb5164efca365de5743812031d467@boot-k.sonic.soniclabs.com:5050",
			"enode://81616eadbe45cf147ddc87dabd44a326cc9c3c78ef446bceca42380bc5ac4260b8f52a1369a444d67c80896232916dbdab732cb9779c86d43af9142a5befcc85@boot-l.sonic.soniclabs.com:5050",
		},
		"blaze-testnet": {
			"enode://8afb7207ac6448871f9d165b59c8392fd1341b6cd89fc031e383b075326c31ca9399a058314ef3265a5000325b67b567502946496c4a94ae5354a0db646d3350@bootstrap-a.blaze.soniclabs.com:5050",
			"enode://6ed780d0f72e3e67c863cc26090040a56579bc989a3597fc88b045a65653633acddc77907635bc4128ad48e9c067ac0b4da406f3b32c92b2d61f79f99978cd40@bootstrap-b.blaze.soniclabs.com:5050",
			"enode://4f3a18cc1321092a8e14b50fe01817c95804835b5f9c0d685664ccf65f81ebc6f5fa1a321701c5bbdd17369d7a61a1a67c47f5d9ec22fb04271da4e78946552c@bootstrap-c.blaze.soniclabs.com:5050",
			"enode://4242ba2a88bbf51e10ea331b5bf44f14b507b439fe2047d0721445e19bd091a4c6f4fe5733a8925ab7c87ac5fdeb03b65c07ec2f1573f5840bc388e081c97b45@bootstrap-d.blaze.soniclabs.com:5050",
		},
	}
)
