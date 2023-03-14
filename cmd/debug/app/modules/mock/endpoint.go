package mock

import (
	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/rpc"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type Endpoint struct {
	blueprint.Endpoint
	moduleID uint32
}

func (e *Endpoint) init(moduleID uint32) {
	e.moduleID = moduleID
}

func (e *Endpoint) Get() statemachine.EndpointHandlers {
	return map[string]statemachine.EndpointHandler{
		"getValidators": e.HandleGetValidators,
	}
}

func (e *Endpoint) HandleGetValidators(w rpc.EndpointResponseWriter, r *statemachine.EndpointRequest) {
	validatorsStore := r.GetStore(storePrefix, storePrefixValidatorsData)
	validatorsData := &ValidatorsData{}
	if err := statemachine.GetDecodable(validatorsStore, emptyKey, validatorsData); err != nil {
		w.Error(err)
		return
	}
	w.Write(validatorsData)
}
