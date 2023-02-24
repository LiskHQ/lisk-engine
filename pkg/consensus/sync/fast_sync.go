package sync

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

// Syncer sync the block with network.
type fastSyncer struct {
	chain     *blockchain.Chain
	conn      *p2p.P2P
	logger    log.Logger
	processor processFn
	reverter  revertFn
}

// Sync with network.
func (s *fastSyncer) Sync(ctx *SyncContext) (bool, error) {
	lastBlockHeader := s.chain.LastBlock().Header
	commonBlockHeader, err := s.getCommonBlock(ctx, lastBlockHeader)
	if err != nil {
		return false, err
	}
	if commonBlockHeader.Height < ctx.FinalizedBlockHeader.Height {
		err = s.conn.ApplyPenalty(ctx.PeerID, p2p.MaxScore)
		if err != nil {
			s.logger.Error("Fail to apply penalty to a peer %v with %v", ctx.PeerID, err)
		}
		return false, errors.New("received common block has hight lower than finalized block")
	}
	twoRounds := uint32(len(ctx.CurrentValidators)) * 2
	if lastBlockHeader.Height-commonBlockHeader.Height > twoRounds || ctx.Block.Header.Height-commonBlockHeader.Height > twoRounds {
		// In this case, abort and wait for new block
		return true, errors.New("height diffference is greater than 2 rounds")
	}
	downloader := NewDownloader(
		ctx.Ctx,
		s.logger,
		s.conn,
		s.chain,
		ctx.PeerID,
		commonBlockHeader.ID,
		commonBlockHeader.Height,
		ctx.Block.Header.ID,
		ctx.Block.Header.Height,
	)
	downloadedBlocks, err := s.downloadAndValidate(ctx, downloader)
	if err != nil {
		return true, err
	}
	// delete to common block
	if err := s.deleteTillCommonBlock(ctx, commonBlockHeader); err != nil {
		// common block delete cannot be recovered by retrying
		return true, err
	}
	// apply downloaded block
	for _, block := range downloadedBlocks {
		if err := s.processor(ctx.Ctx, block, false); err != nil {
			// recover temp block, if failed cannot continue
			if err := s.restoreBlocks(ctx, commonBlockHeader); err != nil {
				return true, err
			}
			errPenalty := s.conn.ApplyPenalty(ctx.PeerID, p2p.MaxScore)
			if errPenalty != nil {
				s.logger.Error("Fail to apply penalty to a peer %v with %v", ctx.PeerID, errPenalty)
			}
			return true, err
		}
	}
	// clear temp block
	if err := s.chain.DataAccess().ClearTempBlocks(); err != nil {
		s.logger.Error("Fail to clear temp blocks with %v", err)
	}
	return true, nil
}

func (s *fastSyncer) downloadAndValidate(ctx *SyncContext, downloader *Downloader) ([]*blockchain.Block, error) {
	go downloader.Start()
	downloadedBlocks := []*blockchain.Block{}
	for downloaded := range downloader.downloaded {
		if downloaded.err != nil {
			return downloadedBlocks, downloaded.err
		}
		if err := downloaded.block.Validate(); err != nil {
			downloader.Stop()
			errPenalty := s.conn.ApplyPenalty(ctx.PeerID, p2p.MaxScore)
			if errPenalty != nil {
				s.logger.Error("Fail to apply penalty to a peer %v with %v", ctx.PeerID, errPenalty)
			}
			return downloadedBlocks, err
		}
		downloadedBlocks = append(downloadedBlocks, downloaded.block)
	}
	return downloadedBlocks, nil
}

func (s *fastSyncer) getCommonBlock(ctx *SyncContext, lastBlockHeader *blockchain.BlockHeader) (*blockchain.BlockHeader, error) {
	heights := getLastHeights(lastBlockHeader.Height, len(ctx.CurrentValidators)*2)
	headers, err := s.chain.DataAccess().GetBlockHeadersByHeights(heights)
	if err != nil {
		return nil, err
	}
	ids := make([][]byte, len(headers))
	for i, block := range headers {
		ids[i] = block.ID
	}
	blockID, err := requestHighestCommonBlock(ctx.Ctx, s.conn, ctx.PeerID, ids)
	if err != nil {
		if errors.Is(err, errCommonBlockNotFound) {
			errPenalty := s.conn.ApplyPenalty(ctx.PeerID, p2p.MaxScore)
			if errPenalty != nil {
				s.logger.Error("Fail to apply penalty to a peer %v with %v", ctx.PeerID, errPenalty)
			}
			return nil, err
		}
		return nil, err
	}
	header, err := s.chain.DataAccess().GetBlockHeader(blockID)
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (s *fastSyncer) deleteTillCommonBlock(ctx *SyncContext, commonBlock *blockchain.BlockHeader) error {
	lastBlockHeight := s.chain.LastBlock().Header.Height
	for lastBlockHeight != commonBlock.Height {
		lastBlock := s.chain.LastBlock()
		if err := s.reverter(ctx.Ctx, lastBlock, true); err != nil {
			return err
		}
		lastBlockHeight = s.chain.LastBlock().Header.Height
	}
	return nil
}

func (s *fastSyncer) restoreBlocks(ctx *SyncContext, commonBlockHeader *blockchain.BlockHeader) error {
	if err := s.deleteTillCommonBlock(ctx, commonBlockHeader); err != nil {
		return err
	}
	blocks, err := s.chain.DataAccess().GetTempBlocks()
	if err != nil {
		return err
	}
	blockchain.SortBlockByHeightAsc(blocks)
	for _, block := range blocks {
		if err := s.processor(ctx.Ctx, block, true); err != nil {
			return err
		}
	}
	return nil
}

func getLastHeights(start uint32, num int) []uint32 {
	result := []uint32{}
	for i := 0; i < num-1; i++ {
		if start < uint32(i) {
			return result
		}
		next := start - uint32(i)
		result = append(result, next)
	}
	return result
}
