// Package engine implements blockchain engine using other packages.
// It is responsible for handling network, consensus, transaction pool management and block generation.
package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/engine/endpoint"
	"github.com/LiskHQ/lisk-engine/pkg/generator"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/router"
	"github.com/LiskHQ/lisk-engine/pkg/rpc"
	"github.com/LiskHQ/lisk-engine/pkg/txpool"
)

type Engine struct {
	abi         labi.ABI
	ctx         context.Context
	cancel      context.CancelFunc
	logger      log.Logger
	config      *config.Config
	initialized bool

	// instances
	router          *router.Router
	blockchainDB    *db.DB
	generatorDB     *db.DB
	conn            *p2p.P2P
	chain           *blockchain.Chain
	consensusExec   *consensus.Executer
	transactionPool *txpool.TransactionPool
	generator       *generator.Generator
	server          *rpc.RPCServer
}

func NewEngine(abi labi.ABI, config *config.Config) *Engine {
	return &Engine{
		abi:         abi,
		config:      config,
		initialized: false,
	}
}

func (e *Engine) Start() error {
	if err := e.init(); err != nil {
		return err
	}
	if err := e.config.InsertDefault(); err != nil {
		return err
	}
	genesisBlock, err := config.ReadGenesisBlock(e.config)
	if err != nil {
		return err
	}
	if err := genesisBlock.Init(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.ctx = ctx
	e.cancel = cancel
	defer e.cancel()

	dataPath, err := resolvedDataPath(e.config.System.DataPath)
	if err != nil {
		return err
	}
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		return err
	}
	e.logger = logger
	e.logger.Infof("Starting application with data-path %s", dataPath)
	blockchainDB, err := db.NewDB(filepath.Join(dataPath, "data", "blockchain.db"))
	if err != nil {
		return err
	}
	e.blockchainDB = blockchainDB
	generatorDB, err := db.NewDB(filepath.Join(dataPath, "data", "generator.db"))
	if err != nil {
		return err
	}
	e.generatorDB = generatorDB
	e.chain.Init(genesisBlock, blockchainDB)
	if err := e.router.Init(e.blockchainDB, e.logger.With("module", "broker"), e.chain); err != nil {
		return err
	}

	chainEndpoint := endpoint.NewChainEndpoint(e.chain, e.consensusExec, e.conn, e.transactionPool, e.abi)
	for method, handler := range chainEndpoint.Endpoint() {
		if err := e.router.RegisterEndpoint("chain", method, handler); err != nil {
			return err
		}
	}
	systemEndpoint := endpoint.NewSystemEndpoint(e.config, e.chain, e.consensusExec, e.conn, e.transactionPool, e.abi)
	for method, handler := range systemEndpoint.Endpoint() {
		if err := e.router.RegisterEndpoint("system", method, handler); err != nil {
			return err
		}
	}
	networkEndpoint := endpoint.NewNetworkEndpoint(e.config, e.chain, e.consensusExec, e.conn, e.transactionPool, e.abi)
	for method, handler := range networkEndpoint.Endpoint() {
		if err := e.router.RegisterEndpoint("network", method, handler); err != nil {
			return err
		}
	}
	generatorEndpoint := endpoint.NewGeneratorEndpoint(e.config, e.chain, e.consensusExec, e.generator, e.blockchainDB, e.generatorDB, e.abi)
	for method, handler := range generatorEndpoint.Endpoint() {
		if err := e.router.RegisterEndpoint("generator", method, handler); err != nil {
			return err
		}
	}

	e.router.RegisterNotFoundHandler(func(namespace, method string, w router.EndpointResponseWriter, r *router.EndpointRequest) {
		resp, err := e.abi.Query(&labi.QueryRequest{
			Method: method,
			Params: r.Params(),
			Header: e.chain.LastBlock().Header,
		})
		if err != nil {
			w.Error(err)
			return
		}
		if err := w.Write(resp); err != nil {
			w.Error(err)
			return
		}
	})
	if _, err := e.abi.Clear(&labi.ClearRequest{}); err != nil {
		return err
	}

	if err := e.consensusExec.Init(&consensus.ExecuterInitParam{
		CTX:          ctx,
		Logger:       e.logger,
		Database:     blockchainDB,
		GenesisBlock: genesisBlock,
	}); err != nil {
		return err
	}
	if err := e.transactionPool.Init(
		e.ctx,
		e.logger.With("module", "TransactionPool"),
		blockchainDB,
		e.chain,
		e.conn,
		e.abi,
	); err != nil {
		return err
	}
	if err := e.generator.Init(&generator.GeneratorInitParams{
		CTX:          e.ctx,
		Logger:       e.logger,
		BlockchainDB: e.blockchainDB,
		GeneratorDB:  e.generatorDB,
		Cfg:          e.config,
	}); err != nil {
		return err
	}
	e.initialized = true

	e.logger.Info("Starting application...")
	nodeInfo := &p2p.NodeInfo{
		ChainID:          e.config.Genesis.ChainID.String(),
		NetworkVersion:   e.config.Network.Version,
		AdvertiseAddress: e.config.Network.AdvertiseAddress,
		Port:             e.config.Network.Port,
		Options:          []byte{},
	}
	// start P2P
	go func() {
		if err := e.conn.Start(e.logger, nodeInfo); err != nil {
			e.logger.Error("Fail to start connection. stopping")
			e.Stop()
		}
	}()
	// start consensus
	go func() {
		if err := e.consensusExec.Start(); err != nil {
			e.logger.Error("Fail to start connection. stopping")
			e.Stop()
		}
	}()
	go e.transactionPool.Start()
	go e.generator.Start()

	e.server = rpc.NewRPCServer(
		e.logger,
		e.config.RPC.Modes,
		e.router,
		e.config.RPC.Port,
		e.config.RPC.Host,
		"",
	)
	go func() {
		if err := e.server.ListenAndServe(); err != nil {
			e.logger.Error("Fail to start connection. stopping")
			e.Stop()
		}
	}()
	go e.handleEvents()

	if _, err := e.abi.Init(&labi.InitRequest{
		ChainID:         e.config.Genesis.ChainID,
		LastBlockHeight: e.chain.LastBlock().Header.Height,
		LastStateRoot:   e.chain.LastBlock().Header.StateRoot,
	}); err != nil {
		return err
	}

	for range ctx.Done() {
		if e.server != nil {
			e.server.Close()
		}
		if err := e.conn.Stop(); err != nil {
			e.logger.Error("Fail to stop connection with %w", err)
		}
		if err := e.consensusExec.Stop(); err != nil {
			e.logger.Error("Fail to stop consensus with %w", err)
		}
	}
	return nil
}

