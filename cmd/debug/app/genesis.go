package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/cmd/debug/app/modules/mock"
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
)

func resolvePath(output string) (string, error) {
	if filepath.IsAbs(output) {
		return output, nil
	}
	return filepath.Abs(output)
}

type assetJSON struct {
	Module string          `fieldNumber:"1" json:"module"`
	Data   json.RawMessage `fieldNumber:"2" json:"data"`
}

type assetsJSON struct {
	Assets []*assetJSON `json:"assets"`
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
				Name:  "height",
				Usage: "Height of the genesis block",
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
			assets := &assetsJSON{}
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
			blockAssets := blockchain.BlockAssets{}
			// specific to mock implementation
			if len(assets.Assets) > 0 && assets.Assets[0].Module == "mock" {
				data := &mock.ValidatorsData{}
				if err := json.Unmarshal(assets.Assets[0].Data, data); err != nil {
					return err
				}
				mockAsset := blockchain.BlockAsset{
					Module: "mock",
					Data:   data.Encode(),
				}
				blockAssets = append(blockAssets, &mockAsset)
			}

			genesis, err := app.GenerateGenesisBlock(uint32(c.Int("height")), timestamp, previousBlockID, blockAssets)
			if err != nil {
				return err
			}

			output := c.String("output")
			if output == "" {
				genesisJSON, err := json.MarshalIndent(genesis, " ", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(genesisJSON))
				return nil
			}
			encodedGenesis := genesis.Encode()

			if err := os.WriteFile(output, encodedGenesis, 0755); err != nil { //nolint:gosec // ok to write as 0755
				return err
			}

			return nil
		},
	}
}
