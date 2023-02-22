// Package txpool provides transaction pool which maintain transactions by nonce asc and fee desc order.
package txpool

import (
	"bytes"
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/event"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type DatabaseReader interface {
	Get(key []byte) ([]byte, error)
}

const (
	RPCEventPostTransactionAnnouncement = "postTransactionsAnnouncement"
	RPCEndpointGetTransactions          = "getTransactions"
)

type p2pConnection interface {
	Broadcast(ctx context.Context, event string, data []byte) error
	RegisterRPCHandler(endpoint string, handler p2p.RPCHandler) error
	RegisterEventHandler(name string, handler p2p.EventHandler) error
	ApplyPenalty(pid string, score int) error
	RequestFrom(ctx context.Context, peerID string, procedure string, data []byte) p2p.Response
}
type ABI interface {
	VerifyTransaction(req *labi.VerifyTransactionRequest) (*labi.VerifyTransactionResponse, error)
}

type TransactionPool struct {
	mutex            *sync.RWMutex
	allTransactions  map[string]*TransactionWithFeePriority
	perAccount       map[string]*addressTransactions
	feePriorityQueue FeeMinHeap

	// init
	ctx         context.Context
	config      *TransactionPoolConfig
	logger      log.Logger
	database    DatabaseReader
	chain       *blockchain.Chain
	conn        p2pConnection
	abi         ABI
	ticker      *time.Ticker
	broadcaster *Broadcaster
	events      *event.EventEmitter
	closeCh     chan bool
}

type TransactionPoolConfig struct {
	MaxTransactions             int    `json:"maxTransactions"`
	MaxTransactionsPerAccount   int    `json:"maxTransactionsPerAccount"`
	TransactionExpiryTime       int    `json:"transactionExpiryTime"`
	MinEntranceFeePriority      uint64 `json:"minEntranceFeePriority,string"`
	MinReplacementFeeDifference uint64 `json:"minReplacementFeeDifference,string"`
}

func (c *TransactionPoolConfig) SetDefault() {
	if c.MaxTransactions == 0 {
		c.MaxTransactions = 4096
	}
	if c.MaxTransactionsPerAccount == 0 {
		c.MaxTransactionsPerAccount = 64
	}
	if c.TransactionExpiryTime == 0 {
		c.TransactionExpiryTime = 3 * 60 * 60 // 3hours
	}
	if c.MinReplacementFeeDifference == 0 {
		c.MinReplacementFeeDifference = 1
	}
}

func NewTransactionPool(cfg *TransactionPoolConfig) *TransactionPool {
	config := cfg
	if config == nil {
		config = &TransactionPoolConfig{}
	}
	config.SetDefault()
	queue := FeeMinHeap{}
	heap.Init(&queue)
	return &TransactionPool{
		allTransactions:  map[string]*TransactionWithFeePriority{},
		perAccount:       map[string]*addressTransactions{},
		mutex:            new(sync.RWMutex),
		broadcaster:      NewBroadcaster(),
		feePriorityQueue: queue,
		config:           config,
		events:           event.New(),
		closeCh:          make(chan bool),
	}
}

func (t *TransactionPool) Init(
	ctx context.Context,
	logger log.Logger,
	database *db.DB,
	chain *blockchain.Chain,
	conn p2pConnection,
	abi ABI,
) error {
	t.ctx = ctx
	t.logger = logger
	t.database = database
	t.chain = chain
	t.conn = conn
	t.abi = abi
	t.ticker = time.NewTicker(500 * time.Millisecond)
	if err := t.conn.RegisterRPCHandler(RPCEndpointGetTransactions, t.HandleRPCEndpointGetTransaction); err != nil {
		return err
	}
	if err := t.conn.RegisterEventHandler(RPCEventPostTransactionAnnouncement, func(event *p2p.Event) {
		t.onTransactionAnnoucement(event.Data(), event.PeerID())
	}); err != nil {
		return err
	}
	return nil
}