func (e *Engine) Stop() {
	e.logger.Info("Closing application")
	e.cancel()
}

func (e *Engine) init() error {
	e.conn = p2p.NewP2P(e.config.Network)
	e.chain = blockchain.NewChain(&blockchain.ChainConfig{
		MaxBlockCache:         e.config.System.GetMaxBlokckCache(),
		ChainID:               e.config.Genesis.ChainID,
		MaxTransactionsLength: e.config.Genesis.MaxTransactionsSize,
	})
	e.consensusExec = consensus.NewExecuter(&consensus.ExecuterConfig{
		CTX:       e.ctx,
		Chain:     e.chain,
		ABI:       e.abi,
		Conn:      e.conn,
		BlockTime: e.config.Genesis.BlockTime,
		BatchSize: int(e.config.Genesis.BFTBatchSize),
	})
	poolConfig := &txpool.TransactionPoolConfig{
		MaxTransactions:             e.config.TransactionPool.MaxTransactions,
		MaxTransactionsPerAccount:   e.config.TransactionPool.MaxTransactionsPerAccount,
		TransactionExpiryTime:       e.config.TransactionPool.TransactionExpiryTime,
		MinEntranceFeePriority:      e.config.TransactionPool.MinEntranceFeePriority,
		MinReplacementFeeDifference: e.config.TransactionPool.MinReplacementFeeDifference,
	}
	e.transactionPool = txpool.NewTransactionPool(poolConfig)
	e.generator = generator.NewGenerator(&generator.GeneratorParams{
		ABI:       e.abi,
		Consensus: e.consensusExec,
		Pool:      e.transactionPool,
		Chain:     e.chain,
	})
	e.router = router.NewRouter()
	return nil
}

func (e *Engine) handleEvents() {
	newBlock := e.consensusExec.Subscribe(consensus.EventBlockNew)
	deleteBlock := e.consensusExec.Subscribe(consensus.EventBlockDelete)
	chainFork := e.consensusExec.Subscribe(consensus.EventChainFork)
	validatorChange := e.consensusExec.Subscribe(consensus.EventValidatorsChange)
	networkNewBlock := e.consensusExec.Subscribe(consensus.EventNetworkBlockNew)
	txpoolNewTransaction := e.transactionPool.Subscribe(txpool.EventTransactionNew)
	txpoolTransactionAnnouncement := e.transactionPool.Subscribe(txpool.EventTransactionAnnouncement)
	for {
		select {
		case <-e.ctx.Done():
			return
		case msg := <-newBlock:
			eventMsg, ok := msg.(*consensus.EventBlockNewMessage)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventBlockHeader{
				BlockHeader: eventMsg.Block.Header,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventChainNewBlock, publishData)
		case msg := <-deleteBlock:
			eventMsg, ok := msg.(*consensus.EventBlockDeleteMessage)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventBlockHeader{
				BlockHeader: eventMsg.Block.Header,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventChainDeleteBlock, publishData)
		case msg := <-chainFork:
			eventMsg, ok := msg.(*consensus.EventChainForkMessage)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventBlockHeader{
				BlockHeader: eventMsg.Block.Header,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventChainForked, publishData)
		case msg := <-validatorChange:
			eventMsg, ok := msg.(*consensus.EventChangeValidator)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventValidatorChange{
				Validators:           eventMsg.NextValidators,
				PrecommitThreshold:   eventMsg.PrecommitThreshold,
				CertificateThreshold: eventMsg.CertificateThreshold,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventChainValidatorsChanged, publishData)
		case msg := <-networkNewBlock:
			eventMsg, ok := msg.(*consensus.EventNetworkBlockNewMessage)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventBlockHeader{
				BlockHeader: eventMsg.Block.Header,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventNetworkNewBlock, publishData)
		case msg := <-txpoolTransactionAnnouncement:
			eventMsg, ok := msg.(*txpool.EventNewTransactionAnnouncementMessage)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventTransactionIDs{
				TransactionIDs: eventMsg.TransactionIDs,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventNetworkNewTransaction, publishData)
		case msg := <-txpoolNewTransaction:
			eventMsg, ok := msg.(*txpool.EventNewTransactionMessage)
			if !ok {
				e.logger.Errorf("Failed to cast event data with %v", msg)
				continue
			}
			data := &EventTransaction{
				Transaction: eventMsg.Transaction,
			}
			publishData, err := json.Marshal(data)
			if err != nil {
				e.logger.Errorf("Failed to marshal publishing data with %s", err)
				continue
			}
			e.server.Publish(RPCEventTxpoolNewTransaction, publishData)
		}
	}
}

func resolvedDataPath(dataPath string) (string, error) {
	if strings.Contains(dataPath, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return strings.ReplaceAll(dataPath, "~", homedir), nil
	}
	return dataPath, nil
}
