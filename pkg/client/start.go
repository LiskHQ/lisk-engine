package client

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
)

type Application interface {
	Start() error
	Stop() error
	GenerateGenesisBlock(height, timestamp uint32, previoudBlockID codec.Hex, assets blockchain.BlockAssets) (*blockchain.Block, error)
}

type Starter interface {
	GetApplication(config *config.ApplicationConfig) (Application, error)
}

type CommandInput struct {
	DataPath string
}

func GetStartCommand(starter Starter) *cli.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return &cli.Command{
		Name:  "start",
		Usage: "Start blockchain application",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "data-path",
				Aliases:     []string{"d"},
				Usage:       "Datapath to store blockchain data",
				DefaultText: filepath.Join(homedir, ".lisk", "default"),
			},
			&cli.StringFlag{
				Name:        "network",
				Aliases:     []string{"n"},
				Usage:       "Datapath to store blockchain data",
				DefaultText: "default",
			},
			&cli.StringFlag{
				Name:        "rpc",
				Aliases:     []string{"r"},
				Usage:       "rpc mode",
				DefaultText: "none",
			},
		},
		Action: func(c *cli.Context) error {
			dataPath := c.String("data-path")
			config := &config.ApplicationConfig{}
			if err := config.InsertDefault(); err != nil {
				return err
			}
			config.System.DataPath = dataPath
			app, err := starter.GetApplication(config)
			if err != nil {
				return err
			}
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				if err := app.Start(); err != nil {
					fmt.Println("Application start error", err)
					if err := app.Stop(); err != nil {
						panic(err)
					}
				}
			}()
			<-signalChan
			if err := app.Stop(); err != nil {
				return err
			}
			return nil
		},
	}
}
