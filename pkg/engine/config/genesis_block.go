package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
)

func resolveTilda(p, homedir string) string {
	return strings.ReplaceAll(p, "~", homedir)
}

func ReadGenesisBlock(config *Config) (*blockchain.Block, error) {
	if len(config.Genesis.Block.Blob) != 0 {
		return blockchain.NewBlock(config.Genesis.Block.Blob)
	}
	if config.Genesis.Block.FromFile != "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		genesisBlockFilePath := resolveTilda(config.Genesis.Block.FromFile, homedir)
		if !filepath.IsAbs(genesisBlockFilePath) {
			var err error
			dataPath := resolveTilda(config.System.DataPath, homedir)
			genesisBlockFilePath, err = filepath.Abs(filepath.Join(dataPath, config.Genesis.Block.FromFile))
			if err != nil {
				return nil, err
			}
		}
		genesisBlockFile, err := os.ReadFile(genesisBlockFilePath)
		if err != nil {
			return nil, err
		}
		return blockchain.NewBlock(genesisBlockFile)
	}

	return nil, errors.New("genesis block config is empty")
}
