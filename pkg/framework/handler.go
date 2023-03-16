package framework

import (
	"context"
	"fmt"
	"strings"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/batchdb"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/rpc"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
	"github.com/LiskHQ/lisk-engine/pkg/trie/smt"
)

const (
	querySeparator = "_"
)

var (
	GeneratorDBPrefix []byte = []byte{0}
	emptyBytes        []byte = []byte{}
)

type ABIHandler struct {
	ctx              context.Context
	config           *config.ApplicationConfig
	logger           log.Logger
	chainID          codec.Hex
	genesisBlock     *blockchain.Block
	stateMachine     *statemachine.Executer
	stateDB          *db.DB
	moduleDB         *db.DB
	executionContext *executionContext
	modules          []Module
}

func NewABIHandler(
	ctx context.Context,
	config *config.ApplicationConfig,
	logger log.Logger,
	stateMachine *statemachine.Executer,
	genesisBlock *blockchain.Block,
	stateDB *db.DB,
	moduleDB *db.DB,
	modules []Module,
) *ABIHandler {
	return &ABIHandler{
		ctx:          ctx,
		config:       config,
		logger:       logger,
		genesisBlock: genesisBlock,
		stateMachine: stateMachine,
		stateDB:      stateDB,
		moduleDB:     moduleDB,
		modules:      modules,
	}
}

type executionContext struct {
	id          codec.Hex
	header      *blockchain.BlockHeader
	diffStore   *diffdb.Database
	moduleStore *diffdb.Database
}

func (a *ABIHandler) Init(req *labi.InitRequest) (*labi.InitResponse, error) {
	a.chainID = req.ChainID
	currentState, exist := a.stateDB.Get(bytes.Join(StateDBPrefixTreeState, emptyBytes))
	if !exist {
		currentState = bytes.Join(bytes.FromUint32(0), emptyHash)
	}

	currentHeight := bytes.ToUint32(currentState[:4])
	currentRoot := currentState[4:]

	if currentHeight < req.LastBlockHeight {
		return nil, fmt.Errorf("invalid application state. current height is %d but engine state is %d ", currentHeight, req.LastBlockHeight)
	}
	for currentHeight > req.LastBlockHeight {
		a.logger.Debugf("Reverting application state from %d to %d", currentHeight, req.LastBlockHeight)
		nextRoot, err := a.revert(currentHeight, currentRoot, nil)
		if err != nil {
			return nil, err
		}
		currentRoot = nextRoot
		currentHeight -= 1
	}
	if !bytes.Equal(currentRoot, req.LastStateRoot) {
		return nil, fmt.Errorf("invalid application state. conflict in state root at height %d with application %s engine %s", currentHeight, codec.Hex(currentRoot), codec.Hex(req.LastStateRoot))
	}

	return &labi.InitResponse{}, nil
}

func (a *ABIHandler) InitStateMachine(req *labi.InitStateMachineRequest) (*labi.InitStateMachineResponse, error) {
	if a.executionContext != nil {
		return nil, fmt.Errorf("state machine is already initialized with id %s", a.executionContext.id)
	}
	req.Header.Init()
	id := crypto.Hash(req.Header.Encode())
	a.executionContext = &executionContext{
		id:          id,
		diffStore:   diffdb.New(a.stateDB, StateDBPrefixState),
		moduleStore: diffdb.New(a.moduleDB, GeneratorDBPrefix),
		header:      req.Header,
	}
	return &labi.InitStateMachineResponse{
		ContextID: id,
	}, nil
}

func (a *ABIHandler) InitGenesisState(req *labi.InitGenesisStateRequest) (*labi.InitGenesisStateResponse, error) {
	eventLogger := statemachine.NewEventLogger(a.executionContext.header.Height)
	context := statemachine.NewGenesisBlockProcessingContext(
		a.ctx,
		a.logger,
		a.executionContext.diffStore,
		eventLogger,
		a.executionContext.header,
		a.genesisBlock.Assets,
	)
	if err := a.stateMachine.ExecGenesis(context); err != nil {
		return nil, err
	}
	precommitThreshold, certificateThreshold, validators := context.GetNextValidators()
	return &labi.InitGenesisStateResponse{
		Events:               eventLogger.Events(),
		PreCommitThreshold:   precommitThreshold,
		CertificateThreshold: certificateThreshold,
		NextValidators:       validators,
	}, nil
}

