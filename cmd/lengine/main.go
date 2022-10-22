// lengine is a CLI which runs Lisk engine process.
package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/engine"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/labi_client"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func resolveIPCPath(originalPath string) (string, error) {
	if strings.Contains(originalPath, "ipc://") {
		return originalPath, nil
	}
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	resolvedPath := strings.Replace(originalPath, "~", userDir, 1)
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath, err = filepath.Abs(resolvedPath)
		if err != nil {
			return "", err
		}
	}
	if !strings.Contains(resolvedPath, "ipc://") {
		resolvedPath = "ipc://" + resolvedPath
	}
	return resolvedPath, nil
}

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}
	app := cli.App{
		Usage: "Lisk blockchain engine",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start blockchain application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "socket-path",
						Aliases:  []string{"s"},
						Usage:    "Path to ABI socket",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "config",
						Aliases:  []string{"c"},
						Usage:    "Path to engine config",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					configPath := c.String("config")
					configData, err := os.ReadFile(configPath)
					if err != nil {
						return err
					}
					engineConfig := &config.Config{}
					if err := json.Unmarshal(configData, engineConfig); err != nil {
						return err
					}

					socketPath := c.String("socket-path")
					resolvedSocketPath, err := resolveIPCPath(socketPath)
					if err != nil {
						return err
					}
					client := labi_client.NewIPCClient(logger)
					logger.Infof("Connecting to socket with path %s", resolvedSocketPath)
					if err := client.Start(resolvedSocketPath); err != nil {
						return err
					}
					defer client.Close()

					lengine := engine.NewEngine(client, engineConfig)

					engineChan := make(chan error, 1)
					signalChan := make(chan os.Signal, 1)
					signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

					go func() {
						if err := lengine.Start(); err != nil {
							engineChan <- err
						}
					}()
					for {
						select {
						case <-signalChan:
							lengine.Stop()
							logger.Info("Closing engine with SIGTERM")
							return nil
						case err := <-engineChan:
							lengine.Stop()
							logger.Errorf("Closing engine with err %s", err)
						}
					}
				},
			},
		},
	}
	err = app.Run(os.Args)
	if err != nil {
		logger.Errorf("Fail running application with %s", err)
	}
}
