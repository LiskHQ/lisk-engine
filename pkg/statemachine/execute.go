package statemachine

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

// Executer handles block and transaction execution.
type Executer struct {
	modules []Module
	logger  log.Logger
}

// NewExecuter returns a execution.
func NewExecuter() *Executer {
	return &Executer{
		modules: []Module{},
	}
}

func (s *Executer) Init(logger log.Logger) {
	s.logger = logger
}

func (s *Executer) ModuleRegistered(name string) bool {
	for _, m := range s.modules {
		if m.Name() == name {
			return true
		}
	}
	return false
}

func (s *Executer) AddModule(mod Module) error {
	for _, existingModule := range s.modules {
		if existingModule.Name() == mod.Name() {
			return fmt.Errorf("module Name %s cannot be registered twice", mod.Name())
		}
	}
	s.modules = append(s.modules, mod)
	return nil
}
func (s *Executer) RegisteredModules() []Module {
	return s.modules
}

// ExecGenesis  block.
func (s *Executer) ExecGenesis(ctx *GenesisBlockProcessingContext) error {
	s.logger.Info("Executing genesis block")
	ctx.eventLogs.SetDefaultTopic(EventTopicInitGenesisState)
	for _, mod := range s.modules {
		if err := mod.InitGenesisState(ctx); err != nil {
			s.logger.Errorf("Fail executing InitGenesisState for module %s with %v", mod.Name(), err)
			return err
		}
	}
	ctx.eventLogs.SetDefaultTopic(EventTopicFinalizeGenesisState)
	for _, mod := range s.modules {
		if err := mod.FinalizeGenesisState(ctx); err != nil {
			s.logger.Errorf("Fail executing FinalizeGenesisState for module %s with %v", mod.Name(), err)
			return err
		}
	}
	return nil
}

type TransactionExecutionError struct {
	ID  codec.Hex
	Err error
}

func NewTransactionExecError(id codec.Hex, err error) *TransactionExecutionError {
	return &TransactionExecutionError{
		ID:  id,
		Err: err,
	}
}

func (e *TransactionExecutionError) Error() string {
	return e.Err.Error()
}

func (s *Executer) VerifyAssets(ctx *VerifyAssetsContext) error {
	for _, mod := range s.modules {
		if err := mod.VerifyAssets(ctx); err != nil {
			s.logger.Errorf("Fail executing afterBlockApply for module %s with %v", mod.Name(), err)
			return err
		}
	}
	return nil
}

func (s *Executer) VerifyTransaction(ctx *TransactionVerifyContext) VerifyResult {
	for _, mod := range s.modules {
		if result := mod.VerifyTransaction(ctx); !result.OK() {
			s.logger.Debugf("Fail verifying for module %s with %v", mod.Name(), result.Err())
			return result
		}
	}
	command, ok := s.GetCommand(ctx.transaction.Module, ctx.transaction.Command)
	if !ok {
		return NewVerifyResultError(fmt.Errorf("module %s Command %s does not exist", ctx.transaction.Module, ctx.transaction.Command))
	}
	if result := command.Verify(ctx); !result.OK() {
		s.logger.Errorf("Fail verifying for command %s with %v", command.Name(), result.Err())
		return result
	}
	return NewVerifyResultOK()
}

func (s *Executer) InsertAssets(ctx *InsertAssetsContext) error {
	for _, mod := range s.modules {
		if err := mod.InsertAssets(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Executer) BeforeExecute(ctx *BeforeTransactionsExecuteContext) error {
	ctx.eventLogs.SetDefaultTopic(EventTopicBeforeTransactionsExecute)
	for _, mod := range s.modules {
		if err := mod.BeforeTransactionsExecute(ctx); err != nil {
			s.logger.Errorf("Failed executing beforeTransactionsExecute for module %s", mod.Name())
			return err
		}
	}
	return nil
}

func (s *Executer) AfterExecute(ctx *AfterTransactionsExecuteContext) error {
	ctx.eventLogs.SetDefaultTopic(EventTopicAfterTransactionsExecute)
	for _, mod := range s.modules {
		if err := mod.AfterTransactionsExecute(ctx); err != nil {
			s.logger.Errorf("Fail executing afterTransactionsExecute for module %s with %v", mod.Name(), err)
			return err
		}
	}
	return nil
}

// ExecuteTransaction transaction.
func (s *Executer) ExecuteTransaction(ctx *TransactionExecuteContext) ExecResult {
	ctx.eventLogs.SetDefaultTopic(ctx.transaction.ID)
	for _, mod := range s.modules {
		if err := mod.BeforeCommandExecute(ctx); err != nil {
			s.logger.Debugf("Fail on BeforeCommandExecute for module: %s with %s", mod.Name(), err)
			return NewExecResultInvalid(err)
		}
	}
	command, ok := s.GetCommand(ctx.transaction.Module, ctx.transaction.Command)
	if !ok {
		err := NewTransactionExecError(ctx.transaction.ID, fmt.Errorf("module %s Command %s does not exist", ctx.transaction.Module, ctx.transaction.Command))
		s.logger.Debugf("Fail to get command for TxID: %s with %s", ctx.transaction.ID, err)
		return NewExecResultInvalid(err)
	}
	success := true
	var cmdErr error
	ctx.eventLogs.CreateSnapshot()
	snapshotID := ctx.diffStore.Snapshot()
	if err := command.Execute(ctx); err != nil {
		success = false
		cmdErr = err
		if err := ctx.diffStore.RestoreSnapshot(snapshotID); err != nil {
			return NewExecResultInvalid(err)
		}
		ctx.eventLogs.RestoreSnapshot()
	}
	ctx.diffStore.DeleteSnapshot(snapshotID)
	for _, mod := range s.modules {
		if err := mod.AfterCommandExecute(ctx); err != nil {
			s.logger.Debugf("Fail on AfterCommandExecute for module %s with %s", mod.Name(), err)
			return NewExecResultInvalid(NewTransactionExecError(ctx.transaction.ID, err))
		}
	}
	if err := ctx.eventLogs.Add(ctx.transaction.Module, blockchain.EventNameDefault, blockchain.NewStandardTransactionEventData(success), []codec.Hex{}); err != nil {
		s.logger.Warningf("Fail to add standard event with%s", err)
		return NewExecResultInvalid(NewTransactionExecError(ctx.transaction.ID, err))
	}
	if !success {
		return NewExecResultFail(cmdErr)
	}
	return NewExecResultOK()
}

func (s *Executer) GetCommand(module, command string) (Command, bool) {
	for _, mod := range s.modules {
		if mod.Name() != module {
			continue
		}
		return mod.GetCommand(command)
	}
	return nil, false
}
