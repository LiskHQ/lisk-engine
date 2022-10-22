// app runs blockchain node with the default configurations for debugging.
package main

import (
	"encoding/json"
	"os"
	"path"

	"github.com/LiskHQ/lisk-engine/pkg/client"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
	"github.com/LiskHQ/lisk-engine/pkg/framework/preset"
)

type starter struct{}

func (s *starter) GetApplication(commandConfig *config.ApplicationConfig) (client.Application, error) {
	config := &config.ApplicationConfig{}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	configFile, err := os.ReadFile(path.Join(wd, "./config.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configFile, config); err != nil {
		return nil, err
	}
	if err := config.InsertDefault(); err != nil {
		return nil, err
	}

	// Overwrite command input
	config.Merge(&commandConfig.Config)

	presetApp := preset.NewPresetApplication(config)
	app := presetApp.App

	return app, nil
}
