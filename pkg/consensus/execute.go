package consensus

import (
	"context"
	"fmt"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/certificate"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/forkchoice"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/liskbft"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/sync"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/validator"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/event"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

const (
	P2PEventPostBlock         = "postBlock"
	P2PEventPostSingleCommits = "postSingleCommits"

	certificateBroadcastFrequency = 5
)

// ExecuterConfig holds config for consensus.
type ExecuterConfig struct {
	CTX       context.Context
	ABI       labi.ABI
	Chain     *blockchain.Chain
	Conn      *p2p.P2P
	BlockTime uint32
	BatchSize int
}

// Executer is main process for processing a block.
type Executer struct {
	blockTime       uint32
	batchSize       int
	abi             labi.ABI
	chain           *blockchain.Chain
	conn            *p2p.P2P
	certificatePool *certificate.Pool
	liskBFT         *liskbft.Module
	// Dynamic values
	ctx      context.Context
	database *db.DB
	logger   log.Logger
	// initialize
	blockSlot         *validator.BlockSlot
	syncying          bool
	events            *event.EventEmitter
	processCh         chan *ProcessContext
	closeCh           chan bool
	syncer            *sync.Syncer
	lastBlockReceived *time.Time
	certificateTime   *time.Ticker
}

// NewExecuter creates new consensus component.
func NewExecuter(config *ExecuterConfig) *Executer {
	executer := &Executer{
		abi:             config.ABI,
		chain:           config.Chain,
		conn:            config.Conn,
		liskBFT:         liskbft.NewModule(),
		certificatePool: certificate.NewPool(),
		blockTime:       config.BlockTime,
		batchSize:       config.BatchSize,
		processCh:       make(chan *ProcessContext, 200),
		closeCh:         make(chan bool),
		events:          event.New(),
		syncying:        false,
	}
	return executer
}

type ExecuterInitParam struct {
	CTX          context.Context
	Logger       log.Logger
	Database     *db.DB
	GenesisBlock *blockchain.Block
}

// Init consensus process.
func (c *Executer) Init(param *ExecuterInitParam) error {
	c.ctx = param.CTX
	c.logger = param.Logger
	c.database = param.Database
	c.certificateTime = time.NewTicker(certificateBroadcastFrequency * time.Second)
	c.logger.Info("Initializing consensus")
	if err := c.liskBFT.Init(c.batchSize); err != nil {
		return err
	}
	// handle genesis block
	exist, err := c.chain.GenesisBlockExist(param.GenesisBlock)
	if err != nil {
		return err
	}
	if !exist {
		// process genesis block
		c.logger.Info("Processing genesis block")
		if err := c.processGenesisBlock(&ProcessContext{
			ctx:   c.ctx,
			block: param.GenesisBlock,
		}); err != nil {
			return err
		}
	}
	// Prepare cache
	if err := c.chain.PrepareCache(); err != nil {
		return err
	}
	c.blockSlot = validator.NewBlockSlot(param.GenesisBlock.Header.Timestamp, c.blockTime)
	c.syncer = sync.NewSyncer(c.chain, c.blockSlot, c.conn, c.logger.With("module", "syncer"), c.processValidated, c.deleteBlock)
	// register handler for p2p
	if err := c.conn.RegisterRPCHandler(sync.RPCEndpointGetLastBlock, c.syncer.HandleRPCEndpointGetLastBlock()); err != nil {
		return err
	}
	if err := c.conn.RegisterRPCHandler(sync.RPCEndpointGetHighestCommonBlock, c.syncer.HandleRPCEndpointGetHighestCommonBlock()); err != nil {
		return err
	}
	if err := c.conn.RegisterRPCHandler(sync.RPCEndpointGetBlocksFromID, c.syncer.HandleRPCEndpointGetBlocksFromID()); err != nil {
		return err
	}
	if err := c.conn.RegisterEventHandler(P2PEventPostBlock, func(event *p2p.Event) {
		c.OnBlockReceived(event.Data(), event.PeerID())
	}); err != nil {
		return err
	}
	if err := c.conn.RegisterEventHandler(P2PEventPostSingleCommits, func(event *p2p.Event) {
		c.OnSingleCommitsReceived(event.Data(), event.PeerID())
	}); err != nil {
		return err
	}
	// check for temp blocks
	c.logger.Debug("Finish initializing consensus")
	if _, err := c.abi.Clear(&labi.ClearRequest{}); err != nil {
		return err
	}
	return nil
}

// Start consensus process.
func (c *Executer) Start() error {
	// register topic validator for p2p
	if err := c.conn.RegisterTopicValidator(P2PEventPostBlock, c.blockValidator); err != nil {
		return err
	}
	if err := c.conn.RegisterTopicValidator(P2PEventPostSingleCommits, c.singleCommitValidator); err != nil {
		return err
	}

	for {
		select {
		case <-c.closeCh:
			return nil
		case <-c.ctx.Done():
			return nil
		case <-c.certificateTime.C:
			if err := c.broadcastCertificate(); err != nil {
				c.logger.Errorf("Fail to broadcast certificate with error %v", err)
			}
		case ctx := <-c.processCh:
			if err := c.process(ctx); err != nil {
				c.logger.Errorf("Fail to process block with error %v", err)
			}
		}
	}
}

// Stop consensus process.
func (c *Executer) Stop() error {
	c.events.Close()
	c.certificateTime.Stop()
	close(c.closeCh)
	return nil
}

// Syncing is getter for syncing status.
func (c *Executer) Syncing() bool {
	return c.syncying
}

func (c *Executer) Subscribe(topic string) <-chan interface{} {
	return c.events.Subscribe(topic)
}

// Start consensus process.
func (c *Executer) OnBlockReceived(msgData []byte, peerID string) {
	postBlockEvent := &EventPostBlock{}
	err := postBlockEvent.DecodeStrict(msgData)
	if err != nil {
		c.conn.ApplyPenalty(peerID, p2p.MaxScore)
		c.logger.Errorf("Received invalid block from peer %s. Banning", peerID)
		return
	}
	block, err := blockchain.NewBlock(postBlockEvent.Block)
	if err != nil {
		c.conn.ApplyPenalty(peerID, p2p.MaxScore)
		c.logger.Errorf("Received invalid block from peer %s. Banning", peerID)
		return
	}
	c.events.Publish(EventNetworkBlockNew, &EventNetworkBlockNewMessage{
		Block: block,
	})
	ctx := &ProcessContext{
		ctx:    context.Background(),
		block:  block,
		peerID: peerID,
	}
	// try insert, and not block
	select {
	case c.processCh <- ctx:
	default:
		c.logger.Info("Process queue is full")
	}
}

// TODO - implement this function (GH issue #70)
func (c *Executer) blockValidator(ctx context.Context, msg *p2p.Message) p2p.ValidationResult {
	return p2p.ValidationAccept
}

func (c *Executer) AddInternal(block *blockchain.Block) {
	ctx := &ProcessContext{
		ctx:    context.Background(),
		block:  block,
		peerID: "",
	}
	// try insert, and not block
	select {
	case c.processCh <- ctx:
	default:
		c.logger.Info("Process queue is full")
	}
}

func (c *Executer) Synced(height, maxHeightPrevoted, maxHeightPreviouslyForged uint32) (bool, error) {
	lastBlockHeader := c.chain.LastBlock().Header
	if lastBlockHeader.Version == 0 {
		return height <= lastBlockHeader.Height && maxHeightPrevoted <= lastBlockHeader.Height, nil
	}
	consensusStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	currentMaxHeightPrevoted, _, _, err := c.liskBFT.API().GetBFTHeights(consensusStore)
	if err != nil {
		return false, err
	}
	return (maxHeightPrevoted < currentMaxHeightPrevoted || (maxHeightPrevoted == currentMaxHeightPrevoted && height < lastBlockHeader.Height)), nil
}

// API

func (c *Executer) GetSlotNumber(unixTime uint32) int {
	return c.blockSlot.GetSlotNumber(unixTime)
}
func (c *Executer) GetSlotTime(slot int) uint32 {
	return c.blockSlot.GetSlotTime(slot)
}

func (c *Executer) GetBFTHeights(diffStore *diffdb.Database) (uint32, uint32, uint32, error) {
	return c.liskBFT.API().GetBFTHeights(diffStore)
}
func (c *Executer) GetBFTParameters(diffStore *diffdb.Database, height uint32) (*liskbft.BFTParams, error) {
	return c.liskBFT.API().GetBFTParameters(diffStore, height)
}

func (c *Executer) GetGeneratorKeys(diffStore *diffdb.Database, height uint32) (liskbft.Generators, error) {
	return c.liskBFT.API().GetGeneratorKeys(diffStore, height)
}

func (c *Executer) HeaderHasPriority(context *diffdb.Database, header blockchain.SealedBlockHeader, height, maxHeightPrevoted, maxHeightGenerated uint32) (bool, error) {
	return c.liskBFT.API().HeaderHasPriority(context, header, height, maxHeightPrevoted, maxHeightGenerated)
}

func (c *Executer) BFTBeforeTransactionsExecute(blockHeader blockchain.SealedBlockHeader, diffStore *diffdb.Database) error {
	return c.liskBFT.BeforeTransactionsExecute(blockHeader, diffStore)
}

func (c *Executer) ImpliesMaximalPrevotes(context *diffdb.Database, blockHeader blockchain.ReadableBlockHeader) (bool, error) {
	return c.liskBFT.API().ImpliesMaximalPrevotes(context, blockHeader)
}

// ProcessContext holds context for processing a block.
type ProcessContext struct {
	ctx    context.Context
	block  *blockchain.Block
	peerID string
}

// process received block.
func (c *Executer) process(ctx *ProcessContext) error {
	now := time.Now()
	defer func() {
		c.logger.Debug("Processed block With", time.Since(now))
	}()
	lastBlock := c.chain.LastBlock()
	// Apply fork choice rule
	forkChocie, err := forkchoice.NewForkChoice(lastBlock.Header, ctx.block.Header, c.blockSlot, c.lastBlockReceived)
	if err != nil {
		return err
	}
	if forkChocie.IsIdenticalBlock() {
		c.logger.Debug("Received identical block")
		return nil
	}
	if forkChocie.IsValidBlock() {
		c.logger.Info("Processing valid block")
		c.lastBlockReceived = &now
		if err := ctx.block.Validate(); err != nil {
			return err
		}
		publish := ctx.peerID == "" // if peerID is empty, it means it is internal block
		if err := c.processValidated(ctx.ctx, ctx.block, publish, false); err != nil {
			c.logger.Errorf("Fail to process valid block with %v", err)
			return err
		}
		return nil
	}
	if forkChocie.IsDoubleForging() {
		c.logger.Info("Discarding block due to double forging")
		return nil
	}
	if forkChocie.IsTieBreak() {
		c.lastBlockReceived = &now
		if err := ctx.block.Validate(); err != nil {
			return err
		}
		lastBlock := c.chain.LastBlock()
		if err := c.deleteBlock(ctx.ctx, lastBlock, false); err != nil {
			return err
		}
		publish := ctx.peerID == "" // if peerID is empty, it means it is internal block
		if err := c.processValidated(ctx.ctx, ctx.block, publish, false); err != nil {
			c.logger.Errorf("Fail to process tie break block. Reverting to original block.")
			if err := c.processValidated(ctx.ctx, lastBlock, publish, false); err != nil {
				c.logger.Errorf("Fail to revert the tie break block")
			}
		}
		return nil
	}
	if forkChocie.IsDifferentChain() {
		// start sync
		c.logger.Info("Detected different chain. Sync start")
		c.logger.Debugf("Last block height %d with ID %s, and received height %d with previousBlockID %s", c.chain.LastBlock().Header.Height, c.chain.LastBlock().Header.ID, ctx.block.Header.Height, ctx.block.Header.PreviousBlockID)
		sContext, err := c.createSyncContext(ctx)
		if err != nil {
			return err
		}
		c.syncying = true
		defer func() {
			c.syncying = false
		}()
		if err := c.syncer.Sync(sContext); err != nil {
			c.logger.Errorf("Sync failed with error %v", err)
			return err
		}
		c.logger.Info("Finished sync with different chain")
		return nil
	}

	c.logger.Infof("Discarding block received by %s at height %d", ctx.block.Header.GeneratorAddress.String(), ctx.block.Header.Height)
	return nil
}

func (c *Executer) processValidated(ctx context.Context, block *blockchain.Block, publish bool, removeTemp bool) error {
	consensusStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	if err := c.verifyBlock(consensusStore, block); err != nil {
		return err
	}
	abi, err := newBlockExecuteABI(c.abi, c.liskBFT, consensusStore, block.Header)
	if err != nil {
		return err
	}
	defer abi.Clear() //nolint:errcheck // TODO: update to return err
	if err := abi.Verify(block); err != nil {
		return err
	}

	// publish block to a topic only if it was internally generated
	if publish {
		go func() {
			eventMsg := &EventPostBlock{
				Block: block.MustEncode(),
			}
			if c.syncying {
				return
			}
			err = c.conn.Publish(ctx, P2PEventPostBlock, eventMsg.MustEncode())
			if err != nil {
				c.logger.Errorf("Fail to publish postBlock with %v", err)
			}
		}()
	}

	nextValidatorParams, err := abi.Execute(consensusStore, block)
	if err != nil {
		return err
	}

	resultParams, err := c.liskBFT.API().GetBFTParameters(consensusStore, block.Header.Height+1)
	if err != nil {
		return err
	}
	if !bytes.Equal(resultParams.ValidatorsHash(), block.Header.ValidatorsHash) {
		return fmt.Errorf("invalid validatorsHash. Expected %s but received %s", block.Header.ValidatorsHash, resultParams.ValidatorsHash())
	}
	if len(abi.Events()) > int(blockchain.MaxEventsPerBlock) {
		return fmt.Errorf("invalid number of events. Event must be less than %d but received %d", blockchain.MaxEventsPerBlock, len(abi.Events()))
	}

	currentFinalziedHeight, err := c.chain.DataAccess().GetFinalizedHeight()
	if err != nil {
		return err
	}
	_, maxHeightPrecommited, _, err := c.liskBFT.API().GetBFTHeights(consensusStore)
	if err != nil {
		return err
	}
	batch := c.database.NewBatch()
	diff, err := consensusStore.Commit(batch)
	if err != nil {
		return err
	}
	if err := batch.Set(bytes.Join(blockchain.DBPrefixToBytes(blockchain.DBPrefixStateDiff), bytes.FromUint32(block.Header.Height)), diff.MustEncode()); err != nil {
		return err
	}

	// Save to DB
	if err := abi.Commit(c.chain.LastBlock().Header.StateRoot, block.Header.StateRoot); err != nil {
		return err
	}

	finalizedHeightUpdated := false
	nextFinalizedHeight := currentFinalziedHeight
	if maxHeightPrecommited > currentFinalziedHeight {
		nextFinalizedHeight = maxHeightPrecommited
		finalizedHeightUpdated = true
		// Delete finalized diff
		keys, err := c.database.IterateKey(blockchain.DBPrefixToBytes(blockchain.DBPrefixStateDiff), -1, false)
		if err != nil {
			return err
		}
		for _, key := range keys {
			heightByte := key[1:]
			height := bytes.ToUint32(heightByte)
			if height < maxHeightPrecommited {
				if err := batch.Del(key); err != nil {
					return err
				}
			}
		}
	}
	if err := c.chain.AddBlock(batch, block, abi.Events(), nextFinalizedHeight, removeTemp); err != nil {
		return err
	}
	c.logger.Infof("Processed block at height %d with ID %s", block.Header.Height, block.Header.ID)
	if finalizedHeightUpdated {
		c.events.Publish(EventBlockFinalize, &EventBlockFinalizeMessage{
			Original: currentFinalziedHeight,
			Next:     maxHeightPrecommited,
			Trigger:  block.Header,
		})
	}
	c.events.Publish(EventBlockNew, &EventBlockNewMessage{
		Block:  block,
		Events: abi.Events(),
	})
	if nextValidatorParams != nil {
		c.events.Publish(EventValidatorsChange, &EventChangeValidator{
			NextValidators:       nextValidatorParams.NextValidators,
			PrecommitThreshold:   nextValidatorParams.PrecommitThreshold,
			CertificateThreshold: nextValidatorParams.CertificateThreshold,
		})
	}
	return nil
}

func (c *Executer) processGenesisBlock(ctx *ProcessContext) error {
	c.logger.Info("Processing genesis block")
	if err := ctx.block.ValidateGenesis(); err != nil {
		return err
	}
	consensusStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	abi, err := newGenesisExecuteABI(c.abi, c.liskBFT, ctx.block.Header)
	if err != nil {
		return err
	}
	defer abi.Clear() //nolint:errcheck // TODO: update to return err
	if err := abi.ExecuteGenesis(consensusStore, ctx.block); err != nil {
		return err
	}
	batch := c.database.NewBatch()
	// Check result
	resultParams, err := c.liskBFT.API().GetBFTParameters(consensusStore, ctx.block.Header.Height+1)
	if err != nil {
		return err
	}
	if !bytes.Equal(resultParams.ValidatorsHash(), ctx.block.Header.ValidatorsHash) {
		return fmt.Errorf("invalid validatorsHash. Expected %s but received %s", ctx.block.Header.ValidatorsHash, resultParams.ValidatorsHash())
	}
	calculatedEventRoot, err := blockchain.CalculateEventRoot(abi.Events())
	if err != nil {
		return err
	}
	if !bytes.Equal(calculatedEventRoot, ctx.block.Header.EventRoot) {
		return fmt.Errorf("invalid event root. Event root %s does not match with calculated %s", ctx.block.Header.EventRoot, codec.Hex(calculatedEventRoot))
	}
	diff, err := consensusStore.Commit(batch)
	if err != nil {
		return err
	}
	if err := batch.Set(bytes.Join(blockchain.DBPrefixToBytes(blockchain.DBPrefixStateDiff), bytes.FromUint32(ctx.block.Header.Height)), diff.MustEncode()); err != nil {
		return err
	}
	if err := abi.Commit(ctx.block.Header.StateRoot); err != nil {
		return err
	}
	if err := c.chain.AddBlock(batch, ctx.block, abi.Events(), ctx.block.Header.Height, false); err != nil {
		return err
	}
	return nil
}

func (c *Executer) deleteBlock(ctx context.Context, deletingBlock *blockchain.Block, saveTemp bool) error {
	finalizedHeight, err := c.chain.DataAccess().GetFinalizedHeight()
	if err != nil {
		return err
	}
	if deletingBlock.Header.Height <= finalizedHeight {
		return fmt.Errorf("block height %d cannot be deleted. Height %d is already finalized", deletingBlock.Header.Height, finalizedHeight)
	}
	secondLastBlock, err := c.chain.DataAccess().GetBlockHeaderByHeight(deletingBlock.Header.Height - 1)
	if err != nil {
		return err
	}
	diffStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	abi, err := newBlockRevertABI(c.abi, diffStore, deletingBlock.Header)
	if err != nil {
		return err
	}
	defer abi.Clear() //nolint:errcheck // TODO: update to return err

	c.logger.Debugf("Deleting block %d", deletingBlock.Header.Height)
	diffKey := bytes.Join(blockchain.DBPrefixToBytes(blockchain.DBPrefixStateDiff), bytes.FromUint32(deletingBlock.Header.Height))
	diffBytes, err := c.database.Get(diffKey)
	if err != nil {
		return err
	}
	diff := &diffdb.Diff{}
	if err := diff.Decode(diffBytes); err != nil {
		return err
	}
	batch := c.database.NewBatch()
	if err := diffStore.RevertDiff(batch, diff); err != nil {
		return err
	}
	if err := batch.Del(diffKey); err != nil {
		return err
	}
	if err := abi.Revert(secondLastBlock.StateRoot); err != nil {
		return err
	}
	if err := c.chain.RemoveBlock(batch, saveTemp); err != nil {
		return err
	}
	c.events.Publish(EventBlockDelete, &EventBlockDeleteMessage{
		Block: deletingBlock,
	})
	return nil
}

func (c *Executer) createSyncContext(pCtx *ProcessContext) (*sync.SyncContext, error) {
	diffStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	params, err := c.liskBFT.API().GetBFTParameters(diffStore, c.chain.LastBlock().Header.Height+1)
	if err != nil {
		return nil, err
	}
	currentValidators := make([]codec.Lisk32, len(params.Validators()))
	for i, validator := range params.Validators() {
		currentValidators[i] = validator.Address()
	}
	finalizedHeight, err := c.chain.DataAccess().GetFinalizedHeight()
	if err != nil {
		return nil, err
	}
	header, err := c.chain.DataAccess().GetBlockHeaderByHeight(finalizedHeight)
	if err != nil {
		return nil, err
	}
	ctx := &sync.SyncContext{
		Ctx:                  pCtx.ctx,
		Block:                pCtx.block,
		PeerID:               pCtx.peerID,
		CurrentValidators:    currentValidators,
		FinalizedBlockHeader: header,
	}
	return ctx, nil
}
