// app runs blockchain node with the default configurations for debugging.
package main

import (
	"github.com/LiskHQ/lisk-engine/cmd/debug/app/modules/mock"
	"github.com/LiskHQ/lisk-engine/pkg/framework"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
)

type starter struct{}

func (s *starter) GetApplication(commandConfig *config.ApplicationConfig) (*framework.Application, error) {
	app := framework.NewApplication(commandConfig)
	mockModule := mock.NewModule()
	if err := app.RegisterModule(mockModule); err != nil {
		return nil, err
	}

	return app, nil
}
