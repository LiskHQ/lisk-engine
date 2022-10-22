package statemachine

import (
	"context"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func ModuleStorePrefix(moduleID uint32, storePrifx uint16) []byte {
	return bytes.Join(bytes.FromUint32(moduleID), bytes.FromUint16(storePrifx))
}

type commonProcessingContext struct {
	ctx     context.Context
	logger  log.Logger
	chainID []byte
}

func (c *commonProcessingContext) Context() context.Context { return c.ctx }
func (c *commonProcessingContext) Logger() log.Logger       { return c.logger }
func (c *commonProcessingContext) ChainID() []byte          { return c.chainID }

type StateAccessContext interface {
	APIContext() ImmutableAPIContext
	GetStore(moduleID uint32, bucketID uint16) ImmutableStore
}

type stateAccessContext struct {
	diffStore *diffdb.Database
}

func (c *stateAccessContext) GetStore(moduleID uint32, bucketID uint16) ImmutableStore {
	return c.diffStore.WithPrefix(ModuleStorePrefix(moduleID, bucketID))
}

func (c *stateAccessContext) APIContext() ImmutableAPIContext {
	return &immutableAPIContext{diffStore: c.diffStore}
}

type stateChangeContext struct {
	diffStore *diffdb.Database
	eventLogs *EventLogger
}

func (c *stateChangeContext) EventQueue() EventAdder {
	return c.eventLogs
}

func (c *stateChangeContext) Snapshot() int {
	return c.diffStore.Snapshot()
}

func (c *stateChangeContext) RestoreSnapshot(snapshotID int) error {
	return c.diffStore.RestoreSnapshot(snapshotID)
}

func (c *stateChangeContext) GetStore(moduleID uint32, bucketID uint16) Store {
	return c.diffStore.WithPrefix(ModuleStorePrefix(moduleID, bucketID))
}

func (c *stateChangeContext) APIContext() APIContext {
	return &apiContext{diffStore: c.diffStore}
}

func (c *stateChangeContext) Immutable() StateAccessContext {
	return &stateAccessContext{
		diffStore: c.diffStore,
	}
}

type blockHeaderContext struct {
	blockHeader *blockchain.BlockHeader
	blockAssets blockchain.BlockAssets
}

func (c *blockHeaderContext) BlockHeader() blockchain.ReadableBlockHeader {
	return c.blockHeader.Readonly()
}

func (c *blockHeaderContext) BlockAssets() blockchain.ReadableBlockAssets {
	return c.blockAssets
}

type consensusContext struct {
	currentValidators  []*labi.Validator
	implyMaxPrevote    bool
	maxHeightCertified uint32
}

func (c *consensusContext) ImplyMaxPrevote() bool {
	return c.implyMaxPrevote
}

func (c *consensusContext) MaxHeightCertified() uint32 {
	return c.maxHeightCertified
}

func (c *consensusContext) CurrentValidators() []*labi.Validator {
	copied := make([]*labi.Validator, len(c.currentValidators))
	copy(copied, c.currentValidators)
	return copied
}

type consensusSetter struct {
	preCommitThreshold   uint64
	certificateThreshold uint64
	nextValidators       []*labi.Validator
}

func (c *consensusSetter) SetNextValidators(preCommitThreshold, certificateThreshold uint64, validators []*labi.Validator) {
	c.preCommitThreshold = preCommitThreshold
	c.certificateThreshold = certificateThreshold
	c.nextValidators = validators
}

func (c *consensusSetter) GetNextValidators() (uint64, uint64, []*labi.Validator) {
	return c.preCommitThreshold, c.certificateThreshold, c.nextValidators
}

type InsertAssetsContext struct {
	commonProcessingContext
	diffStore       *diffdb.Database
	generatorStore  *diffdb.Database
	partialHeader   *blockchain.BlockHeader
	blockAssets     *blockchain.BlockAssets
	finazliedHeight uint32
}

func (c *InsertAssetsContext) BlockHeader() blockchain.ReadableBlockHeader {
	return c.partialHeader.Readonly()
}
func (c *InsertAssetsContext) BlockAssets() blockchain.WritableBlockAssets {
	return c.blockAssets
}
func (c *InsertAssetsContext) SetAsset(module string, data []byte) {
	c.blockAssets.SetAsset(module, data)
}
func (c *InsertAssetsContext) GetGeneratorStore(moduleID uint32, storePrefix uint16) Store {
	return c.generatorStore.WithPrefix(ModuleStorePrefix(moduleID, storePrefix))
}
func (c *InsertAssetsContext) APIContext() ImmutableAPIContext {
	return NewAPIContext(c.diffStore).Immutable()
}
func (c *InsertAssetsContext) GetStore(moduleID uint32, bucketID uint16) ImmutableStore {
	return c.diffStore.WithPrefix(ModuleStorePrefix(moduleID, bucketID))
}
func (c *InsertAssetsContext) FinalizedHeight() uint32 {
	return c.finazliedHeight
}

func (c *InsertAssetsContext) Assets() blockchain.BlockAssets {
	return *c.blockAssets
}

func NewInsertAssetsContext(
	ctx context.Context,
	logger log.Logger,
	chainID []byte,
	diffStore *diffdb.Database,
	generatorStore *diffdb.Database,
	partialHeader *blockchain.BlockHeader,
	blockAssets *blockchain.BlockAssets,
	finalizedHeight uint32,
) *InsertAssetsContext {
	return &InsertAssetsContext{
		commonProcessingContext: commonProcessingContext{
			ctx:     ctx,
			logger:  logger,
			chainID: chainID,
		},
		diffStore:       diffStore,
		generatorStore:  generatorStore,
		partialHeader:   partialHeader,
		blockAssets:     blockAssets,
		finazliedHeight: finalizedHeight,
	}
}

type VerifyAssetsContext struct {
	commonProcessingContext
	stateAccessContext
	header      *blockchain.BlockHeader
	blockAssets blockchain.BlockAssets
}

func (c *VerifyAssetsContext) BlockHeader() blockchain.SealedBlockHeader {
	return c.header.Readonly()
}

func (c *VerifyAssetsContext) BlockAssets() blockchain.ReadableBlockAssets {
	return c.blockAssets
}

func NewVerifyAssetsContext(
	ctx context.Context,
	logger log.Logger,
	chainID []byte,
	diffStore *diffdb.Database,
	header *blockchain.BlockHeader,
	blockAssets blockchain.BlockAssets,
) *VerifyAssetsContext {
	return &VerifyAssetsContext{
		commonProcessingContext: commonProcessingContext{
			ctx:     ctx,
			logger:  logger,
			chainID: chainID,
		},
		stateAccessContext: stateAccessContext{
			diffStore: diffStore,
		},
		header:      header,
		blockAssets: blockAssets,
	}
}

type TransactionVerifyContext struct {
	commonProcessingContext
	stateAccessContext
	transaction *blockchain.Transaction
}

func (c *TransactionVerifyContext) Transaction() blockchain.FrozenTransaction {
	return c.transaction.Freeze()
}

func NewTransactionVerifyContext(
	ctx context.Context,
	logger log.Logger,
	chainID []byte,
	diffStore *diffdb.Database,
	transaction *blockchain.Transaction,
) *TransactionVerifyContext {
	return &TransactionVerifyContext{
		commonProcessingContext: commonProcessingContext{
			ctx:     ctx,
			logger:  logger,
			chainID: chainID,
		},
		stateAccessContext: stateAccessContext{
			diffStore: diffStore,
		},
		transaction: transaction,
	}
}

type TransactionExecuteContext struct {
	commonProcessingContext
	stateChangeContext
	blockHeaderContext
	transaction *blockchain.Transaction
	consensusContext
}

func (c *TransactionExecuteContext) Transaction() blockchain.FrozenTransaction {
	return c.transaction.Freeze()
}

func NewTransactionExecuteContext(
	ctx context.Context,
	logger log.Logger,
	chainID []byte,
	diffStore *diffdb.Database,
	eventLogger *EventLogger,
	blockHeader *blockchain.BlockHeader,
	blockAssets blockchain.BlockAssets,
	currentValidators []*labi.Validator,
	implyMaxPrevote bool,
	maxHeightCertified uint32,
	transaction *blockchain.Transaction,
) *TransactionExecuteContext {
	return &TransactionExecuteContext{
		commonProcessingContext: commonProcessingContext{
			ctx:     ctx,
			logger:  logger,
			chainID: chainID,
		},
		stateChangeContext: stateChangeContext{
			diffStore: diffStore,
			eventLogs: eventLogger,
		},
		transaction: transaction,
		blockHeaderContext: blockHeaderContext{
			blockAssets: blockAssets,
			blockHeader: blockHeader,
		},
		consensusContext: consensusContext{
			currentValidators:  currentValidators,
			implyMaxPrevote:    implyMaxPrevote,
			maxHeightCertified: maxHeightCertified,
		},
	}
}

type BeforeTransactionsExecuteContext struct {
	commonProcessingContext
	stateChangeContext
	blockHeaderContext
	consensusContext
}

func NewBeforeTransactionsExecuteContext(
	ctx context.Context,
	logger log.Logger,
	chainID []byte,
	store *diffdb.Database,
	eventLogger *EventLogger,
	partialHeader *blockchain.BlockHeader,
	blockAssets blockchain.BlockAssets,
	currentValidators []*labi.Validator,
	implyMaxPrevote bool,
	maxHeightCertified uint32,
) *BeforeTransactionsExecuteContext {
	return &BeforeTransactionsExecuteContext{
		commonProcessingContext: commonProcessingContext{
			ctx:     ctx,
			logger:  logger,
			chainID: chainID,
		},
		stateChangeContext: stateChangeContext{
			diffStore: store,
			eventLogs: eventLogger,
		},
		blockHeaderContext: blockHeaderContext{
			blockHeader: partialHeader,
			blockAssets: blockAssets,
		},
		consensusContext: consensusContext{
			currentValidators:  currentValidators,
			implyMaxPrevote:    implyMaxPrevote,
			maxHeightCertified: maxHeightCertified,
		},
	}
}

type AfterTransactionsExecuteContext struct {
	BeforeTransactionsExecuteContext
	consensusSetter
	transactions []*blockchain.Transaction
}

func (c *AfterTransactionsExecuteContext) Transactions() []blockchain.FrozenTransaction {
	txs := make([]blockchain.FrozenTransaction, len(c.transactions))
	for i, tx := range c.transactions {
		txs[i] = tx.Freeze()
	}
	return txs
}

func NewAfterTransactionsExecuteContext(
	ctx context.Context,
	logger log.Logger,
	chainID []byte,
	store *diffdb.Database,
	eventLogger *EventLogger,
	partialHeader *blockchain.BlockHeader,
	blockAssets blockchain.BlockAssets,
	currentValidators []*labi.Validator,
	implyMaxPrevote bool,
	maxHeightCertified uint32,
	transactions []*blockchain.Transaction,
) *AfterTransactionsExecuteContext {
	return &AfterTransactionsExecuteContext{
		BeforeTransactionsExecuteContext: BeforeTransactionsExecuteContext{
			commonProcessingContext: commonProcessingContext{
				ctx:     ctx,
				logger:  logger,
				chainID: chainID,
			},
			stateChangeContext: stateChangeContext{
				diffStore: store,
				eventLogs: eventLogger,
			},
			blockHeaderContext: blockHeaderContext{
				blockHeader: partialHeader,
				blockAssets: blockAssets,
			},
			consensusContext: consensusContext{
				currentValidators:  currentValidators,
				implyMaxPrevote:    implyMaxPrevote,
				maxHeightCertified: maxHeightCertified,
			},
		},
		transactions: transactions,
		consensusSetter: consensusSetter{
			preCommitThreshold:   0,
			certificateThreshold: 0,
			nextValidators:       []*labi.Validator{},
		},
	}
}

type GenesisBlockProcessingContext struct {
	ctx    context.Context
	logger log.Logger
	header *blockchain.BlockHeader
	assets blockchain.BlockAssets
	stateChangeContext
	consensusSetter
}

func (c *GenesisBlockProcessingContext) Context() context.Context { return c.ctx }
func (c *GenesisBlockProcessingContext) Logger() log.Logger       { return c.logger }

func (c *GenesisBlockProcessingContext) BlockHeader() blockchain.SealedBlockHeader {
	return c.header.Readonly()
}
func (c *GenesisBlockProcessingContext) BlockAssets() blockchain.ReadableBlockAssets {
	return c.assets
}

func NewGenesisBlockProcessingContext(
	ctx context.Context,
	logger log.Logger,
	store *diffdb.Database,
	eventLogger *EventLogger,
	header *blockchain.BlockHeader,
	assets blockchain.BlockAssets,
) *GenesisBlockProcessingContext {
	return &GenesisBlockProcessingContext{
		ctx:    ctx,
		logger: logger,
		stateChangeContext: stateChangeContext{
			diffStore: store,
			eventLogs: eventLogger,
		},
		consensusSetter: consensusSetter{
			preCommitThreshold:   0,
			certificateThreshold: 0,
			nextValidators:       []*labi.Validator{},
		},
		header: header,
		assets: assets,
	}
}

type ImmutableAPIContext interface {
	GetStore(id uint32, bucketrID uint16) ImmutableStore
}

type EventAdder interface {
	Add(module, name string, data []byte, topics []codec.Hex) error
	AddUnrevertible(module, name string, data []byte, topics []codec.Hex) error
}

type APIContext interface {
	GetStore(id uint32, bucketrID uint16) Store
	Immutable() ImmutableAPIContext
}

func NewAPIContext(diffStore *diffdb.Database) APIContext {
	return &apiContext{
		diffStore: diffStore,
	}
}

type apiContext struct {
	diffStore *diffdb.Database
}

func (c *apiContext) GetStore(moduleID uint32, bucketID uint16) Store {
	return c.diffStore.WithPrefix(ModuleStorePrefix(moduleID, bucketID))
}

func (c *apiContext) Immutable() ImmutableAPIContext {
	return &immutableAPIContext{diffStore: c.diffStore}
}

type immutableAPIContext struct {
	diffStore *diffdb.Database
}

func (c *immutableAPIContext) GetStore(moduleID uint32, bucketID uint16) ImmutableStore {
	return c.diffStore.WithPrefix(ModuleStorePrefix(moduleID, bucketID))
}
