package blueprint

import (
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

// Module has default behavior for the module.
type Module struct {
}

func (m *Module) Endpoint() statemachine.Endpoint {
	return &Endpoint{}
}

func (m *Module) API() interface{} {
	return nil
}

func (m *Module) Events() []string {
	return []string{}
}

func (m *Module) Init(cfg []byte) error {
	return nil
}

func (m *Module) GetCommand(name string) (statemachine.Command, bool) {
	return nil, false
}

func (m *Module) InsertAssets(ctx *statemachine.InsertAssetsContext) error {
	return nil
}

func (m *Module) VerifyAssets(ctx *statemachine.VerifyAssetsContext) error {
	return nil
}

func (m *Module) VerifyTransaction(ctx *statemachine.TransactionVerifyContext) statemachine.VerifyResult {
	return statemachine.NewVerifyResultOK()
}

func (m *Module) InitGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	return nil
}

func (m *Module) FinalizeGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	return nil
}

func (m *Module) BeforeTransactionsExecute(ctx *statemachine.BeforeTransactionsExecuteContext) error {
	return nil
}

func (m *Module) AfterTransactionsExecute(ctx *statemachine.AfterTransactionsExecuteContext) error {
	return nil
}

func (m *Module) BeforeCommandExecute(ctx *statemachine.TransactionExecuteContext) error {
	return nil
}

func (m *Module) AfterCommandExecute(ctx *statemachine.TransactionExecuteContext) error {
	return nil
}
