package sync

import (
	"context"
	"errors"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const (
	RPCEndpointGetLastBlock          = "getLastBlock"
	RPCEndpointGetHighestCommonBlock = "getHighestCommonBlock"
	RPCEndpointGetBlocksFromID       = "getBlocksFromId"
	defaultTimeout                   = 3 * time.Second
)

var (
	errCommonBlockNotFound = errors.New("peer did not return common block")
)

func requestLastBlockHeader(ctx context.Context, conn *p2p.P2P, peerID string) (*blockchain.BlockHeader, error) {
	respChan := make(chan p2p.Response)
	childCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	go func() {
		defer close(respChan)
		resp := conn.RequestFrom(childCtx, peerID, RPCEndpointGetLastBlock, nil)
		respChan <- resp
	}()
	select {
	case result := <-respChan:
		if result.Error() != nil {
			return nil, result.Error()
		}
		block, err := blockchain.NewBlock(result.Data())
		if err != nil {
			return nil, err
		}
		return block.Header, nil
	case <-childCtx.Done():
		return nil, errors.New("context ended")
	}
}

type getHighestCommonBlockRequest struct {
	ids [][]byte `fieldNumber:"1"`
}

type getHighestCommonBlockResponse struct {
	id []byte `fieldNumber:"1"`
}

func requestHighestCommonBlock(ctx context.Context, conn *p2p.P2P, peerID string, ids [][]byte) ([]byte, error) {
	respChan := make(chan p2p.Response)
	childCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	body := &getHighestCommonBlockRequest{ids: ids}
	encoded, err := body.Encode()
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(respChan)
		resp := conn.RequestFrom(childCtx, peerID, RPCEndpointGetHighestCommonBlock, encoded)
		respChan <- resp
	}()
	select {
	case result := <-respChan:
		if result.Error() != nil {
			return nil, result.Error()
		}
		commonBlockResp := &getHighestCommonBlockResponse{}
		if err := commonBlockResp.Decode(result.Data()); err != nil {
			return nil, err
		}
		if len(commonBlockResp.id) == 0 {
			return nil, errCommonBlockNotFound
		}
		return commonBlockResp.id, nil
	case <-childCtx.Done():
		return nil, errors.New("context ended")
	}
}

type getBlocksFromIDRequest struct {
	blockID []byte `fieldNumber:"1"`
}

type getBlocksFromIDResponse struct {
	blocks [][]byte `fieldNumber:"1"`
}

func requestBlocksFromID(ctx context.Context, conn *p2p.P2P, peerID string, id []byte) ([]*blockchain.Block, error) {
	respChan := make(chan p2p.Response)
	childCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	body := &getBlocksFromIDRequest{blockID: id}
	encoded, err := body.Encode()
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(respChan)
		resp := conn.RequestFrom(childCtx, peerID, RPCEndpointGetBlocksFromID, encoded)
		respChan <- resp
	}()
	select {
	case result := <-respChan:
		if result.Error() != nil {
			return nil, result.Error()
		}
		if result.Data() == nil {
			return nil, errors.New("peer did not return common block")
		}
		resp := &getBlocksFromIDResponse{}
		if err := resp.Decode(result.Data()); err != nil {
			return nil, err
		}
		blocks := make([]*blockchain.Block, len(resp.blocks))
		for i, blockBytes := range resp.blocks {
			block, err := blockchain.NewBlock(blockBytes)
			if err != nil {
				return nil, err
			}
			blocks[i] = block
		}
		return blocks, nil
	case <-childCtx.Done():
		return nil, errors.New("context ended")
	}
}
