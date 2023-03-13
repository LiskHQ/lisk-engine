package mock

import (
	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type UpdateValidatorsParams struct {
	Keys []*ValidatorKey `fieldNumber:"1"`
}

type UpdateValidators struct {
	blueprint.Command
}

func (a *UpdateValidators) Name() string {
	return "updateValidators"
}

func (a *UpdateValidators) Verify(ctx *statemachine.TransactionVerifyContext) error {
	params := &UpdateValidatorsParams{}
	if err := params.DecodeStrict(ctx.Transaction().Params()); err != nil {
		return err
	}
	if err := (&ValidatorsData{
		Keys: params.Keys,
	}).validate(); err != nil {
		return err
	}

	return nil
}

func (a *UpdateValidators) Execute(ctx *statemachine.TransactionExecuteContext) error {
	params := &UpdateValidatorsParams{}
	params.MustDecode(ctx.Transaction().Params())

	validatorStore := ctx.GetStore(storePrefix, storePrefixValidatorsData)
	if err := statemachine.SetEncodable(validatorStore, emptyKey, &ValidatorsData{
		Keys: params.Keys,
	}); err != nil {
		return err
	}

	return nil
}
