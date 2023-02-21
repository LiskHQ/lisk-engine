// Package sync provides block synchronization mechanism following the BFT protocol.
package sync

import (
	"bytes"
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/validator"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type processFn = func(ctx context.Context, block *blockchain.Block, removeTemp bool) error
type revertFn = func(ctx context.Context, deletingBlock *blockchain.Block, saveTemp bool) error

// Syncer sync the block with network.
type Syncer struct {
	chain       *blockchain.Chain
	blockslot   *validator.BlockSlot
	conn        *p2p.P2P
	logger      log.Logger
	blockSyncer *blockSyncer
	fastSyncer  *fastSyncer
}

func NewSyncer(
	chain *blockchain.Chain,
	blockslot *validator.BlockSlot,
	conn *p2p.P2P,
	logger log.Logger,
	processor processFn,
	reverter revertFn,
) *Syncer {
	blockSync := &blockSyncer{
		chain:     chain,
		conn:      conn,
		logger:    logger,
		processor: processor,
		reverter:  reverter,
	}
	fastSync := &fastSyncer{
		chain:     chain,
		conn:      conn,
		logger:    logger,
		processor: processor,
		reverter:  reverter,
	}
	syncer := &Syncer{
		chain:       chain,
		logger:      logger,
		conn:        conn,
		blockslot:   blockslot,
		blockSyncer: blockSync,
		fastSyncer:  fastSync,
	}
	return syncer
}

type SyncContext struct {
	Ctx                  context.Context
	Block                *blockchain.Block
	FinalizedBlockHeader *blockchain.BlockHeader
	PeerID               string
	CurrentValidators    []codec.Lisk32
}

// Sync with network.
func (s *Syncer) Sync(ctx *SyncContext) error {
	if err := ctx.Block.Validate(); err != nil {
		return err
	}
	if s.shouldFastSync(ctx) {
		for {
			done, err := s.fastSyncer.Sync(ctx)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
	if s.shouldSync(ctx) {
		for {
			done, err := s.blockSyncer.Sync(ctx)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
	s.logger.Info("Sync method cannot be determined")
	return nil
}

func (s *Syncer) shouldFastSync(ctx *SyncContext) bool {
	twoRounds := len(ctx.CurrentValidators) * 2
	lastBlockHeader := s.chain.LastBlock().Header
	diff := int(math.Abs(float64(ctx.Block.Header.Height) - float64(lastBlockHeader.Height)))
	if diff > twoRounds {
		return false
	}
	for _, validatorAddress := range ctx.CurrentValidators {
		if bytes.Equal(validatorAddress, ctx.Block.Header.GeneratorAddress) {
			return true
		}
	}
	return false
}

func (s *Syncer) shouldSync(ctx *SyncContext) bool {
	threeRounds := len(ctx.CurrentValidators) * 3
	currentSlot := s.blockslot.GetSlotNumber(uint32(time.Now().Unix()))
	finalizedSlot := s.blockslot.GetSlotNumber(ctx.FinalizedBlockHeader.Timestamp)
	return currentSlot-finalizedSlot > threeRounds
}

func (s *Syncer) HandleRPCEndpointGetLastBlock() p2p.RPCHandler {
	return func(w p2p.ResponseWriter, r *p2p.RequestMsg) {
		lastBlock := s.chain.LastBlock()
		encoded, err := lastBlock.Encode()
		if err != nil {
			w.Error(err)
			return
		}
		w.Write(encoded)
	}
}

type GetHighestCommonBlockRequest struct {
	IDs [][]byte `json:"ids" fieldNumber:"1"`
}

type GetHighestCommonBlockResponse struct {
	ID []byte `json:"id" fieldNumber:"1"`
}

func (s *Syncer) HandleRPCEndpointGetHighestCommonBlock() p2p.RPCHandler {
	return func(w p2p.ResponseWriter, r *p2p.RequestMsg) {
		if r.Data == nil {
			s.conn.ApplyPenalty(r.PeerID, 100)
			s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
			return
		}
		req := &GetHighestCommonBlockRequest{}
		if err := req.Decode(r.Data); err != nil {
			s.conn.ApplyPenalty(r.PeerID, 100)
			s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
			return
		}
		if len(req.IDs) == 0 {
			s.conn.ApplyPenalty(r.PeerID, 100)
			s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
			return
		}
		for _, id := range req.IDs {
			if len(id) != 32 {
				s.conn.ApplyPenalty(r.PeerID, 100)
				s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
				return
			}
		}
		var wg sync.WaitGroup
		resultChan := make(chan *blockchain.BlockHeader)
		wg.Add(len(req.IDs))

		for _, id := range req.IDs {
			go func(id []byte) {
				defer wg.Done()
				header, err := s.chain.DataAccess().GetBlockHeader(id)
				if err == nil {
					resultChan <- header
				}
			}(id)
		}
		go func() {
			wg.Wait()
			close(resultChan)
		}()
		headers := []*blockchain.BlockHeader{}
		for r := range resultChan {
			headers = append(headers, r)
		}
		if len(headers) == 0 {
			w.Write(nil)
			return
		}
		// sort by desc height
		sort.Slice(headers, func(i, j int) bool { return headers[i].Height > headers[j].Height })
		resp := &GetHighestCommonBlockResponse{
			ID: headers[0].ID,
		}
		encoded, err := resp.Encode()
		if err != nil {
			w.Error(err)
			return
		}
		w.Write(encoded)
	}
}

type GetBlocksFromIDRequest struct {
	ID codec.Hex `json:"id" fieldNumber:"1"`
}

type GetBlocksFromIDResponse struct {
	Blocks []*blockchain.Block `json:"blocks" fieldNumber:"1"`
}

func (s *Syncer) HandleRPCEndpointGetBlocksFromID() p2p.RPCHandler {
	return func(w p2p.ResponseWriter, r *p2p.RequestMsg) {
		if r.Data == nil {
			s.conn.ApplyPenalty(r.PeerID, 100)
			s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
			return
		}
		req := &GetBlocksFromIDRequest{}
		if err := req.Decode(r.Data); err != nil {
			s.conn.ApplyPenalty(r.PeerID, 100)
			s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
			return
		}
		if len(req.ID) != 32 {
			s.conn.ApplyPenalty(r.PeerID, 100)
			s.logger.Warningf("Banning peer %s with invalid request on getHighestCommonBlock", r.PeerID)
			return
		}
		requestedBlock, err := s.chain.DataAccess().GetBlockHeader(req.ID)
		if err != nil {
			w.Error(err)
			return
		}
		from := requestedBlock.Height + 1
		to := ints.Min(requestedBlock.Height+103, s.chain.LastBlock().Header.Height)
		blocks, err := s.chain.DataAccess().GetBlocksBetweenHeight(from, to)
		if err != nil {
			w.Error(err)
			return
		}
		resp := &GetBlocksFromIDResponse{
			Blocks: blocks,
		}
		encoded, err := resp.Encode()
		if err != nil {
			w.Error(err)
			return
		}
		w.Write(encoded)
	}
}
