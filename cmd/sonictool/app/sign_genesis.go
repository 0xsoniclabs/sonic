package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/genesis"
	ogenesis "github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/prompt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
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

	signatureString, err := prompt.UserPrompt.PromptInput("Signature (hex): ")
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

func getGenesisHeaderHashes(genesisFile string) (header ogenesis.Header, genesisHashes ogenesis.Hashes, err error) {
	genesisReader, err := os.Open(genesisFile)
	// note, genesisStore closes the reader, no need to defer close it here
	if err != nil {
		return ogenesis.Header{}, nil, fmt.Errorf("failed to open genesis file: %w", err)
	}

	genesisStore, genesisHashes, err := genesisstore.OpenGenesisStore(genesisReader)
	if err != nil {
		return ogenesis.Header{}, nil, errors.Join(
			fmt.Errorf("failed to read genesis file: %w", err),
			utils.AnnotateIfError(genesisReader.Close(), "failed to close the genesis file"))
	}
	defer caution.CloseAndReportError(&err, genesisStore, "failed to close the genesis store")
	return genesisStore.Header(), genesisHashes, nil
}
