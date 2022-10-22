package statemachine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

type execEnv struct {
	executer    *Executer
	sample      *sampleMod
	system      *systemMod
	diffStore   *diffdb.Database
	eventLogger *EventLogger
}

func prepareExecEnv(t *testing.T) *execEnv {
	executer := NewExecuter()
	executer.Init(log.DefaultLogger)
	mod1 := &sampleMod{}
	mod2 := &systemMod{}
	err := executer.AddModule(mod1)
	assert.NoError(t, err)
	err = executer.AddModule(mod2)
	assert.NoError(t, err)

	database, _ := db.NewInMemoryDB()

	diffStore := diffdb.New(database, []byte{1})
	return &execEnv{
		executer:    executer,
		sample:      mod1,
		system:      mod2,
		diffStore:   diffStore,
		eventLogger: NewEventLogger(1),
	}
}

func TestExecuterAddModule(t *testing.T) {
	executer := NewExecuter()
	executer.Init(log.DefaultLogger)

	err := executer.AddModule(&sampleMod{})
	assert.NoError(t, err)

	err = executer.AddModule(&sampleMod{})
	assert.EqualError(t, err, "module Name sample cannot be registered twice")

	err = executer.AddModule(&systemMod{})
	assert.NoError(t, err)

	err = executer.AddModule(&systemMod{})
	assert.EqualError(t, err, "module Name system cannot be registered twice")

	assert.True(t, executer.ModuleRegistered("sample"))
	assert.True(t, executer.ModuleRegistered("system"))
	assert.False(t, executer.ModuleRegistered("sample2"))
}

func TestExecuterExecGenesis(t *testing.T) {
	env := prepareExecEnv(t)

	ctx := NewGenesisBlockProcessingContext(
		context.Background(),
		log.DefaultLogger,
		env.diffStore,
		env.eventLogger,
		&blockchain.BlockHeader{},
		[]*blockchain.BlockAsset{},
	)
	env.sample.On("InitGenesisState", mock.Anything).Return(nil).Once()
	env.sample.On("FinalizeGenesisState", mock.Anything).Return(nil).Once()
	env.system.On("InitGenesisState", mock.Anything).Return(nil).Once()
	env.system.On("FinalizeGenesisState", mock.Anything).Return(nil).Once()

	err := env.executer.ExecGenesis(ctx)
	assert.NoError(t, err)

	env.sample.AssertCalled(t, "InitGenesisState", mock.Anything)
	env.sample.AssertCalled(t, "FinalizeGenesisState", mock.Anything)
	env.system.AssertCalled(t, "InitGenesisState", mock.Anything)
	env.system.AssertCalled(t, "FinalizeGenesisState", mock.Anything)

	env = prepareExecEnv(t)

	env.sample.On("InitGenesisState", mock.Anything).Return(errors.New("some error")).Once()
	env.sample.On("FinalizeGenesisState", mock.Anything).Return(nil).Times(0)
	env.system.On("InitGenesisState", mock.Anything).Return(nil).Once()
	env.system.On("FinalizeGenesisState", mock.Anything).Return(nil).Once()

	err = env.executer.ExecGenesis(ctx)
	assert.EqualError(t, err, "some error")

	env.sample.AssertCalled(t, "InitGenesisState", mock.Anything)
	env.sample.AssertNotCalled(t, "FinalizeGenesisState", mock.Anything)
	env.system.AssertNotCalled(t, "InitGenesisState", mock.Anything)
	env.system.AssertNotCalled(t, "FinalizeGenesisState", mock.Anything)
}
