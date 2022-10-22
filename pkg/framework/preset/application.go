// Package preset provides default application which can be extended.
package preset

import (
	"github.com/LiskHQ/lisk-engine/pkg/framework"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
	"github.com/LiskHQ/lisk-engine/pkg/modules/auth"
	"github.com/LiskHQ/lisk-engine/pkg/modules/validators"
)

type PresetApplication struct {
	App        *framework.Application
	AuthModule *auth.Module
}

// NewPresetApplication returns application with preset modules.
func NewPresetApplication(config *config.ApplicationConfig) *PresetApplication {
	app := framework.NewApplication(config)

	validatorModule := validators.NewModule()
	authModule := auth.NewModule()

	if err := app.RegisterModule(validatorModule); err != nil {
		panic(err)
	}

	if err := app.RegisterModule(authModule); err != nil {
		panic(err)
	}

	return &PresetApplication{
		App:        app,
		AuthModule: authModule,
	}
}
