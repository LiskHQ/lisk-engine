package mock

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type Pair struct {
	Key   codec.Hex `fieldNumber:"1"`
	Value codec.Hex `fieldNumber:"2"`
}

type DataSetParams struct {
	Pairs []*Pair `fieldNumber:"1"`
	Hash  bool    `fieldNumber:"2"`
}

type SetData struct {
	blueprint.Command
}

func (a *SetData) Name() string {
	return "setData"
}

func (a *SetData) Verify(ctx *statemachine.TransactionVerifyContext) error {
	params := &DataSetParams{}
	if err := params.DecodeStrict(ctx.Transaction().Params()); err != nil {
		return err
	}
	return nil
}

func (a *SetData) Execute(ctx *statemachine.TransactionExecuteContext) error {
	params := &DataSetParams{}
	params.MustDecode(ctx.Transaction().Params())

	validatorStore := ctx.GetStore(storePrefix, storePrefixValidatorsData)
	for _, pair := range params.Pairs {
		value := pair.Value
		if params.Hash {
			value = crypto.Hash(value)
		}
		if err := validatorStore.Set(pair.Key, value); err != nil {
			return err
		}
		if err := ctx.EventQueue().Add("mock", "setdata", pair.Value, []codec.Hex{pair.Key}); err != nil {
			return err
		}
	}
	return nil
}
