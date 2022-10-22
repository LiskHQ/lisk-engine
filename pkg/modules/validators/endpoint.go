package validators

import (
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type Endpoint struct {
	moduleID uint32
}

func (e *Endpoint) init(moduleID uint32) {
	e.moduleID = moduleID
}

func (e *Endpoint) Get() statemachine.EndpointHandlers {
	return map[string]statemachine.EndpointHandler{}
}
