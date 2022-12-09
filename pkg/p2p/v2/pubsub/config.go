package p2p

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is a struct that hold all information.
type Config struct {
	IsBootstrapNode bool     `json:"isBootstrapNode"`
	NetworkName     string   `json:"networkName"`
	BootNode        []string `json:"bootNode"`
}

// ReadConfigFromFile reads json config with path file.
func ReadConfigFromFile(fpath string) (c Config, err error) {
	var file *os.File
	file, err = os.Open(fpath)
	if err != nil {
		return
	}
	defer func() {
		errClose := file.Close()
		if errClose != nil {
			fmt.Println("Error in closing file:", errClose)
		}
	}()

	err = json.NewDecoder(file).Decode(&c)
	if err != nil {
		return
	}

	return
}

// DefaultConfig returns a config by default values.
func DefaultConfig() Config {
	return Config{
		IsBootstrapNode: false,
		NetworkName:     "lisk-testnet",
		BootNode:        []string{},
	}
}