func (t *TransactionPool) Start() {
	go t.broadcaster.Start(t.ctx, t.logger, t.conn)
	for {
		select {
		case <-t.ticker.C:
			t.reorg()
		case <-t.closeCh:
			return
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *TransactionPool) End() {
	t.events.Close()
	t.broadcaster.Stop()
	close(t.closeCh)
}

func (t *TransactionPool) Get(id []byte) (*blockchain.Transaction, bool) {
	t.mutex.RLocker().Lock()
	defer t.mutex.RLocker().Unlock()
	tx, exist := t.allTransactions[string(id)]
	if !exist {
		return nil, false
	}
	return tx.Transaction, exist
}

func (t *TransactionPool) GetAll() []*blockchain.Transaction {
	t.mutex.RLocker().Lock()
	defer t.mutex.RLocker().Unlock()
	result := make([]*blockchain.Transaction, len(t.allTransactions))
	index := 0
	for _, tx := range t.allTransactions {
		result[index] = tx.Transaction
		index++
	}
	return result
}

func (t *TransactionPool) GetProcessable() []*blockchain.Transaction {
	t.mutex.RLocker().Lock()
	defer t.mutex.RLocker().Unlock()
	result := []*blockchain.Transaction{}
	for _, list := range t.perAccount {
		processables := list.GetProcessables()
		for _, tx := range processables {
			result = append(result, tx.Transaction)
		}
	}
	return result
}

func (t *TransactionPool) Add(tx *blockchain.Transaction) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	_, exist := t.allTransactions[string(tx.ID)]
	if exist {
		t.logger.Debugf("Transaction id %s already exist in the pool", tx.ID.String())
		return false
	}
	feePriority := calculateFeePriority(tx)
	if feePriority < t.config.MinEntranceFeePriority {
		t.logger.Warningf("Rejecting transaction due to failed minimum entrance fee priority requirement. Minimum was %d but received %d", t.config.MinEntranceFeePriority, feePriority)
		return false
	}
	incomingTx := &TransactionWithFeePriority{
		Transaction: tx,
		FeePriority: feePriority,
		receivedAt:  time.Now(),
	}
	var lowestFeePriorityTx *TransactionWithFeePriority
	if len(t.feePriorityQueue) > 0 {
		lowestFeePriorityTx = t.feePriorityQueue[0]
	}
	if len(t.allTransactions) > t.config.MaxTransactions &&
		lowestFeePriorityTx != nil &&
		feePriority <= lowestFeePriorityTx.FeePriority {
		t.logger.Warningf("Rejecting transaction due to fee priority when the pool is full. Minimum fee priority was %d but received %d", lowestFeePriorityTx.FeePriority, feePriority)
		return false
	}

	if result, _ := t.verifyTransactions([]*blockchain.Transaction{tx}); result != labi.TxVeirfyResultOk {
		if result == labi.TxVeirfyResultInvalid {
			t.logger.Warningf("Received invalid transaction %s from %s", tx.ID.String(), tx.SenderAddress().String())
			return false
		}
	}

	if len(t.allTransactions) > t.config.MaxTransactions {
		evicted := t.evictUnprocessable()
		if !evicted {
			t.evictProcessable()
		}
	}

	accountList, listExist := t.perAccount[string(tx.SenderAddress())]
	if !listExist {
		accountList = newAddressTransactions(
			tx.SenderAddress(),
			t.config.MaxTransactionsPerAccount,
			t.config.MinReplacementFeeDifference,
		)
		t.perAccount[string(tx.SenderAddress())] = accountList
	}
	added, removedID, msg := accountList.Add(incomingTx, false)
	if !added {
		if removedID != nil {
			t.logger.Warningf("Transaction with ID %s is removed", removedID.String())
		}
		if msg != "" {
			t.logger.Warningf("Transaction was not added because %s", msg)
		}
		return false
	}

	t.allTransactions[string(tx.ID)] = incomingTx
	heap.Push(&t.feePriorityQueue, incomingTx)
	t.broadcaster.Enqueue(tx.ID)
	return true
}

func (t *TransactionPool) Remove(id []byte) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.remove(id)
}

func (t *TransactionPool) Subscribe(topic string) <-chan interface{} {
	return t.events.Subscribe(topic)
}

func (t *TransactionPool) evictUnprocessable() bool {
	feeMinHeap := FeeMinHeap{}
	heap.Init(&feeMinHeap)
	for _, list := range t.perAccount {
		unprocessables := list.GetUnprocessables()
		for _, tx := range unprocessables {
			heap.Push(&feeMinHeap, tx)
		}
	}
	if len(feeMinHeap) == 0 {
		return false
	}
	tx := heap.Pop(&feeMinHeap).(*TransactionWithFeePriority)
	t.remove(tx.ID)
	return true
}

func (t *TransactionPool) evictProcessable() bool {
	feeMinHeap := FeeMinHeap{}
	heap.Init(&feeMinHeap)
	for _, list := range t.perAccount {
		processables := list.GetProcessables()
		if len(processables) > 0 {
			tx := processables[len(processables)-1]
			heap.Push(&feeMinHeap, tx)
		}
	}
	if len(feeMinHeap) == 0 {
		return false
	}
	tx := heap.Pop(&feeMinHeap).(*TransactionWithFeePriority)
	t.remove(tx.ID)
	return true
}