func (a *ABIHandler) InsertAssets(req *labi.InsertAssetsRequest) (*labi.InsertAssetsResponse, error) {
	if err := a.checkState(req.ContextID); err != nil {
		return nil, err
	}

	context := statemachine.NewInsertAssetsContext(
		a.ctx,
		a.logger,
		a.chainID,
		a.executionContext.diffStore,
		a.executionContext.moduleStore,
		a.executionContext.header,
		&blockchain.BlockAssets{},
		req.FinalizedHeight,
	)
	if err := a.stateMachine.InsertAssets(context); err != nil {
		return nil, err
	}
	return &labi.InsertAssetsResponse{
		Assets: context.Assets(),
	}, nil
}

func (a *ABIHandler) VerifyAssets(req *labi.VerifyAssetsRequest) (*labi.VerifyAssetsResponse, error) {
	// Remove genesis from memory
	a.genesisBlock = nil
	if err := a.checkState(req.ContextID); err != nil {
		return nil, err
	}
	context := statemachine.NewVerifyAssetsContext(
		a.ctx,
		a.logger,
		a.chainID,
		a.executionContext.diffStore,
		a.executionContext.header,
		req.Assets,
	)
	if err := a.stateMachine.VerifyAssets(context); err != nil {
		return nil, err
	}
	return &labi.VerifyAssetsResponse{}, nil
}

func (a *ABIHandler) BeforeTransactionsExecute(req *labi.BeforeTransactionsExecuteRequest) (*labi.BeforeTransactionsExecuteResponse, error) {
	if err := a.checkState(req.ContextID); err != nil {
		return nil, err
	}
	eventLogger := statemachine.NewEventLogger(a.executionContext.header.Height)
	context := statemachine.NewBeforeTransactionsExecuteContext(
		a.ctx,
		a.logger,
		a.chainID,
		a.executionContext.diffStore,
		eventLogger,
		a.executionContext.header,
		req.Assets,
		req.Consensus.CurrentValidators,
		req.Consensus.ImplyMaxPrevote,
		req.Consensus.MaxHeightCertified,
	)
	if err := a.stateMachine.BeforeExecute(context); err != nil {
		return nil, err
	}
	return &labi.BeforeTransactionsExecuteResponse{
		Events: eventLogger.Events(),
	}, nil
}

func (a *ABIHandler) AfterTransactionsExecute(req *labi.AfterTransactionsExecuteRequest) (*labi.AfterTransactionsExecuteResponse, error) {
	if err := a.checkState(req.ContextID); err != nil {
		return nil, err
	}
	for _, transaction := range req.Transactions {
		transaction.Init()
	}
	eventLogger := statemachine.NewEventLogger(a.executionContext.header.Height)
	context := statemachine.NewAfterTransactionsExecuteContext(
		a.ctx,
		a.logger,
		a.chainID,
		a.executionContext.diffStore,
		eventLogger,
		a.executionContext.header,
		req.Assets,
		req.Consensus.CurrentValidators,
		req.Consensus.ImplyMaxPrevote,
		req.Consensus.MaxHeightCertified,
		req.Transactions,
	)
	if err := a.stateMachine.AfterExecute(context); err != nil {
		return nil, err
	}
	precommitThreshold, certificateThreshold, validators := context.GetNextValidators()
	return &labi.AfterTransactionsExecuteResponse{
		Events:               eventLogger.Events(),
		PreCommitThreshold:   precommitThreshold,
		CertificateThreshold: certificateThreshold,
		NextValidators:       validators,
	}, nil
}

