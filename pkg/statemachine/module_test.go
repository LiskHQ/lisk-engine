package statemachine

import (
	"errors"

	"github.com/stretchr/testify/mock"
)

type sampleCommand struct {
}

func (c *sampleCommand) ID() uint32 {
	return 2
}

func (c *sampleCommand) Name() string {
	return "cmd"
}

func (c *sampleCommand) Verify(ctx *TransactionVerifyContext) VerifyResult {
	store := ctx.GetStore([]byte{0, 0, 0, 3}, []byte{0, 0})
	errorKeyexist := store.Has([]byte("error-key"))
	if ctx.Transaction().Nonce() == 32 && errorKeyexist {
		return NewVerifyResultError(errors.New("error key"))
	}

	return NewVerifyResultOK()
}

func (c *sampleCommand) Execute(*TransactionExecuteContext) error {
	return nil
}

type sampleMod struct {
	mock.Mock
}

func (m *sampleMod) ID() uint32 {
	return 3
}

func (m *sampleMod) Name() string {
	return "sample"
}

func (m *sampleMod) GetCommand(name string) (Command, bool) {
	return &sampleCommand{}, true
}

func (m *sampleMod) InsertAssets(ctx *InsertAssetsContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) VerifyAssets(ctx *VerifyAssetsContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) VerifyTransaction(ctx *TransactionVerifyContext) VerifyResult {
	m.Called(ctx)
	return NewVerifyResultOK()
}

func (m *sampleMod) InitGenesisState(ctx *GenesisBlockProcessingContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) FinalizeGenesisState(ctx *GenesisBlockProcessingContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) BeforeTransactionsExecute(ctx *BeforeTransactionsExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) AfterTransactionsExecute(ctx *AfterTransactionsExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) BeforeCommandExecute(ctx *TransactionExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *sampleMod) AfterCommandExecute(ctx *TransactionExecuteContext) error {
	m.Called(ctx)
	return nil
}

type systemMod struct {
	mock.Mock
}

func (m *systemMod) ID() uint32 {
	return 4
}

func (m *systemMod) Name() string {
	return "system"
}

func (m *systemMod) GetCommand(name string) (Command, bool) {
	return &sampleCommand{}, true
}

func (m *systemMod) InsertAssets(ctx *InsertAssetsContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) VerifyAssets(ctx *VerifyAssetsContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) VerifyTransaction(ctx *TransactionVerifyContext) VerifyResult {
	m.Called(ctx)
	return NewVerifyResultOK()
}

func (m *systemMod) InitGenesisState(ctx *GenesisBlockProcessingContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) FinalizeGenesisState(ctx *GenesisBlockProcessingContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) BeforeTransactionsExecute(ctx *BeforeTransactionsExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) AfterTransactionsExecute(ctx *AfterTransactionsExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) BeforeCommandExecute(ctx *TransactionExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *systemMod) AfterCommandExecute(ctx *TransactionExecuteContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}
