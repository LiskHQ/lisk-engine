package auth

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
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
		"getAccount": e.HandleGetAccount,
	}
}

func (e *Endpoint) HandleGetAccount(w rpc.EndpointResponseWriter, r *statemachine.EndpointRequest) {
	req := &GetAccountRequest{}
	if err := r.Data(req); err != nil {
		w.Error(err)
		return
	}
	if len(req.Address) == 0 {
		w.Error(errors.New("address must be specified"))
		return
	}
	userStore := r.GetStore(e.moduleID, storePrefixUserAccount)
	encoded, err := userStore.Get(req.Address)
	if err != nil {
		w.Error(err)
		return
	}
	data := &UserAccount{}
	if err := data.Decode(encoded); err != nil {
		w.Error(err)
		return
	}
	mandatoryKeys := make([]codec.Hex, len(data.MandatoryKeys))
	for i, key := range data.MandatoryKeys {
		mandatoryKeys[i] = key
	}
	optionalKeys := make([]codec.Hex, len(data.OptionalKeys))
	for i, key := range data.OptionalKeys {
		optionalKeys[i] = key
	}
	resp := &GetAccountResponse{
		Nonce:              data.Nonce,
		NumberOfSignatures: data.NumberOfSignatures,
		MandatoryKeys:      mandatoryKeys,
		OptionalKeys:       optionalKeys,
	}
	if err := w.Write(resp); err != nil {
		w.Error(err)
		return
	}
}