func (a *ABIHandler) VerifyTransaction(req *labi.VerifyTransactionRequest) (*labi.VerifyTransactionResponse, error) {
	var diffStore *diffdb.Database
	if a.executionContext == nil || bytes.Equal(a.executionContext.id, req.ContextID) {
		diffStore = diffdb.New(a.stateDB, StateDBPrefixState)
	} else {
		diffStore = a.executionContext.diffStore
	}
	req.Transaction.Init()
	context := statemachine.NewTransactionVerifyContext(
		a.ctx,
		a.logger,
		a.chainID,
		diffStore,
		req.Transaction,
	)
	result := a.stateMachine.VerifyTransaction(context)
	return &labi.VerifyTransactionResponse{
		Result: result.Code(),
	}, nil
}

func (a *ABIHandler) ExecuteTransaction(req *labi.ExecuteTransactionRequest) (*labi.ExecuteTransactionResponse, error) {
	var diffStore *diffdb.Database
	var header *blockchain.BlockHeader
	eventLogger := statemachine.NewEventLogger(a.executionContext.header.Height)
	if !req.DryRun {
		if err := a.checkState(req.ContextID); err != nil {
			return nil, err
		}
		header = a.executionContext.header
		diffStore = a.executionContext.diffStore
	} else {
		header = a.executionContext.header
		diffStore = diffdb.New(a.stateDB, StateDBPrefixState)
	}
	req.Transaction.Init()
	context := statemachine.NewTransactionExecuteContext(
		a.ctx,
		a.logger,
		a.chainID,
		diffStore,
		eventLogger,
		header,
		req.Assets,
		req.Consensus.CurrentValidators,
		req.Consensus.ImplyMaxPrevote,
		req.Consensus.MaxHeightCertified,
		req.Transaction,
	)
	result := a.stateMachine.ExecuteTransaction(context)
	return &labi.ExecuteTransactionResponse{
		Events: eventLogger.Events(),
		Result: result.Code(),
	}, nil
}

func (a *ABIHandler) Commit(req *labi.CommitRequest) (*labi.CommitResponse, error) {
	if err := a.checkState(req.ContextID); err != nil {
		return nil, err
	}
	batch := a.stateDB.NewBatch()
	stateBatch := newStateBatch(batch)
	diff := a.executionContext.diffStore.Commit(stateBatch)
	encodedDiff := diff.Encode()
	batch.Set(bytes.Join(StateDBPrefixDiff, bytes.FromUint32(a.executionContext.header.Height)), encodedDiff)
	smtDB := batchdb.NewWithPrefix(a.stateDB, batch, StateDBPrefixTree)
	tree := smt.NewTrie(req.StateRoot, stateTreeKeySize)
	root, err := tree.Update(smtDB, stateBatch.keys, stateBatch.values)
	if err != nil {
		return nil, err
	}
	if len(req.ExpectedStateRoot) > 0 && !bytes.Equal(root, req.ExpectedStateRoot) {
		return nil, fmt.Errorf("state root %s does not match with expected state root %s", codec.Hex(root), req.ExpectedStateRoot)
	}
	if req.DryRun {
		return &labi.CommitResponse{
			StateRoot: root,
		}, nil
	}
	currentStateDB := batchdb.NewWithPrefix(a.stateDB, batch, StateDBPrefixTreeState)
	currentState := bytes.Join(bytes.FromUint32(a.executionContext.header.Height), root)
	currentStateDB.Set([]byte{}, currentState)

	a.stateDB.Write(batch)
	return &labi.CommitResponse{
		StateRoot: root,
	}, nil
}

func (a *ABIHandler) Revert(req *labi.RevertRequest) (*labi.RevertResponse, error) {
	if err := a.checkState(req.ContextID); err != nil {
		return nil, err
	}
	root, err := a.revert(a.executionContext.header.Height, req.StateRoot, req.ExpectedStateRoot)
	if err != nil {
		return nil, err
	}
	return &labi.RevertResponse{
		StateRoot: root,
	}, nil
}

