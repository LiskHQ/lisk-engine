// Package generator implements block generation.
package generator

import (
	"container/heap"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/liskbft"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/trie/rmt"
	"github.com/LiskHQ/lisk-engine/pkg/txpool"
)

const (
	generatorCheckInterval = 1 * time.Second
	blockVersion           = 2
)

var (
	GeneratorDBPrefixGeneratedInfo = []byte{0}
	GeneratorDBPrefixKeys          = []byte{1}
)

type Consensus interface {
	Syncing() bool
	AddInternal(block *blockchain.Block)
	GetAggregateCommit() (*blockchain.AggregateCommit, error)
	Certify(from, to uint32, address codec.Lisk32, blsPrivateKey []byte) error
	Subscribe(topic string) <-chan interface{}
	GetSlotNumber(unixTime uint32) int
	GetSlotTime(slot int) uint32
	GetBFTHeights(context *diffdb.Database) (uint32, uint32, uint32, error)
	GetBFTParameters(context *diffdb.Database, height uint32) (*liskbft.BFTParams, error)
	GetGeneratorKeys(context *diffdb.Database, height uint32) (liskbft.Generators, error)
	HeaderHasPriority(context *diffdb.Database, header blockchain.SealedBlockHeader, height, maxHeightPrevoted, maxHeightPreviouslyForged uint32) (bool, error)
	ImpliesMaximalPrevotes(context *diffdb.Database, blockHeader blockchain.ReadableBlockHeader) (bool, error)
	BFTBeforeTransactionsExecute(blockHeader blockchain.SealedBlockHeader, diffStore *diffdb.Database) error
}

type GeneratorParams struct {
	Consensus Consensus
	ABI       labi.ABI
	Pool      *txpool.TransactionPool
	Chain     *blockchain.Chain
}

func NewGenerator(params *GeneratorParams) *Generator {
	return &Generator{
		consensus:   params.Consensus,
		abi:         params.ABI,
		pool:        params.Pool,
		chain:       params.Chain,
		enabledKeys: map[string]*PlainKeys{},
	}
}

type Generator struct {
	// dependencies
	abi       labi.ABI
	consensus Consensus
	pool      *txpool.TransactionPool
	chain     *blockchain.Chain
	// init
	ctx           context.Context
	blockchainDB  *db.DB
	generatorDB   *db.DB
	logger        log.Logger
	cfg           *config.Config
	waitThreshold int

	// instantiate
	checkLoop   *time.Ticker
	enabledKeys map[string]*PlainKeys
}

type GeneratorInitParams struct {
	CTX          context.Context
	Cfg          *config.Config
	Logger       log.Logger
	BlockchainDB *db.DB
	GeneratorDB  *db.DB
}

func (g *Generator) Init(params *GeneratorInitParams) error {
	g.ctx = params.CTX
	g.cfg = params.Cfg
	g.checkLoop = time.NewTicker(generatorCheckInterval)
	g.logger = params.Logger
	g.generatorDB = params.GeneratorDB
	g.blockchainDB = params.BlockchainDB
	g.waitThreshold = int(g.cfg.Genesis.BlockTime) / 5
	if err := g.saveGeneratorsFromFile(); err != nil {
		return err
	}
	if err := g.loadGenerator(); err != nil {
		return err
	}
	return nil
}

func (g *Generator) Start() {
	onNewBlock := g.consensus.Subscribe(consensus.EventBlockNew)
	onDeleteBlock := g.consensus.Subscribe(consensus.EventBlockDelete)
	onFinalizeBlock := g.consensus.Subscribe(consensus.EventBlockFinalize)
	for {
		select {
		case msg := <-onNewBlock:
			g.onNewBlock(msg)
		case msg := <-onDeleteBlock:
			g.onDeleteBlock(msg)
		case msg := <-onFinalizeBlock:
			g.onFinalizeBlock(msg)
		case <-g.checkLoop.C:
			g.forge()
		case <-g.ctx.Done():
			return
		}
	}
}

func (g *Generator) EnableGeneration(address codec.Lisk32, keys *PlainKeys) {
	g.enabledKeys[string(address)] = keys
}

func (g *Generator) DisableGeneration(address codec.Lisk32) {
	delete(g.enabledKeys, string(address))
}

func (g *Generator) IsGenerationEnabled(address codec.Lisk32) bool {
	_, ok := g.enabledKeys[string(address)]
	return ok
}

