// Package auth implements auth module following [LIP-0041].
//
// [LIP-0041]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0041.md
package mock

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

const (
	MaxKeyCount = 64
)

var (
	storePrefix               = []byte{0x00, 0x00, 0x00, 0x00}
	storePrefixValidatorsData = []byte{0x00, 0x00}
	storePrefixUserAccount    = []byte{0x80, 0x00}
	emptyKey                  = []byte{}
)

type AuthModuleConfig struct {
}

type Module struct {
	blueprint.Module
	api      *API
	endpoint *Endpoint
}

func NewModule() *Module {
	return &Module{
		api:      &API{},
		endpoint: &Endpoint{},
	}
}

func (m *Module) Name() string {
	return "auth"
}

func (m *Module) GetCommand(name string) (statemachine.Command, bool) {
	return nil, false
}

func (m *Module) API() *API {
	return m.api
}

func (m *Module) Endpoint() statemachine.Endpoint {
	return m.endpoint
}
func (m *Module) Init(cfg []byte) error {
	return nil
}

func (m *Module) InitGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	assetBytes, ok := ctx.BlockAssets().GetAsset("mock")
	if !ok {
		return fmt.Errorf("asset does not exist for mock")
	}
	assets := &ValidatorsData{}
	if err := assets.DecodeStrict(assetBytes); err != nil {
		return err
	}
	validatorStore := ctx.GetStore(storePrefix, storePrefixValidatorsData)
	if err := statemachine.SetEncodable(validatorStore, emptyKey, assets); err != nil {
		return err
	}
	validators := assets.toLABIValidators()
	sum := assets.totalWeight()
	threshold := sum * 2 / 3
	ctx.SetNextValidators(threshold, threshold, validators)

	return nil
}

func (m *Module) VerifyTransaction(ctx *statemachine.TransactionVerifyContext) statemachine.VerifyResult {
	userStore := ctx.GetStore(storePrefix, storePrefixUserAccount)
	userAccount := &UserAccount{}
	if err := statemachine.GetDecodableOrDefault(userStore, ctx.Transaction().SenderAddress(), userAccount); err != nil {
		return statemachine.NewVerifyResultError(err)
	}
	if err := verifySignatures(userAccount, ctx.ChainID(), ctx.Transaction()); err != nil {
		return statemachine.NewVerifyResultError(err)
	}
	if ctx.Transaction().Nonce() < userAccount.Nonce {
		return statemachine.NewVerifyResultError(fmt.Errorf(
			"address %s has higher nonce %d than transaction nonce %d",
			ctx.Transaction().SenderAddress(),
			userAccount.Nonce, ctx.Transaction().Nonce(),
		))
	}
	if ctx.Transaction().Nonce() > userAccount.Nonce {
		return statemachine.NewVerifyResultPending(fmt.Errorf(
			"address %s has lower nonce %d than transaction nonce %d",
			ctx.Transaction().SenderAddress(),
			userAccount.Nonce, ctx.Transaction().Nonce(),
		))
	}
	return statemachine.NewVerifyResultOK()
}

func (m *Module) BeforeCommandExecute(ctx *statemachine.TransactionExecuteContext) error {
	userAccount := &UserAccount{}
	userStore := ctx.GetStore(storePrefix, storePrefixUserAccount)
	if err := statemachine.GetDecodableOrDefault(userStore, ctx.Transaction().SenderAddress(), userAccount); err != nil {
		return err
	}
	if ctx.Transaction().Nonce() != userAccount.Nonce {
		return fmt.Errorf(
			"address %s has nonce %d but transaction nonce is %d",
			ctx.Transaction().SenderAddress(),
			userAccount.Nonce, ctx.Transaction().Nonce(),
		)
	}
	userAccount.Nonce += 1
	if err := statemachine.SetEncodable(userStore, ctx.Transaction().SenderAddress(), userAccount); err != nil {
		return err
	}
	return nil
}

func (m *Module) AfterTransactionsExecute(ctx *statemachine.AfterTransactionsExecuteContext) error {
	validatorStore := ctx.GetStore(storePrefix, storePrefixValidatorsData)
	validatorsData := &ValidatorsData{}
	if err := statemachine.GetDecodable(validatorStore, emptyKey, validatorsData); err != nil {
		return err
	}
	validators := validatorsData.toLABIValidators()
	sum := validatorsData.totalWeight()
	threshold := sum * 2 / 3
	ctx.SetNextValidators(threshold, threshold, validators)
	return nil
}