func (t *TransactionPool) remove(id []byte) bool {
	existingTx, exist := t.allTransactions[string(id)]
	if !exist {
		return false
	}
	delete(t.allTransactions, string(id))
	list := t.perAccount[string(existingTx.SenderAddress())]
	list.Remove(existingTx.Nonce)
	if list.Size() == 0 {
		delete(t.perAccount, string(existingTx.SenderAddress()))
	}
	newFeePriorityQueue := FeeMinHeap{}
	heap.Init(&newFeePriorityQueue)
	for _, tx := range t.allTransactions {
		heap.Push(&newFeePriorityQueue, tx)
	}
	t.feePriorityQueue = newFeePriorityQueue
	return true
}

func (t *TransactionPool) reorg() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	var wg sync.WaitGroup
	for _, list := range t.perAccount {
		wg.Add(1)
		go func(list *addressTransactions) {
			defer wg.Done()
			promotables := list.GetPromotable()
			if len(promotables) == 0 {
				return
			}
			processables := list.GetProcessables()
			combinedTxs := make([]*blockchain.Transaction, len(promotables)+len(processables))
			index := 0
			for _, processable := range processables {
				combinedTxs[index] = processable.Transaction
				index++
			}
			for _, promotable := range promotables {
				combinedTxs[index] = promotable.Transaction
				index++
			}
			result, failedID := t.verifyTransactions(combinedTxs)
			// success case
			if result == labi.TxVeirfyResultOk {
				list.Promote(promotables)
				return
			}
			// if the transaction is just pending keep
			if result == labi.TxVeirfyResultPending {
				return
			}
			t.logger.Warningf("Transaction %s was invalid", failedID)
			// check where it's failed and remove failed tx
			failedIndex := -1
			for i, tx := range combinedTxs {
				if bytes.Equal(tx.ID, failedID) {
					failedIndex = i
					break
				}
			}
			// at least has one tx from promotable
			if failedIndex >= len(processables)+1 {
				promotableIndex := failedIndex - len(processables)
				list.Promote(promotables[:promotableIndex])
			}
			for i := failedIndex; i < len(combinedTxs); i++ {
				t.logger.Infof("Removing transaction %s because transaction %s was invalid", combinedTxs[i].ID, failedID)
				t.remove(combinedTxs[i].ID)
			}
		}(list)
	}
	wg.Wait()
}

func (t *TransactionPool) onTransactionAnnoucement(data []byte, peerID string) {
	if len(data) == 0 {
		t.logger.Warningf("Banning peer %s for sending invalid post transaction announcement", peerID)
		t.conn.ApplyPenalty(peerID, 100)
		return
	}
	event := &PostTransactionAnnouncementEvent{}
	if err := event.Decode(data); err != nil {
		t.logger.Warningf("Banning peer %s for sending invalid post transaction announcement", peerID)
		t.conn.ApplyPenalty(peerID, 100)
		return
	}
	unknownIDs, err := t.getUnknownTransactionIDs(event.TransactionIDs)
	if err != nil {
		t.logger.Errorf("Fail to get unknown transactionIDs")
		return
	}
	if len(unknownIDs) == 0 {
		return
	}

	t.events.Publish(EventTransactionAnnouncement, &EventNewTransactionAnnouncementMessage{
		TransactionIDs: unknownIDs,
	})

	req := &GetTransactionsRequest{
		TransactionIDs: unknownIDs,
	}
	encodedReq, err := req.Encode()
	if err != nil {
		t.logger.Errorf("Fail to encode get transaction request")
		return
	}
	resp := t.conn.RequestFrom(t.ctx, peerID, RPCEndpointGetTransactions, encodedReq)
	if err := resp.Error(); err != nil {
		t.logger.Errorf("Received error from getTransactions endpoint %v", err)
		return
	}
	respData := &GetTransactionsResponse{}
	if err := respData.Decode(resp.Data()); err != nil {
		t.logger.Warningf("Banning peer %s for sending invalid getTransactions response", peerID)
		t.conn.ApplyPenalty(peerID, 100)
		return
	}

	for _, tx := range respData.Transactions {
		if err := tx.Init(); err != nil {
			t.logger.Warningf("Banning peer %s for sending invalid getTransactions response with %w", peerID, err)
			t.conn.ApplyPenalty(peerID, 100)
			return
		}

		resp, err := t.abi.VerifyTransaction(&labi.VerifyTransactionRequest{
			ContextID:   []byte{},
			Transaction: tx,
		})
		if err != nil {
			return
		}

		if resp.Result == labi.TxVeirfyResultInvalid {
			t.logger.Warningf("Banning peer %s for sending invalid getTransactions response", peerID)
			t.conn.ApplyPenalty(peerID, 100)
			return
		}
		if t.Add(tx) {
			t.events.Publish(EventTransactionNew, &EventNewTransactionMessage{
				Transaction: tx,
			})
			t.logger.Info("Added transaction %s received from %s", tx.ID.String(), peerID)
		}
	}
}

