package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
	"github.com/LiskHQ/lisk-engine/pkg/framework/preset"
)

type generator struct {
	Address             codec.Lisk32 `json:"address"`
	EncryptedPassphrase string       `json:"encryptedPassphrase"`
}

type hashOnion struct {
	Address codec.Lisk32 `json:"address"`
}

type AccountJSON struct {
	Accounts   []*keys      `json:"accounts"`
	Generators []*generator `json:"generators"`
	HashOnions []*hashOnion `json:"hashOnions"`
}

func getDefaultGenesisBlock() (*blockchain.Block, error) {
	// Generate genesis block
	config := &config.ApplicationConfig{}
	if err := config.InsertDefault(); err != nil {
		return nil, err
	}
	presetApp := preset.NewPresetApplication(config)
	genesisBlock, err := presetApp.App.GenerateGenesisBlock(0, uint32(time.Now().Unix()), []byte{}, blockchain.BlockAssets{})
	if err != nil {
		return nil, err
	}
	// Create additional files

	return genesisBlock, nil
}

type assetJSON struct {
	Assets blockchain.BlockAssets `json:"assets"`
}

func GetGenesisCommand(starter Starter) *cli.Command {
	return &cli.Command{
		Name:  "genesis",
		Usage: "Start blockchain application",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "File path to config to use for generating genesis block",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "asset",
				Aliases:  []string{"a"},
				Usage:    "File path to block asset to include",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "height",
				Aliases: []string{"h"},
				Usage:   "Height of the genesis block",
			},
			&cli.IntFlag{
				Name:    "timestamp",
				Aliases: []string{"t"},
				Usage:   "Timestamp of the genesis block",
			},
			&cli.StringFlag{
				Name:    "previous-block-id",
				Aliases: []string{"p"},
				Usage:   "PreviousBlockID of the genesis block",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Specify output directory",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")
			resolvedConfigPath, err := resolvePath(configPath)
			if err != nil {
				return err
			}
			configFile, err := os.ReadFile(resolvedConfigPath)
			if err != nil {
				return err
			}
			config := &config.ApplicationConfig{}
			if err := json.Unmarshal(configFile, config); err != nil {
				return err
			}
			assetPath := c.String("asset")
			resolvedAssetPath, err := resolvePath(assetPath)
			if err != nil {
				return err
			}
			assetsFile, err := os.ReadFile(resolvedAssetPath)
			if err != nil {
				return err
			}
			assets := &assetJSON{}
			if err := json.Unmarshal(assetsFile, assets); err != nil {
				return err
			}
			app, err := starter.GetApplication(config)
			if err != nil {
				return err
			}
			timestamp := uint32(c.Int("timestamp"))
			if timestamp == 0 {
				timestamp = uint32(time.Now().Unix())
			}
			previousBlockIDStr := c.String("previous-block-id")
			previousBlockID := []byte{}
			if previousBlockIDStr != "" {
				previousBlockID, err = hex.DecodeString(previousBlockIDStr)
				if err != nil {
					return err
				}
			}

			genesis, err := app.GenerateGenesisBlock(uint32(c.Int("height")), timestamp, previousBlockID, assets.Assets)
			if err != nil {
				return err
			}

			output := c.String("output")
			if output == "" {
				genesisJSON, err := json.MarshalIndent(genesis, " ", "  ")
				if err != nil {
					return err
				}
				fmt.Println(genesisJSON)
				return nil
			}
			encodedGenesis, err := genesis.Encode()
			if err != nil {
				return err
			}

			if err := os.WriteFile(output, encodedGenesis, 0755); err != nil { //nolint:gosec // ok to write as 0755
				return err
			}

			return nil
		},
	}
}
