package endpoint

import (
	"encoding/json"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/router"
	"github.com/LiskHQ/lisk-engine/pkg/txpool"
)

type chainEndpoint struct {
	chain         *blockchain.Chain
	consensusExec *consensus.Executer
	p2pConn       *p2p.P2P
	txPool        *txpool.TransactionPool
	abi           labi.ABI
}

func NewChainEndpoint(
	chain *blockchain.Chain,
	consensusExec *consensus.Executer,
	p2pConn *p2p.P2P,
	txPool *txpool.TransactionPool,
	abi labi.ABI,
) *chainEndpoint {
	return &chainEndpoint{
		chain:         chain,
		consensusExec: consensusExec,
		p2pConn:       p2pConn,
		txPool:        txPool,
		abi:           abi,
	}
}

func (a *chainEndpoint) Endpoint() router.EndpointHandlers {
	return map[string]router.EndpointHandler{
		"getLastBlock":       a.HandleGetLastBlock,
		"getGetBlockByID":    a.HandleGetBlockByID,
		"getBlockByHeight":   a.HandleGetBlockByHeight,
		"getTransactionByID": a.HandleGetTransactionByID,
		"postBlock":          a.HandlePostBlock,
	}
}

func (a *chainEndpoint) HandleGetLastBlock(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	block, err := a.chain.DataAccess().GetLastBlock()
	if err != nil {
		w.Error(err)
		return
	}
	if err := w.Write(block); err != nil {
		w.Error(err)
	}
}

type GetBlockByIDRequest struct {
	ID codec.Hex `json:"id" fieldNumber:"1"`
}

func (a *chainEndpoint) HandleGetBlockByID(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &GetBlockByIDRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	block, err := a.chain.DataAccess().GetBlock(req.ID)
	if err != nil {
		w.Error(err)
		return
	}
	if err := w.Write(block); err != nil {
		w.Error(err)
	}
}

type GetBlockByHeightRequest struct {
	Height uint32 `json:"height" fieldNumber:"1"`
}

func (a *chainEndpoint) HandleGetBlockByHeight(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &GetBlockByHeightRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	block, err := a.chain.DataAccess().GetBlockByHeight(req.Height)
	if err != nil {
		w.Error(err)
		return
	}
	if err := w.Write(block); err != nil {
		w.Error(err)
	}
}

type GetTransactionByIDRequest struct {
	ID codec.Hex `json:"id" fieldNumber:"1"`
}

func (a *chainEndpoint) HandleGetTransactionByID(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &GetTransactionByIDRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	tx, err := a.chain.DataAccess().GetTransaction(req.ID)
	if err != nil {
		w.Error(err)
		return
	}
	if err := w.Write(tx); err != nil {
		w.Error(err)
	}
}

type PostBlockRequest struct {
	Block *blockchain.Block `json:"block" fieldNumber:"1"`
}

type PostBlockResponse struct {
	BlockID codec.Hex `json:"blockID" fieldNumber:"1"`
}

func (a *chainEndpoint) HandlePostBlock(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &PostBlockRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	if err := req.Block.Init(); err != nil {
		w.Error(err)
		return
	}
	a.consensusExec.AddInternal(req.Block)
	resp := &PostBlockResponse{
		BlockID: req.Block.Header.ID,
	}
	if err := w.Write(resp); err != nil {
		w.Error(err)
	}
}
