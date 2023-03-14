package endpoint

import (
	"encoding/json"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/router"
	"github.com/LiskHQ/lisk-engine/pkg/txpool"
)

type txpoolEndpoint struct {
	config        *config.Config
	chain         *blockchain.Chain
	consensusExec *consensus.Executer
	p2pConn       *p2p.Connection
	txPool        *txpool.TransactionPool
	abi           labi.ABI
}

func NewtxpoolEndpoint(
	config *config.Config,
	chain *blockchain.Chain,
	consensusExec *consensus.Executer,
	p2pConn *p2p.Connection,
	txPool *txpool.TransactionPool,
	abi labi.ABI,
) *txpoolEndpoint {
	return &txpoolEndpoint{
		config:        config,
		chain:         chain,
		consensusExec: consensusExec,
		p2pConn:       p2pConn,
		txPool:        txPool,
		abi:           abi,
	}
}

func (a *txpoolEndpoint) Endpoint() router.EndpointHandlers {
	return map[string]router.EndpointHandler{
		"getTransactionsFromPool": a.HandleGetTransactionsFromPool,
		"postTransaction":         a.HandlePostTransaction,
	}
}

type GetTransactionFromPoolRequest struct {
	OnlyProcessable bool `json:"onlyProcessable" fieldNumber:"1"`
}

type GetTransactionFromPoolResponse struct {
	Transactions []*blockchain.Transaction `json:"transactions" fieldNumber:"1"`
}

func (a *txpoolEndpoint) HandleGetTransactionsFromPool(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &GetTransactionFromPoolRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	var txs []*blockchain.Transaction
	if req.OnlyProcessable {
		txs = a.txPool.GetProcessable()
	} else {
		txs = a.txPool.GetAll()
	}
	resp := &GetTransactionFromPoolResponse{
		Transactions: txs,
	}
	w.Write(resp)
}

type PostTransactionRequest struct {
	Transaction *blockchain.Transaction `json:"transaction" fieldNumber:"1"`
}

type PostTransactionResponse struct {
	TransactionID codec.Hex `json:"transactionID,omitempty" fieldNumber:"1"`
}

func (a *txpoolEndpoint) HandlePostTransaction(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &PostTransactionRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	if err := req.Transaction.Init(); err != nil {
		w.Error(err)
		return
	}
	// validate using state machine
	result, err := a.abi.VerifyTransaction(&labi.VerifyTransactionRequest{
		ContextID:   []byte{},
		Transaction: req.Transaction,
	})
	if err != nil {
		w.Error(fmt.Errorf("transaction %s was not added to the pool with err %w", req.Transaction.ID.String(), err))
		return
	}
	if result.Result == labi.TxVerifyResultInvalid {
		w.Error(fmt.Errorf("transaction %s was not added to the pool", req.Transaction.ID.String()))
		return
	}
	added := a.txPool.Add(req.Transaction)
	if !added {
		w.Error(fmt.Errorf("transaction %s was not added to the pool", req.Transaction.ID.String()))
		return
	}
	resp := &PostTransactionResponse{
		TransactionID: req.Transaction.ID,
	}
	w.Write(resp)
}