func (g *Generator) forge() {
	diffStore := diffdb.New(g.blockchainDB, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	now := time.Now().Unix()
	shouldForge, err := g.shouldForge(now)
	if err != nil {
		g.logger.Errorf("Fail to determine whether to generate a block by %v", err)
		return
	}
	if !shouldForge {
		return
	}

	generators, err := g.consensus.GetGeneratorKeys(diffStore, g.chain.LastBlock().Header.Height+1)
	if err != nil {
		g.logger.Errorf("Fail to get generater by %v", err)
		return
	}
	generator, err := generators.AtTimestamp(g.consensus, uint32(now))
	if err != nil {
		g.logger.Errorf("Fail to get generater by %v", err)
		return
	}
	keys, enabled := g.enabledKeys[string(generator.Address())]
	if !enabled {
		return
	}
	generatorDB := diffdb.New(g.generatorDB, GeneratorDBPrefixGeneratedInfo)
	// create context
	partialHeader, err := g.initBlockHeader(generatorDB, generator.Address(), diffStore)
	if err != nil {
		g.logger.Errorf("Fail to create initial block for generation with %v", err)
		return
	}
	abi, err := newBlockExecuteABI(g.abi, g.chain.ChainID(), diffStore, g.consensus, partialHeader)
	if err != nil {
		g.logger.Errorf("Fail to create ABI for generation with %v", err)
		return
	}
	defer abi.Clear() //nolint:errcheck // TODO: update to return err

	txs := g.pool.GetProcessable()
	assets, err := abi.InsertAssets()
	if err != nil {
		g.logger.Errorf("Fail to insert assets by %v", err)
		return
	}

	if err := abi.BeforeTransactionsExecute(diffStore, partialHeader.Readonly(), assets); err != nil {
		g.logger.Errorf("Fail to execute before hooks with %v", err)
		return
	}

	selectedTxs := []*blockchain.Transaction{}
	if len(txs) != 0 {
		selectedTxs, err = g.selectTransactionsByFee(abi, partialHeader, assets, txs, int(g.cfg.Genesis.MaxTransactionsSize))
		if err != nil {
			g.logger.Errorf("Fail to select transactions with %v", err)
			return
		}
	}
	includingTxs := g.limitTransactionsWithSize(int(g.cfg.Genesis.MaxTransactionsSize), selectedTxs)
	if err != nil {
		g.logger.Errorf("Fail to get next reward with %v", err)
		return
	}

	// exekcute after
	if err := abi.AfterTransactionsExecute(assets, includingTxs); err != nil {
		g.logger.Errorf("Fail to execute after hooks with %v", err)
		return
	}

	stateRoot, err := abi.Commit(g.chain.LastBlock().Header.StateRoot)
	if err != nil {
		g.logger.Errorf("Fail to commit to state with %v", err)
		return
	}

	// calculate sealing properties
	signedBlock, err := g.sealBlock(
		partialHeader,
		includingTxs,
		assets,
		abi.Events(),
		stateRoot,
		keys.GeneratorPrivateKey,
		diffStore,
	)
	if err != nil {
		g.logger.Errorf("Fail to seal block with %v", err)
		return
	}

	previousInfo := &GeneratorInfo{
		Height:             signedBlock.Header.Height,
		MaxHeightPrevoted:  signedBlock.Header.MaxHeightPrevoted,
		MaxHeightGenerated: signedBlock.Header.MaxHeightGenerated,
	}
	encodedPreviousInfo := previousInfo.Encode()
	previousInfoStore := generatorDB.WithPrefix(GeneratorDBPrefixGeneratedInfo)
	previousInfoStore.Set(signedBlock.Header.GeneratorAddress, encodedPreviousInfo)
	batch := g.generatorDB.NewBatch()
	generatorDB.Commit(batch)
	g.generatorDB.Write(batch)
	g.logger.Infof("Generator %s forged block at height %d with %d transactions",
		signedBlock.Header.GeneratorAddress.String(),
		signedBlock.Header.Height,
		len(signedBlock.Transactions),
	)
	// save generatorDB
	g.consensus.AddInternal(signedBlock)
}

func (g *Generator) shouldForge(now int64) (bool, error) {
	if g.consensus.Syncing() {
		return false, nil
	}
	currentTimeslot := g.consensus.GetSlotNumber(uint32(now))
	lastTimeslot := g.consensus.GetSlotNumber(g.chain.LastBlock().Header.Timestamp)
	if currentTimeslot == lastTimeslot {
		return false, nil
	}
	beginningTimeslot := g.consensus.GetSlotTime(currentTimeslot)
	if lastTimeslot < currentTimeslot-1 && now <= int64(beginningTimeslot)+int64(g.waitThreshold) {
		return false, nil
	}
	return true, nil
}

func (g *Generator) selectTransactionsByFee(abi *stateExecuter, header *blockchain.BlockHeader, assets blockchain.BlockAssets, transactions []*blockchain.Transaction, maxSize int) ([]*blockchain.Transaction, error) {
	transactionBySender := getSortedTransactionMapByNonce(transactions)
	// payloadSize := 0
	priorityTx := &FeePriorityTransactions{}
	heap.Init(priorityTx)
	for _, val := range transactionBySender {
		lowestNonce := val[0]
		heap.Push(priorityTx, lowestNonce)
	}
	totalSize := 0
	resultTransaction := []*blockchain.Transaction{}
	for len(transactionBySender) > 0 {
		nextTx := heap.Pop(priorityTx).(*TransactionWithFeePriority)
		if nextTx.Size()+totalSize > maxSize {
			break
		}
		// Verify the transaction against current state before execution
		if err := abi.VerifyTransaction(nextTx.Transaction); err != nil {
			delete(transactionBySender, string(nextTx.SenderAddress()))
			continue
		}
		if err := abi.ExecuteTransaction(header, assets, nextTx.Transaction); err != nil {
			delete(transactionBySender, string(nextTx.SenderAddress()))
			continue
		}
		resultTransaction = append(resultTransaction, nextTx.Transaction)
		totalSize += nextTx.Size()
		sender := nextTx.SenderAddress()
		fromSenderTx := transactionBySender[string(sender)]
		transactionBySender[string(sender)] = fromSenderTx[1:]
		if len(transactionBySender[string(sender)]) == 0 {
			delete(transactionBySender, string(sender))
			continue
		}
		nextSmallest := transactionBySender[string(sender)][0]
		heap.Push(priorityTx, nextSmallest)
	}
	return resultTransaction, nil
}

func (g *Generator) limitTransactionsWithSize(maxTransactionsLength int, txs []*blockchain.Transaction) []*blockchain.Transaction {
	totalSize := 0
	selected := []*blockchain.Transaction{}
	for _, tx := range txs {
		if tx.Size()+totalSize > maxTransactionsLength {
			break
		}
		totalSize += tx.Size()
		selected = append(selected, tx)
	}
	return selected
}

func (g *Generator) saveGeneratorsFromFile() error {
	if g.cfg.Generator.Keys.FromFile == "" {
		return nil
	}
	keysFilePath := g.cfg.Generator.Keys.FromFile
	if !filepath.IsAbs(keysFilePath) {
		var err error
		keysFilePath, err = filepath.Abs(filepath.Join(g.cfg.System.DataPath, g.cfg.Generator.Keys.FromFile))
		if err != nil {
			return err
		}
	}
	keysFileData, err := os.ReadFile(keysFilePath)
	if err != nil {
		return err
	}
	keysFile := &KeysFile{}
	if err := json.Unmarshal(keysFileData, keysFile); err != nil {
		return err
	}
	keysStore := diffdb.New(g.generatorDB, GeneratorDBPrefixKeys)
	for _, rawKeys := range keysFile.Keys {
		if err := rawKeys.Encrypted.Validate(); err == nil {
			keys := &Keys{
				Address: rawKeys.Address,
				Type:    KeyTypeEncrypted,
				Data:    rawKeys.Encrypted.Encode(),
			}
			g.logger.Infof("Saving keys for address %s", rawKeys.Address)
			keysStore.Set(rawKeys.Address, keys.Encode())
			continue
		}
		if err := rawKeys.Plain.Validate(); err == nil {
			keys := &Keys{
				Address: rawKeys.Address,
				Type:    KeyTypePlain,
				Data:    rawKeys.Plain.Encode(),
			}
			g.logger.Infof("Saving keys for address %s", rawKeys.Address)
			keysStore.Set(rawKeys.Address, keys.Encode())
			continue
		}
		g.logger.Errorf("Invalid key for %s", rawKeys.Address)
	}
	batch := g.generatorDB.NewBatch()
	keysStore.Commit(batch)
	g.generatorDB.Write(batch)
	return nil
}

func (g *Generator) loadGenerator() error {
	keysStore := diffdb.New(g.generatorDB, GeneratorDBPrefixKeys)
	keysList := keysStore.Iterate([]byte{}, -1, false)

	for _, encodedKeysPair := range keysList {
		keys := &Keys{}
		if err := keys.Decode(encodedKeysPair.Value()); err != nil {
			return err
		}
		if keys.IsPlain() {
			plainKeys, err := keys.PlainKeys()
			if err != nil {
				return err
			}
			keys.Address = encodedKeysPair.Key()
			g.EnableGeneration(encodedKeysPair.Key(), plainKeys)
		}
	}
	return nil
}

func (g *Generator) onNewBlock(msg interface{}) {
	eventMsg, ok := msg.(*consensus.EventBlockNewMessage)
	if !ok {
		g.logger.Errorf("Invalid new block message")
		return
	}
	for _, tx := range eventMsg.Block.Transactions {
		g.pool.Remove(tx.ID)
		g.logger.Debugf("Removing transaction %s from transaction pool", tx.ID)
	}
}

func (g *Generator) onDeleteBlock(msg interface{}) {
	eventMsg, ok := msg.(*consensus.EventBlockDeleteMessage)
	if !ok {
		g.logger.Errorf("Invalid delete block message")
		return
	}
	for _, tx := range eventMsg.Block.Transactions {
		if ok := g.pool.Add(tx); ok {
			g.logger.Debugf("Adding transaction %s back to transaction pool", tx.ID)
		} else {
			g.logger.Debugf("Fail to add transaction %s back to transaction pool", tx.ID)
		}
	}
}

func (g *Generator) onFinalizeBlock(msg interface{}) {
	eventMsg, ok := msg.(*consensus.EventBlockFinalizeMessage)
	if !ok {
		g.logger.Errorf("Invalid finalize block message")
		return
	}
	key, exist := g.enabledKeys[string(eventMsg.Trigger.GeneratorAddress)]
	if !exist {
		return
	}
	if err := g.consensus.Certify(eventMsg.Original, eventMsg.Next, eventMsg.Trigger.GeneratorAddress, key.BLSPrivateKey); err != nil {
		g.logger.Info("Finalize block received", eventMsg.Next)
		return
	}
	g.logger.Infof("Validator %s certified blocks from %d to %d", eventMsg.Trigger.GeneratorAddress, eventMsg.Original, eventMsg.Next)
}

func (g *Generator) initBlockHeader(
	generatorDB *diffdb.Database,
	generatorAddress codec.Lisk32,
	diffStore *diffdb.Database,
) (*blockchain.BlockHeader, error) {
	lastBlock := g.chain.LastBlock()
	nextHeight := lastBlock.Header.Height + 1

	maxHeightPrevoted, _, _, err := g.consensus.GetBFTHeights(diffStore)
	if err != nil {
		return nil, err
	}
	prevInfoStore := generatorDB.WithPrefix(GeneratorDBPrefixGeneratedInfo)
	previousInfo := &GeneratorInfo{}
	infoBytes, exist := prevInfoStore.Get(generatorAddress)
	if exist {
		if err := previousInfo.Decode(infoBytes); err != nil {
			return nil, err
		}
	}

	nextInfo := &GeneratorInfo{
		Height:             nextHeight,
		MaxHeightPrevoted:  maxHeightPrevoted,
		MaxHeightGenerated: previousInfo.Height,
	}
	encodedNextInfo := nextInfo.Encode()
	prevInfoStore.Set(generatorAddress, encodedNextInfo)

	aggregateCommit, err := g.consensus.GetAggregateCommit()
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	partialHeader := &blockchain.BlockHeader{
		Version:            blockVersion,
		Timestamp:          uint32(now),
		Height:             nextHeight,
		MaxHeightPrevoted:  maxHeightPrevoted,
		MaxHeightGenerated: previousInfo.Height,
		PreviousBlockID:    lastBlock.Header.ID,
		GeneratorAddress:   generatorAddress,
		AggregateCommit:    aggregateCommit,
	}
	return partialHeader, nil
}

func (g *Generator) sealBlock(
	partialHeader *blockchain.BlockHeader,
	txs []*blockchain.Transaction,
	blockAssets blockchain.BlockAssets,
	events []*blockchain.Event,
	stateRoot codec.Hex,
	privateKey []byte,
	diffStore *diffdb.Database,
) (*blockchain.Block, error) {
	blockAssets.Sort()
	txIDs := make([][]byte, len(txs))
	for i, tx := range txs {
		txIDs[i] = tx.ID
	}

	params, err := g.consensus.GetBFTParameters(diffStore, partialHeader.Height+1)
	if err != nil {
		return nil, err
	}
	partialHeader.ValidatorsHash = params.ValidatorsHash()
	partialHeader.TransactionRoot = rmt.CalculateRoot(txIDs)
	assetRoot := blockAssets.GetRoot()
	partialHeader.AssetRoot = assetRoot
	eventRoot, err := blockchain.CalculateEventRoot(events)
	if err != nil {
		return nil, err
	}
	partialHeader.EventRoot = eventRoot
	partialHeader.StateRoot = stateRoot

	partialHeader.Sign(g.chain.ChainID(), privateKey)

	signedBlock := &blockchain.Block{
		Header:       partialHeader,
		Assets:       blockAssets,
		Transactions: txs,
	}
	return signedBlock, nil
}
