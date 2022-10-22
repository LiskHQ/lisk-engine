package blueprint

import (
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

// BaseModule has default behavior for the module.
type Plugin struct {
}

func (b *Plugin) Endpoint() statemachine.Endpoint {
	return &Endpoint{}
}

func (b *Plugin) Events() []string {
	return []string{}
}
