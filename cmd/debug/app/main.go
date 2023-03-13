package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Usage: "Lisk Golang SDK CLI tool",
		Commands: []*cli.Command{
			GetStartCommand(&starter{}),
			GetGenesisCommand(&starter{}),
			GetKeysCommand(),
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
