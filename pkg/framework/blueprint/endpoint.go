package blueprint

import (
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type Endpoint struct{}

func (a *Endpoint) Get() statemachine.EndpointHandlers {
	return map[string]statemachine.EndpointHandler{}
}
