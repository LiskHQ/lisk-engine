package txpool

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type sampleMod struct {
}

type sampleCommand struct {
}

func (c *sampleCommand) ID() uint32 {
	return 2
}

func (c *sampleCommand) Name() string {
	return "cmd"
}

func (c *sampleCommand) Verify(ctx *statemachine.TransactionVerifyContext) statemachine.VerifyResult {
	store := ctx.GetStore(3, 0)
	errorKeyexist, _ := store.Has([]byte("error-key"))
	if ctx.Transaction().Nonce() == 32 && errorKeyexist {
		return statemachine.NewVerifyResultError(errors.New("error key"))
	}

	return statemachine.NewVerifyResultOK()
}

func (c *sampleCommand) Execute(*statemachine.TransactionExecuteContext) error {
	return nil
}

func (m *sampleMod) ID() uint32 {
	return 3
}

func (m *sampleMod) Name() string {
	return "sample"
}

func (m *sampleMod) GetCommand(name string) (statemachine.Command, bool) {
	return &sampleCommand{}, true
}

func (m *sampleMod) InsertAssets(ctx *statemachine.InsertAssetsContext) error {
	return nil
}

func (m *sampleMod) VerifyAssets(ctx *statemachine.VerifyAssetsContext) error {
	return nil
}

func (m *sampleMod) VerifyTransaction(ctx *statemachine.TransactionVerifyContext) statemachine.VerifyResult {
	return statemachine.NewVerifyResultOK()
}

func (m *sampleMod) InitGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	return nil
}

func (m *sampleMod) FinalizeGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	return nil
}

func (m *sampleMod) BeforeTransactionsExecute(ctx *statemachine.BeforeTransactionsExecuteContext) error {
	return nil
}

func (m *sampleMod) AfterTransactionsExecute(ctx *statemachine.AfterTransactionsExecuteContext) error {
	return nil
}

func (m *sampleMod) BeforeCommandExecute(ctx *statemachine.TransactionExecuteContext) error {
	return nil
}

func (m *sampleMod) AfterCommandExecute(ctx *statemachine.TransactionExecuteContext) error {
	return nil
}
