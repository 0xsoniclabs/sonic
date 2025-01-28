package app

import (
	"fmt"
	"github.com/0xsoniclabs/sonic/cmd/sonictool/genesis"
	ogenesis "github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
	"os"
)

func signGenesis(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		return fmt.Errorf("this command requires an argument - the genesis file to import")
	}

	header, genesisHashes, err := getGenesisHeaderHashes(ctx.Args().First())
	if err != nil {
		return err
	}

	for sectionName, sectionHash := range genesisHashes {
		log.Info("Section", "name", sectionName, "hash", hexutil.Encode(sectionHash.Bytes()))
	}
	if _, ok := genesisHashes["signature"]; ok {
		return fmt.Errorf("genesis file is already signed")
	}

	hash, rawData, err := genesis.GetGenesisMetadata(header, genesisHashes)
	if err != nil {
		return err
	}

	log.Info("Hash to sign", "hash", hexutil.Encode(hash))
	log.Info("Raw data", "rawdata", hexutil.Encode([]byte(rawData)))

	fmt.Printf("Signature (hex): ")
	var signatureString string
	_, err = fmt.Scanln(&signatureString)
	if err != nil {
		return fmt.Errorf("failed to read signature: %w", err)
	}
	signature, err := hexutil.Decode(signatureString)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if err := genesis.CheckGenesisSignature(hash, signature); err != nil {
		return err
	}
	if err = genesis.WriteSignatureIntoGenesisFile(header, signature, ctx.Args().First()); err != nil {
		return fmt.Errorf("failed to write signature into genesis file: %w", err)
	}
	log.Info("Signature successfully written into genesis file")
	return nil
}

func getGenesisHeaderHashes(genesisFile string) (ogenesis.Header, ogenesis.Hashes, error) {
	genesisReader, err := os.Open(genesisFile)
	if err != nil {
		return ogenesis.Header{}, nil, fmt.Errorf("failed to open the genesis file: %w", err)
	}
	defer genesisReader.Close()

	genesisStore, genesisHashes, err := genesisstore.OpenGenesisStore(genesisReader)
	if err != nil {
		return ogenesis.Header{}, nil, fmt.Errorf("failed to read genesis file: %w", err)
	}
	defer genesisStore.Close()

	return genesisStore.Header(), genesisHashes, nil
}