type GetTransactionsRequest struct {
	TransactionIDs []codec.Hex `json:"transactionIDs" fieldNumber:"1"`
}

func (e *GetTransactionsRequest) Validate() error {
	if len(e.TransactionIDs) > broadcastReleaseLimit {
		return fmt.Errorf("broadcast release cannot be greater than %d, but received %d", broadcastReleaseLimit, len(e.TransactionIDs))
	}
	for _, id := range e.TransactionIDs {
		if len(id) != 32 {
			return fmt.Errorf("transaction ID must be 32 bytes long")
		}
	}
	return nil
}

type GetTransactionsResponse struct {
	Transactions []*blockchain.Transaction `json:"transactions" fieldNumber:"1"`
}

func (t *TransactionPool) HandleRPCEndpointGetTransaction(w p2p.ResponseWriter, r *p2p.RequestMsg) {
	// Case without request body
	if len(r.Data) == 0 {
		processables := t.GetProcessable()
		if len(processables) > broadcastReleaseLimit {
			processables = processables[:broadcastReleaseLimit]
		}
		resp := &GetTransactionsResponse{
			Transactions: processables,
		}
		encoded, err := resp.Encode()
		if err != nil {
			w.Error(err)
			return
		}
		w.Write(encoded)
		return
	}
	req := &GetTransactionsRequest{}
	if err := req.Decode(r.Data); err != nil {
		t.logger.Warningf("Banning peer %s for sending invalid get transaction request", r.PeerID)
		w.Error(err)
		t.conn.ApplyPenalty(r.PeerID, 100)
		return
	}
	if err := req.Validate(); err != nil {
		t.logger.Warningf("Banning peer %s for sending invalid get transaction request", r.PeerID)
		w.Error(err)
		t.conn.ApplyPenalty(r.PeerID, 100)
		return
	}
	responseList := []*blockchain.Transaction{}
	idNotInPool := [][]byte{}
	for _, id := range req.TransactionIDs {
		tx, exist := t.Get(id)
		if !exist {
			idNotInPool = append(idNotInPool, id)
			continue
		}
		responseList = append(responseList, tx)
	}
	if len(idNotInPool) > 0 {
		transactions, err := t.chain.DataAccess().GetTransactions(idNotInPool)
		if err != nil {
			w.Error(err)
			return
		}
		responseList = append(responseList, transactions...)
	}
	resp := &GetTransactionsResponse{
		Transactions: responseList,
	}
	encoded, err := resp.Encode()
	if err != nil {
		w.Error(err)
		return
	}
	w.Write(encoded)
}

func (t *TransactionPool) getUnknownTransactionIDs(ids []codec.Hex) ([]codec.Hex, error) {
	notInPool := []codec.Hex{}
	for _, id := range ids {
		_, exist := t.Get(id)
		if !exist {
			notInPool = append(notInPool, id)
		}
	}
	// TODO: make it parallel
	unknown := []codec.Hex{}
	for _, id := range notInPool {
		_, err := t.chain.DataAccess().GetTransaction(id)
		if err != nil {
			if errors.Is(err, db.ErrDataNotFound) {
				unknown = append(unknown, id)
				continue
			}
			return nil, err
		}
	}
	return unknown, nil
}

func (t *TransactionPool) verifyTransactions(txs []*blockchain.Transaction) (int32, codec.Hex) {
	if len(txs) == 0 {
		return labi.TxVeirfyResultOk, nil
	}

	for _, tx := range txs {
		res, err := t.abi.VerifyTransaction(&labi.VerifyTransactionRequest{
			ContextID:   []byte{},
			Transaction: tx,
		})
		if err != nil {
			return labi.TxVeirfyResultInvalid, tx.ID
		}
		if res.Result == labi.TxVeirfyResultInvalid {
			return res.Result, tx.ID
		}
	}
	return labi.TxVeirfyResultOk, nil
}
