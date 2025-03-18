package genesis

import (
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
)

type GenesisTemplate struct {
	Name   string
	Header genesis.Header
	Hashes genesis.Hashes
}

var (
	allowedGenesisSigners = []common.Address{
		common.HexToAddress("0xCe5409bE69D1116FEa622be6Fd64475FB4D3bf3e"), // HW
	}

	mainnetHeader = genesis.Header{
		GenesisID:   hash.HexToHash("0x4a53c5445584b3bfc20dbfb2ec18ae20037c716f3ba2d9e1da768a9deca17cb4"),
		NetworkID:   opera.MainNetworkID,
		NetworkName: "main",
	}

	testnetHeader = genesis.Header{
		GenesisID:   hash.HexToHash("0xc4a5fc96e575a16a9a0c7349d44dc4d0f602a54e0a8543360c2fee4c3937b49e"),
		NetworkID:   opera.TestNetworkID,
		NetworkName: "test",
	}

	allowedGenesis = []GenesisTemplate{
		{
			Name:   "Mainnet-5577 with pruned MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x945d8084b4e6e1e78cfe9472fefca3f6ecc7041765dfed24f64e9946252f569a"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xe3ec041f3cca79928aa4abef588b48e96ff3cfa3908b2268af3ac5496c722fec"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x12dd52ac21fee5d76b47a64386e73187d5260e448e8044f38c6c73eaa627e4b5"),
			},
		},
		{
			Name:   "Mainnet-5577 with full MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x945d8084b4e6e1e78cfe9472fefca3f6ecc7041765dfed24f64e9946252f569a"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xe3ec041f3cca79928aa4abef588b48e96ff3cfa3908b2268af3ac5496c722fec"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x54614c9475963ed706f3e654bee0faf9ca21e29c588ad4070fd5b5897c8e0b5d"),
			},
		},
		{
			Name:   "Mainnet-5577 with full MPT for Sonic",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):     hash.HexToHash("0x945d8084b4e6e1e78cfe9472fefca3f6ecc7041765dfed24f64e9946252f569a"),
				genesisstore.BlocksSection(0):     hash.HexToHash("0xe3ec041f3cca79928aa4abef588b48e96ff3cfa3908b2268af3ac5496c722fec"),
				genesisstore.FwsLiveSection(0):    hash.HexToHash("0x7af5e0ec141aa21c797be39b5a515ca8160ec8a4d6c0e181e34a2708c171908a"),
				genesisstore.FwsArchiveSection(0): hash.HexToHash("0x6da7139b20139732ec290f47faf6c7f3a6439edcbae3c1c0fcc492cc6ede634f"),
			},
		},
		{
			Name:   "Mainnet-109331 without history",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0xf0bef59de85dde7772bf43c62267eb312b0dd2c412ec5f04d96b6ea55178e901"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x80fb348f77f65f0c357f69e29ca123a4c8f1ba60ff445510474be952a1e28d7a"),
			},
		},
		{
			Name:   "Mainnet-109331 without MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x71cdb819c2745a4853016bbb9690053b70fac679b168cd9a4999bf2a3dfb5578"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xb7394f84b73528423a5b634bfb3cec8ab0a015b387bf6cbe70b378b08e9253bd"),
			},
		},
		{
			Name:   "Mainnet-109331 with pruned MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x71cdb819c2745a4853016bbb9690053b70fac679b168cd9a4999bf2a3dfb5578"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xb7394f84b73528423a5b634bfb3cec8ab0a015b387bf6cbe70b378b08e9253bd"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x617b8c4d74d1598f7d3914ba4c7cd46b7c98d5e044987c6c8d023cc59e849df7"),
			},
		},
		{
			Name:   "Mainnet-109331 with full MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x71cdb819c2745a4853016bbb9690053b70fac679b168cd9a4999bf2a3dfb5578"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xb7394f84b73528423a5b634bfb3cec8ab0a015b387bf6cbe70b378b08e9253bd"),
				genesisstore.EvmSection(0):    hash.HexToHash("0xef0e1b833321a8de98aaaa1a3946378c78d66ab16b39eb0ad56636d5f7f9f2c5"),
			},
		},

		{
			Name:   "Mainnet-171200 without history",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0xb1954a0ac01a35a9ff6026d239e0be659fb4c37c356a94318bbe9201b9b2f3bf"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x48ea3ccd2e2ff819386aa6d5ba86b12abb92f2f2e3b405e903f71e3f33f1d258"),
			},
		},
		{
			Name:   "Mainnet-171200 without MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x71cdb819c2745a4853016bbb9690053b70fac679b168cd9a4999bf2a3dfb5578"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xb7394f84b73528423a5b634bfb3cec8ab0a015b387bf6cbe70b378b08e9253bd"),
				genesisstore.EpochsSection(1): hash.HexToHash("0xda430371772ee2fefd1caa342b6a5cb188041a01730f681099dd241bc57a3f77"),
				genesisstore.BlocksSection(1): hash.HexToHash("0x14b8b9c3b47cc174ae5c36599cebdef551ad35032ed29c087abb814ac5559619"),
			},
		},
		{
			Name:   "Mainnet-171200 with pruned MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x71cdb819c2745a4853016bbb9690053b70fac679b168cd9a4999bf2a3dfb5578"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xb7394f84b73528423a5b634bfb3cec8ab0a015b387bf6cbe70b378b08e9253bd"),
				genesisstore.EpochsSection(1): hash.HexToHash("0xda430371772ee2fefd1caa342b6a5cb188041a01730f681099dd241bc57a3f77"),
				genesisstore.BlocksSection(1): hash.HexToHash("0x14b8b9c3b47cc174ae5c36599cebdef551ad35032ed29c087abb814ac5559619"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x2a685df416eeca50f4b725117ae88deb35f05e3c51f34e9555ff6ffc62e75d14"),
			},
		},
		{
			Name:   "Mainnet-279701 with Carmen live state only",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):  hash.HexToHash("0x6a685f5b446eb17cc69047ddd230c0ccc0c820d4ba05bfdd30aa6176b40618da"),
				genesisstore.BlocksSection(0):  hash.HexToHash("0xdf6ed841b928fad8632b7e532f0b989d93e26b8332fe8429504822df6d44a642"),
				genesisstore.FwsLiveSection(0): hash.HexToHash("0x702c987a8e799d7550db6a3fc9a571cbaeac7a00d7bb984a00374a27bd25d908"),
			},
		},
		{
			Name:   "Mainnet-279701 with Carmen live and archive state",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):     hash.HexToHash("0x6a685f5b446eb17cc69047ddd230c0ccc0c820d4ba05bfdd30aa6176b40618da"),
				genesisstore.BlocksSection(0):     hash.HexToHash("0xdf6ed841b928fad8632b7e532f0b989d93e26b8332fe8429504822df6d44a642"),
				genesisstore.FwsLiveSection(0):    hash.HexToHash("0x702c987a8e799d7550db6a3fc9a571cbaeac7a00d7bb984a00374a27bd25d908"),
				genesisstore.FwsArchiveSection(0): hash.HexToHash("0xf445000720ef2969aa0fb4db6f5542452b5ea83fbf54dc1dcbe9202af0feafd8"),
			},
		},
		{
			Name:   "Mainnet-282500 with Carmen live state only",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):  hash.HexToHash("0x46ed30acff8b17680e73e78bcc1c1ece2b0288fecc8fce0e01bd058fa2d34c2b"),
				genesisstore.BlocksSection(0):  hash.HexToHash("0x2bd68ede30496bc53a57f53f2a47e22a4d0c6ae21168717078bb09c2e09e9b10"),
				genesisstore.FwsLiveSection(0): hash.HexToHash("0x47025ee3895fe976750bdd883f84ef451a5fe1051d4d2ca294bebf63eb6555c6"),
			},
		},
		{
			Name:   "Mainnet-282500 with Carmen live and archive state",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):     hash.HexToHash("0x46ed30acff8b17680e73e78bcc1c1ece2b0288fecc8fce0e01bd058fa2d34c2b"),
				genesisstore.BlocksSection(0):     hash.HexToHash("0x2bd68ede30496bc53a57f53f2a47e22a4d0c6ae21168717078bb09c2e09e9b10"),
				genesisstore.FwsLiveSection(0):    hash.HexToHash("0x47025ee3895fe976750bdd883f84ef451a5fe1051d4d2ca294bebf63eb6555c6"),
				genesisstore.FwsArchiveSection(0): hash.HexToHash("0x7d8c63e5a080fd53daa991aa30d5b990d421c199d7c3c398982a73ddc59b1541"),
			},
		},
		{
			Name:   "Mainnet-283890 with Carmen live state only and the last epoch blocks only",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):  hash.HexToHash("0x776ce5900d2e8e2f088ff4eb2cfb9ee4632a0d9f0378bc8fd21da0d9d49b9272"),
				genesisstore.BlocksSection(0):  hash.HexToHash("0x3f6c530995c5e7c506d9106d20d69f94fa79aa68a24b426e3f0653c64d1100b7"),
				genesisstore.FwsLiveSection(0): hash.HexToHash("0xf8b42be75150cd76b5f86c8c4d67a9aaae90332d0be648df4f93e0bc4830d35e"),
			},
		},
		{
			Name:   "Mainnet-285300 with Carmen live state only and the last epoch blocks only",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):  hash.HexToHash("0x2027d90dc318ace70a3d0cfc22a69cfa31cf18f874f7ea15d34396dc8fdb11d1"),
				genesisstore.BlocksSection(0):  hash.HexToHash("0x99ff07d5b8423215304821151c2490ef9ac9abdb030d58823ffe656586ee3af3"),
				genesisstore.FwsLiveSection(0): hash.HexToHash("0x77ea7b0496b026981857b99c48b03e26c5071c010b5f0ff5536db8492ef27257"),
			},
		},
		{
			Name:   "Mainnet-285300 with Carmen live and archive state",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):     hash.HexToHash("0x2027d90dc318ace70a3d0cfc22a69cfa31cf18f874f7ea15d34396dc8fdb11d1"),
				genesisstore.BlocksSection(0):     hash.HexToHash("0xfa7418c8d8bc4738a0cdb32c87feee75453c6fa580a8568dc0eed3e65b03ae90"),
				genesisstore.FwsLiveSection(0):    hash.HexToHash("0xfe13c9d30c72c5d6d9f381edb5877c3a248bcba90a302b236fb22c5f52f4b953"),
				genesisstore.FwsArchiveSection(0): hash.HexToHash("0xa0c1d01a0bc88470ce4b910470e270bf80e4f13ff8e6ac4e60df0f9281e627c3"),
			},
		},
		{
			Name:   "Mainnet-286540 with Carmen live state only and the last epoch blocks only",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):  hash.HexToHash("0x9c88095cbb4b222d603bddf7f4d222c630b43742dba7f519e3168d09b1a54d56"),
				genesisstore.BlocksSection(0):  hash.HexToHash("0xbe6217623a596de6601e1f780036faf5ea13eea83a994c9e0380ff2dfe773db5"),
				genesisstore.FwsLiveSection(0): hash.HexToHash("0x4df8b57c5cfdbcd6393ce5ca8ebfe80e36a5b51f2e0b9539dcf6d34578911fdc"),
			},
		},
		{
			Name:   "Mainnet-288000 with Carmen live state only and the last epoch blocks only",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):  hash.HexToHash("0xeb9d17cca2be7b9a95043d51dc2c71b7c1704bee20162561722806a78f27c5a4"),
				genesisstore.BlocksSection(0):  hash.HexToHash("0xb48e5335406a68b9510e3c9e1884e8999e9a68a9ab98cd1d0d1b0d54e518058f"),
				genesisstore.FwsLiveSection(0): hash.HexToHash("0x857d914c5df198c833026d53014485a9053304909742c5a6a3544aee0001b9d7"),
			},
		},
		{
			Name:   "Mainnet-288000 with Carmen live and archive state",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0):     hash.HexToHash("0xeb9d17cca2be7b9a95043d51dc2c71b7c1704bee20162561722806a78f27c5a4"),
				genesisstore.BlocksSection(0):     hash.HexToHash("0x31158f12cba0e4b695ee744266cce626591764bd390b67177438426b0a0f5331"),
				genesisstore.FwsLiveSection(0):    hash.HexToHash("0x857d914c5df198c833026d53014485a9053304909742c5a6a3544aee0001b9d7"),
				genesisstore.FwsArchiveSection(0): hash.HexToHash("0x5c44b48107ee71a0a643d330f17bd0fd0e75b1fd466d6270c1cc6764b9d8b67b"),
			},
		},

		{
			Name:   "Testnet-2458 with pruned MPT",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x4a5caf86d7f5a31dad91f2cbd44db052c602f515e5319f828adb585a7a6723d6"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x07eadb81c1e2a1b5c444c8c2430c6873380f447de64790b25abe9e7fa6874f65"),
				genesisstore.EvmSection(0):    hash.HexToHash("0xa96e006ae17d15e1244c3e7ff4d556e5a3849e70df7a81704787f3273f37c9b1"),
			},
		},
		{
			Name:   "Testnet-2458 with full MPT",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x4a5caf86d7f5a31dad91f2cbd44db052c602f515e5319f828adb585a7a6723d6"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x07eadb81c1e2a1b5c444c8c2430c6873380f447de64790b25abe9e7fa6874f65"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x3c635232f82cabdfc76405fd03c58134fb00ff9fc0ad080a8a8ae40a7a6fe604"),
			},
		},
		{
			Name:   "Testnet-6226 without history",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x5527a0c5e45b84c1350ccc77a9644eb797afd1e87887c35526d7622b12881b22"),
				genesisstore.BlocksSection(0): hash.HexToHash("0xf209c98aa5d3473dd71164165152e8802fb95b71d9dbfe394a0addcf81808d5c"),
			},
		},
		{
			Name:   "Testnet-6226 without MPT",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x61040a80f16755b86d67f13880f55c1238d307e2e1c6fc87951eb3bdee0bdff2"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x12010621d8cf4dcd4ea357e98eac61edf9517a6df752cb2d929fca69e56bd8d1"),
			},
		},
		{
			Name:   "Testnet-6226 with pruned MPT",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x61040a80f16755b86d67f13880f55c1238d307e2e1c6fc87951eb3bdee0bdff2"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x12010621d8cf4dcd4ea357e98eac61edf9517a6df752cb2d929fca69e56bd8d1"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x86ec3c7938ab053fc84bbbc8f5259bc81885ec424df91272c553f371464840fc"),
			},
		},
		{
			Name:   "Testnet-6226 with full MPT",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x61040a80f16755b86d67f13880f55c1238d307e2e1c6fc87951eb3bdee0bdff2"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x12010621d8cf4dcd4ea357e98eac61edf9517a6df752cb2d929fca69e56bd8d1"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x9227c80bf56e4af08dc32cb6043cc43672f2be8177d550ab34a7a9f57f4f104b"),
			},
		},

		{
			Name:   "Testnet-16200 without history",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0xd72e9bf39c645df8d978955fab8997a7e9cd7cb5812c007e2bb4b51a8c570a90"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x7d651ed0e0f3e92ffd89cb52112598db54afd8bf3050bc083ff0bfe1b98948fd"),
			},
		},
		{
			Name:   "Testnet-16200 without MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x61040a80f16755b86d67f13880f55c1238d307e2e1c6fc87951eb3bdee0bdff2"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x12010621d8cf4dcd4ea357e98eac61edf9517a6df752cb2d929fca69e56bd8d1"),
				genesisstore.EpochsSection(1): hash.HexToHash("0xd72e9bf39c645df8d978955fab8997a7e9cd7cb5812c007e2bb4b51a8c570a90"),
				genesisstore.BlocksSection(1): hash.HexToHash("0x7d651ed0e0f3e92ffd89cb52112598db54afd8bf3050bc083ff0bfe1b98948fd"),
			},
		},
		{
			Name:   "Testnet-16200 with pruned MPT",
			Header: mainnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x61040a80f16755b86d67f13880f55c1238d307e2e1c6fc87951eb3bdee0bdff2"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x12010621d8cf4dcd4ea357e98eac61edf9517a6df752cb2d929fca69e56bd8d1"),
				genesisstore.EpochsSection(1): hash.HexToHash("0xd72e9bf39c645df8d978955fab8997a7e9cd7cb5812c007e2bb4b51a8c570a90"),
				genesisstore.BlocksSection(1): hash.HexToHash("0x7d651ed0e0f3e92ffd89cb52112598db54afd8bf3050bc083ff0bfe1b98948fd"),
				genesisstore.EvmSection(0):    hash.HexToHash("0xbd66dcbbe77881d5aae5091ee9c455d213cebef2cc53c0d4bb356840c7020f7b"),
			},
		},
		{
			Name:   "Testnet-16200 with full MPT",
			Header: testnetHeader,
			Hashes: genesis.Hashes{
				genesisstore.EpochsSection(0): hash.HexToHash("0x61040a80f16755b86d67f13880f55c1238d307e2e1c6fc87951eb3bdee0bdff2"),
				genesisstore.BlocksSection(0): hash.HexToHash("0x12010621d8cf4dcd4ea357e98eac61edf9517a6df752cb2d929fca69e56bd8d1"),
				genesisstore.EpochsSection(1): hash.HexToHash("0xd72e9bf39c645df8d978955fab8997a7e9cd7cb5812c007e2bb4b51a8c570a90"),
				genesisstore.BlocksSection(1): hash.HexToHash("0x7d651ed0e0f3e92ffd89cb52112598db54afd8bf3050bc083ff0bfe1b98948fd"),
				genesisstore.EvmSection(0):    hash.HexToHash("0x9227c80bf56e4af08dc32cb6043cc43672f2be8177d550ab34a7a9f57f4f104b"),
				genesisstore.EvmSection(1):    hash.HexToHash("0x2376016f7ba13123244c6b56088a76e2e8bd5d5795fa92bad65f61488d12c236"),
			},
		},
	}
)
