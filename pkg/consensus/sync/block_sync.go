package sync

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/forkchoice"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

// Syncer sync the block with network.
type blockSyncer struct {
	chain     *blockchain.Chain
	conn      *p2p.Connection
	logger    log.Logger
	processor processFn
	reverter  revertFn
}

// Sync with network.
func (s *blockSyncer) Sync(ctx *SyncContext) (bool, error) {
	peers := s.conn.ConnectedPeers()

	nodeInfos := make([]*NodeInfo, 0, len(peers))
	wg := sync.WaitGroup{}

	// Get last block header from all peers and create node info for each peer.
	for _, p := range peers {
		wg.Add(1)
		go func(peer p2p.PeerID) {
			defer wg.Done()
			blockHeader, err := requestLastBlockHeader(ctx.Ctx, s.conn, peer)
			if err != nil {
				s.logger.Errorf("Fail to get last block header from %s with %v", peer.String(), err)
				return
			}
			nodeInfo := NewNodeInfo(blockHeader.Height, blockHeader.MaxHeightPrevoted, blockHeader.Version, blockHeader.ID)
			nodeInfo.PeerID = peer
			nodeInfos = append(nodeInfos, nodeInfo)
		}(p)
	}
	wg.Wait()

	nodeInfo, err := getBestNodeInfo(nodeInfos)
	if err != nil {
		return false, err
	}
	lastBlockHeader := s.chain.LastBlock().Header
	if lastBlockHeader.Version == 2 {
		if !forkchoice.IsDifferentChain(lastBlockHeader.MaxHeightPrevoted, nodeInfo.maxHeightPrevoted, lastBlockHeader.Height, nodeInfo.height) {
			return false, errors.New("invalid to trigger sync condition")
		}
	}
	// compute last block
	networkLastBlockHeader, err := s.getAndValidateNetworkLastBlock(ctx, nodeInfo)
	if err != nil {
		return false, err
	}
	s.logger.Infof("Received and validated network last block at height %d", networkLastBlockHeader.Height)
	commonBlock, err := s.getCommonBlockHeader(ctx, nodeInfo)
	if err != nil {
		return false, err
	}
	s.logger.Infof("Deleting until common block height %d", commonBlock.Height)
	if err := s.deleteTillCommonBlock(ctx, commonBlock); err != nil {
		return false, err
	}
	downloader := NewDownloader(
		ctx.Ctx,
		s.logger,
		s.conn,
		s.chain,
		nodeInfo.PeerID,
		commonBlock.ID,
		commonBlock.Height,
		networkLastBlockHeader.ID,
		networkLastBlockHeader.Height,
	)
	if err := s.downloadAndProcess(ctx, downloader); err != nil {
		return false, err
	}

	s.chain.DataAccess().ClearTempBlocks()
	return true, nil
}

func (s *blockSyncer) downloadAndProcess(ctx *SyncContext, downloader *Downloader) error {
	go downloader.Start()
	for downloaded := range downloader.downloaded {
		if downloaded.err != nil {
			return downloaded.err
		}
		if err := downloaded.block.Validate(); err != nil {
			downloader.Stop()
			s.conn.ApplyPenalty(ctx.PeerID, p2p.MaxPenaltyScore)
			return err
		}
		publish := ctx.PeerID == "" // if PeerID is empty, it means it is internal block
		if err := s.processor(ctx.Ctx, downloaded.block, publish, false); err != nil {
			downloader.Stop()
			return err
		}
	}
	return nil
}

// getAndValidateNetworkLastBlock returns ok bool. in case of false, if err is nil, it can app.
func (s *blockSyncer) getAndValidateNetworkLastBlock(ctx *SyncContext, nodeInfo *NodeInfo) (*blockchain.BlockHeader, error) {
	networkLastBlockHeader, err := requestLastBlockHeader(ctx.Ctx, s.conn, nodeInfo.PeerID)
	if err != nil {
		return nil, err
	}
	if err := networkLastBlockHeader.Validate(); err != nil {
		s.logger.Infof("Applying penalty to %s because it provided invalid last block", nodeInfo.PeerID)
		s.conn.ApplyPenalty(nodeInfo.PeerID, p2p.MaxPenaltyScore)
		return nil, fmt.Errorf("invalid block received from %s", nodeInfo.PeerID)
	}
	lastBlockHeader := s.chain.LastBlock().Header
	if lastBlockHeader.Version == 2 {
		if !forkchoice.IsDifferentChain(lastBlockHeader.MaxHeightPrevoted, networkLastBlockHeader.MaxHeightPrevoted, lastBlockHeader.Height, networkLastBlockHeader.Height) {
			s.logger.Infof("Applying penalty to %s because it provided last block which does not have priority", nodeInfo.PeerID)
			s.conn.ApplyPenalty(nodeInfo.PeerID, p2p.MaxPenaltyScore)
			return nil, fmt.Errorf("last block received from %s does not have priority", nodeInfo.PeerID)
		}
	}
	return networkLastBlockHeader, nil
}

func (s *blockSyncer) getCommonBlockHeader(ctx *SyncContext, nodeInfo *NodeInfo) (*blockchain.BlockHeader, error) {
	lastBlock := s.chain.LastBlock()
	startingHeight := getCommonBlockStartSearchHeight(lastBlock.Header.Height, len(ctx.CurrentValidators))
	for trial := 0; trial < 3; trial++ {
		heights := getHeightWithGap(startingHeight, ctx.FinalizedBlockHeader.Height, len(ctx.CurrentValidators), 10)
		headers, err := s.chain.DataAccess().GetBlockHeadersByHeights(heights)
		if err != nil {
			return nil, err
		}
		ids := make([][]byte, len(headers))
		for i, header := range headers {
			ids[i] = header.ID
		}
		blockID, err := requestHighestCommonBlock(ctx.Ctx, s.conn, nodeInfo.PeerID, ids)
		if err != nil {
			if errors.Is(err, errCommonBlockNotFound) {
				startingHeight = heights[len(heights)-1] - uint32(len(ctx.CurrentValidators))
				continue
			}
			return nil, err
		}
		header, err := s.chain.DataAccess().GetBlockHeader(blockID)
		if err != nil {
			return nil, err
		}
		return header, nil
	}
	return nil, errors.New("fail to obtain common block")
}

func (s *blockSyncer) deleteTillCommonBlock(ctx *SyncContext, commonBlock *blockchain.BlockHeader) error {
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

func getCommonBlockStartSearchHeight(height uint32, roundLength int) uint32 {
	currentRound := math.Ceil(float64(height) / float64(roundLength))
	if currentRound == 0 {
		return 0
	}
	startingHeight := uint32((currentRound - 1) * float64(roundLength))
	return startingHeight
}

func getHeightWithGap(start, minimum uint32, gap, num int) []uint32 {
	result := []uint32{}
	if start <= minimum {
		result = append(result, minimum)
		return result
	}
	for i := 0; i < num-1; i++ {
		if start < minimum+uint32(i*gap) {
			return result
		}
		next := start - uint32(i*gap)
		result = append(result, next)
	}
	return result
}
