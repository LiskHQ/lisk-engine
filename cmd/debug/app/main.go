package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/client"
)

func main() {
	app := cli.App{
		Usage: "Lisk Golang SDK CLI tool",
		Commands: []*cli.Command{
			client.GetStartCommand(&starter{}),
			client.GetGenesisCommand(&starter{}),
			client.GetKeysCommand(),
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
