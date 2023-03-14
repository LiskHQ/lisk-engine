package framework

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/validator"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/batchdb"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/engine"
	engineConfig "github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
	"github.com/LiskHQ/lisk-engine/pkg/trie/smt"
)

type Application struct {
	cfg         *config.ApplicationConfig
	cancel      context.CancelFunc
	ctx         context.Context
	logger      log.Logger
	done        chan bool
	modules     []Module
	plugins     []Plugin
	initialized bool

	// instances
	stateDB          *db.DB
	moduleDB         *db.DB
	stateMachineExec *statemachine.Executer
	handler          *ABIHandler
	engine           *engine.Engine
}

func NewApplication(cfg *config.ApplicationConfig) *Application {
	app := &Application{
		cfg:     cfg,
		ctx:     context.Background(),
		done:    make(chan bool),
		modules: []Module{},
		plugins: []Plugin{},
	}
	app.stateMachineExec = statemachine.NewExecuter()
	return app
}

func (app *Application) Start() error {
	if err := app.cfg.InsertDefault(); err != nil {
		return err
	}
	// create root context
	ctx, cancel := context.WithCancel(app.ctx)
	app.cancel = cancel
	defer app.cancel()
	// create new logger
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		return err
	}
	// create new DB
	dataPath, err := app.resolvedDataPath()
	if err != nil {
		return err
	}
	logger.Infof("Starting application with data-path %s", dataPath)
	stateDB, err := db.NewDB(filepath.Join(dataPath, "data", "statedb.db"))
	if err != nil {
		return err
	}
	moduleDB, err := db.NewDB(filepath.Join(dataPath, "data", "module.db"))
	if err != nil {
		return err
	}

	genesis, err := engineConfig.ReadGenesisBlock(&app.cfg.Config)
	if err != nil {
		return err
	}
	app.logger = logger.With("module", "app")
	app.logger.Infof("Starting an application with chain ID %s", app.cfg.Genesis.ChainID)

	app.stateDB = stateDB
	app.moduleDB = moduleDB
	// initialize all dynamic settings
	app.stateMachineExec.Init(logger.With("module", "StateMachine"))
	// Add system modules

	for _, module := range app.modules {
		moduleConfig, exist := app.cfg.Modules[module.Name()]
		if !exist {
			moduleConfig = config.EmptyJSON
		}
		if err := module.Init(moduleConfig); err != nil {
			logger.Errorf("Fail to initialize module %s with %v", module.Name(), err)
			return err
		}
	}

	for _, plugin := range app.plugins {
		c, exist := app.cfg.Plugins[plugin.Name()]
		if !exist {
			c = config.EmptyJSON
		}
		pluginConfig := &PluginConfig{
			Config:   c,
			Context:  app.ctx,
			Logger:   app.logger.With("plugin", plugin.Name()),
			DataPath: filepath.Join(dataPath, "plugins", plugin.Name()),
		}
		if err := plugin.Init(pluginConfig); err != nil {
			return err
		}
	}

	app.initialized = true

	app.logger.Info("Starting application...")

	for _, plugin := range app.plugins {
		go func(p Plugin) {
			app.logger.Infof("Starting plugin %s", p.Name())
			p.Start()
		}(plugin)
	}

	app.handler = NewABIHandler(
		app.ctx,
		app.cfg,
		app.logger.With("module", "abi"),
		app.stateMachineExec,
		genesis,
		stateDB,
		moduleDB,
		app.modules,
	)

	app.engine = engine.NewEngine(app.handler, &app.cfg.Config)

	go func() {
		if err := app.engine.Start(); err != nil {
			app.logger.Errorf("Fail to start engine with %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			app.engine.Stop()
			return nil
		case <-app.done:
			app.engine.Stop()
			return nil
		}
	}
}

func (app *Application) Stop() error {
	app.logger.Info("Closing application")
	app.cancel()
	for _, plugin := range app.plugins {
		plugin.Stop()
	}
	close(app.done)
	return nil
}

func (app *Application) RegisterModule(module Module) error {
	app.modules = append(app.modules, module)
	if err := app.stateMachineExec.AddModule(module); err != nil {
		return err
	}
	return nil
}

func (app *Application) GenerateGenesisBlock(height, timestamp uint32, previoudBlockID codec.Hex, assets blockchain.BlockAssets) (*blockchain.Block, error) {
	// if module is not initialized
	logger, err := log.NewSilentLogger()
	if err != nil {
		return nil, err
	}
	app.stateMachineExec.Init(logger.With("module", "StateMachine"))

	for _, module := range app.modules {
		moduleConfig, exist := app.cfg.Modules[module.Name()]
		if !exist {
			moduleConfig = config.EmptyJSON
		}
		if err := module.Init(moduleConfig); err != nil {
			return nil, err
		}
	}

	stateDB, err := db.NewInMemoryDB()
	if err != nil {
		return nil, err
	}
	diffStore := diffdb.New(stateDB, StateDBPrefixState)

	eventLogger := statemachine.NewEventLogger(height)
	genesisBlock, err := blockchain.NewGenesisBlock(height, timestamp, previoudBlockID, assets)
	if err != nil {
		return nil, err
	}
	genesisContext := statemachine.NewGenesisBlockProcessingContext(
		app.ctx,
		logger,
		diffStore,
		eventLogger,
		genesisBlock.Header,
		genesisBlock.Assets,
	)
	if err := app.stateMachineExec.ExecGenesis(genesisContext); err != nil {
		return nil, err
	}
	batch := stateDB.NewBatch()
	stateBatch := newStateBatch(batch)
	diffStore.Commit(stateBatch)

	smtDB := batchdb.NewWithPrefix(stateDB, batch, StateDBPrefixTree)
	tree := smt.NewTrie(nil, stateTreeKeySize)
	root, err := tree.Update(smtDB, stateBatch.keys, stateBatch.values)
	if err != nil {
		return nil, err
	}
	_, certificateThreshold, validators := genesisContext.GetNextValidators()
	hashingValidators := make([]validator.HashValidator, len(validators))
	for i, val := range validators {
		hashingValidators[i] = validator.NewHashValidator(val.BLSKey, val.BFTWeight)
	}
	validatorsHash, err := validator.ComputeValidatorsHash(hashingValidators, certificateThreshold)
	if err != nil {
		return nil, err
	}
	eventRoot, err := blockchain.CalculateEventRoot(eventLogger.Events())
	if err != nil {
		return nil, err
	}
	genesisBlock.Header.StateRoot = root
	genesisBlock.Header.EventRoot = eventRoot
	genesisBlock.Header.ValidatorsHash = validatorsHash

	return genesisBlock, nil
}

func (app *Application) RegisteredModules() []string {
	moduleNames := make([]string, len(app.modules))
	for i, mod := range app.modules {
		moduleNames[i] = mod.Name()
	}
	return moduleNames
}

func (app *Application) RegisteredPlugins() []string {
	pluginNames := make([]string, len(app.plugins))
	for i, mod := range app.plugins {
		pluginNames[i] = mod.Name()
	}
	return pluginNames
}

func (app *Application) RegisterPlugin(plugin Plugin) error {
	app.plugins = append(app.plugins, plugin)
	return nil
}

func (app *Application) resolvedDataPath() (string, error) {
	if strings.Contains(app.cfg.System.DataPath, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return strings.ReplaceAll(app.cfg.System.DataPath, "~", homedir), nil
	}
	return app.cfg.System.DataPath, nil
}