func (a *ABIHandler) Finalize(req *labi.FinalizeRequest) (*labi.FinalizeResponse, error) {
	if req.FinalizedHeight == 0 {
		return &labi.FinalizeResponse{}, nil
	}
	batch := a.stateDB.NewBatch()
	keys := a.stateDB.IterateKey(StateDBPrefixDiff, -1, false)

	for _, key := range keys {
		heightByte := key[len(StateDBPrefixDiff):]
		height := bytes.ToUint32(heightByte)
		if height < req.FinalizedHeight {
			batch.Del(key)
		}
	}
	a.stateDB.Write(batch)
	return &labi.FinalizeResponse{}, nil
}

func (a *ABIHandler) Clear(req *labi.ClearRequest) (*labi.ClearResponse, error) {
	a.executionContext = nil
	return &labi.ClearResponse{}, nil
}

func (a *ABIHandler) GetMetadata(req *labi.MetadataRequest) (*labi.MetadataResponse, error) {
	return nil, nil
}

func (a *ABIHandler) Query(req *labi.QueryRequest) (*labi.QueryResponse, error) {
	splitMethod := strings.Split(req.Method, querySeparator)
	if len(splitMethod) != 2 {
		return nil, fmt.Errorf("invalid method %s", req.Method)
	}
	req.Header.Init()
	namespace, endpoint := splitMethod[0], splitMethod[1]
	for _, mod := range a.modules {
		if namespace == mod.Name() {
			handler, exist := mod.Endpoint().Get()[endpoint]
			if !exist {
				return nil, fmt.Errorf("method %s does not exist for %s", endpoint, namespace)
			}
			reader := a.stateDB.NewReader()
			defer reader.Close()
			req := statemachine.NewEndpointRequest(
				a.ctx,
				a.logger,
				diffdb.New(reader, StateDBPrefixState),
				a.chainID,
				req.Header,
				req.Params,
			)

			resp := rpc.NewEndpointResponseWriter()
			handler(resp, req)
			result := resp.Result()
			if result.Err() != nil {
				return nil, result.Err()
			}
			jsonData, err := result.JSONData()
			if err != nil {
				return nil, err
			}
			return &labi.QueryResponse{
				Data: jsonData,
			}, nil
		}
	}
	return nil, fmt.Errorf("method %s not found", req.Method)
}

func (a *ABIHandler) Prove(req *labi.ProveRequest) (*labi.ProveResponse, error) { return nil, nil }

func (a *ABIHandler) checkState(id codec.Hex) error {
	if a.executionContext == nil || !bytes.Equal(a.executionContext.id, id) {
		return fmt.Errorf("invalid context id %s. context is not initialized or different", id)
	}
	return nil
}

func (a *ABIHandler) revert(height uint32, stateRoot, expectedStateRoot codec.Hex) ([]byte, error) {
	encodedDiff, exist := a.stateDB.Get(bytes.Join(StateDBPrefixDiff, bytes.FromUint32(height)))
	if !exist {
		return nil, fmt.Errorf("diff at height %d does not exist", height)
	}
	diff := &diffdb.Diff{}
	if err := diff.Decode(encodedDiff); err != nil {
		return nil, err
	}
	batch := a.stateDB.NewBatch()
	stateBatch := newStateBatch(batch)
	diffStore := diffdb.New(a.stateDB, StateDBPrefixState)
	diffStore.RevertDiff(stateBatch, diff)
	tree := smt.NewTrie(stateRoot, stateTreeKeySize)
	smtDB := batchdb.NewWithPrefix(a.stateDB, batch, StateDBPrefixTree)
	root, err := tree.Update(smtDB, stateBatch.keys, stateBatch.values)
	if err != nil {
		return nil, err
	}
	if len(expectedStateRoot) > 0 && !bytes.Equal(root, expectedStateRoot) {
		return nil, fmt.Errorf("state root %s does not match with expected state root %s", codec.Hex(root), expectedStateRoot)
	}
	currentStateDB := batchdb.NewWithPrefix(a.stateDB, batch, StateDBPrefixTreeState)
	currentState := bytes.Join(bytes.FromUint32(a.executionContext.header.Height-1), root)
	currentStateDB.Set(emptyBytes, currentState)
	a.stateDB.Write(batch)
	return root, nil
}
