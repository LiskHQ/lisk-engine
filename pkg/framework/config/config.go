// Package config provides framework config structure.
package config

import (
	"encoding/json"

	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
)

var EmptyJSON = []byte("{}")

type ApplicationConfig struct {
	config.Config
	Modules map[string]json.RawMessage `json:"modules"`
	Plugins map[string]json.RawMessage `json:"plugins"`
}
