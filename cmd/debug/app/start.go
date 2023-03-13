package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/framework"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
)

type Starter interface {
	GetApplication(config *config.ApplicationConfig) (*framework.Application, error)
}

type CommandInput struct {
	DataPath string
}

func GetStartCommand(starter Starter) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start blockchain application",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "data-path",
				Aliases: []string{"d"},
				Usage:   "Datapath to store blockchain data",
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
			configFile, err := os.ReadFile(path.Join(dataPath, "./config.json"))
			if err != nil {
				return err
			}
			config := &config.ApplicationConfig{}
			if err := json.Unmarshal(configFile, config); err != nil {
				return err
			}
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
