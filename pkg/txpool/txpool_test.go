package txpool

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type abiMock struct {
	mock.Mock
	allowModule string
}

func (m *abiMock) setAllowModule(mod string) {
	m.allowModule = mod
}

func (m *abiMock) VerifyTransaction(req *labi.VerifyTransactionRequest) (*labi.VerifyTransactionResponse, error) {
	result := labi.TxVerifyResultOk
	if req.Transaction.Module != m.allowModule {
		result = labi.TxVerifyResultInvalid
	}
	return &labi.VerifyTransactionResponse{
		Result: result,
	}, nil
}

type connMock struct {
	mock.Mock
	txs map[string]*blockchain.Transaction
}

func (c *connMock) Broadcast(ctx context.Context, event string, data []byte) error { return nil }
func (c *connMock) RegisterRPCHandler(endpoint string, handler p2p.RPCHandler, opts ...p2p.RPCOption) error {
	return nil
}
func (c *connMock) RegisterEventHandler(name string, handler p2p.EventHandler, validator p2p.Validator) error {
	return nil
}
func (c *connMock) ApplyPenalty(peerID p2p.PeerID, score int) {}
func (c *connMock) RequestFrom(ctx context.Context, peerID p2p.PeerID, procedure string, data []byte) p2p.Response {
	if procedure == RPCEndpointGetTransactions {
		return *p2p.NewResponse(0, "", nil, nil)
	}
	return *p2p.NewResponse(0, "", []byte{}, errors.New("invalid req"))
}
func (c *connMock) Publish(ctx context.Context, topicName string, data []byte) error { return nil }

func TestTxPoolAdd(t *testing.T) {
	cfg := &TransactionPoolConfig{}
	cfg.SetDefault()
	cfg.MinEntranceFeePriority = 1
	pool := NewTransactionPool(cfg)
	stateMachine := statemachine.NewExecuter()
	inMemory, _ := db.NewInMemoryDB()
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 1024,
		MaxBlockCache:         20,
	})
	stateMachine.AddModule(&sampleMod{})
	p2pMock := &connMock{}
	labiMock := &abiMock{allowModule: (&sampleMod{}).Name()}

	pool.Init(
		context.Background(),
		log.DefaultLogger,
		inMemory,
		chain,
		p2pMock,
		labiMock,
	)
	go pool.Start()
	defer pool.End()

	txs := make([]*blockchain.Transaction, 1024)
	pk := crypto.RandomBytes(32)
	var wg sync.WaitGroup
	for i := range txs {
		txs[i] = &blockchain.Transaction{
			SenderPublicKey: pk,
			Module:          (&sampleMod{}).Name(),
			Command:         "transfer",
			Params:          crypto.RandomBytes(20),
			Nonce:           uint64(i) % 64,
			Fee:             10000000000000,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		}
		txs[i].Init()
		if i%64 == 0 {
			pk = crypto.RandomBytes(32)
		}
		wg.Add(1)
		i := i
		go func(tx *blockchain.Transaction) {
			defer wg.Done()
			added := pool.Add(txs[i])
			assert.True(t, added)
		}(txs[i])
	}
	wg.Wait()

	allTxs := pool.GetAll()
	assert.Len(t, allTxs, 1024)

	_, ok := pool.Get(txs[0].ID)
	assert.True(t, ok)

	pool.reorg()
	processableTxs := pool.GetProcessable()
	assert.Greater(t, len(processableTxs), 0)

	invalidTxs := []*blockchain.Transaction{
		// Same tx
		txs[0],
		// Lower fee
		{
			SenderPublicKey: txs[0].SenderPublicKey,
			Module:          "dpos",
			Command:         "vote",
			Params:          crypto.RandomBytes(20),
			Nonce:           0,
			Fee:             100,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		},
		// Same fee
		{
			SenderPublicKey: txs[0].SenderPublicKey,
			Module:          "dpos",
			Command:         "vote",
			Params:          crypto.RandomBytes(20),
			Nonce:           0,
			Fee:             10000000000000,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		},
		{
			SenderPublicKey: crypto.RandomBytes(32),
			Module:          "legacy",
			Command:         "revert",
			Params:          crypto.RandomBytes(20),
			Nonce:           0,
			Fee:             10000000000,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		},
	}

	for _, tx := range invalidTxs {
		t.Logf("Adding tx with fee %d", tx.Fee)
		tx.Init()
		added := pool.Add(tx)
		assert.False(t, added)
	}

	validTxs := []*blockchain.Transaction{
		// higher fee
		{
			SenderPublicKey: txs[0].SenderPublicKey,
			Module:          (&sampleMod{}).Name(),
			Command:         "transfer",
			Params:          crypto.RandomBytes(20),
			Nonce:           0,
			Fee:             90000000000000,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		},
	}

	for _, tx := range validTxs {
		tx.Init()
		added := pool.Add(tx)
		assert.True(t, added)
	}
}

func TestTxPoolOnAnnouncement(t *testing.T) {
	cfg := &TransactionPoolConfig{}
	cfg.SetDefault()
	cfg.MinEntranceFeePriority = 1
	pool := NewTransactionPool(cfg)
	stateMachine := statemachine.NewExecuter()
	inMemory, _ := db.NewInMemoryDB()
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 1024,
		MaxBlockCache:         20,
	})
	chain.Init(&blockchain.Block{
		Header:       &blockchain.BlockHeader{},
		Assets:       []*blockchain.BlockAsset{},
		Transactions: []*blockchain.Transaction{},
	}, inMemory)
	stateMachine.AddModule(&sampleMod{})

	txsMap := map[string]*blockchain.Transaction{}
	tx := &blockchain.Transaction{
		SenderPublicKey: crypto.RandomBytes(32),
		Module:          (&sampleMod{}).Name(),
		Command:         "transfer",
		Params:          crypto.RandomBytes(20),
		Nonce:           uint64(0) % 64,
		Fee:             10000000000000,
		Signatures:      []codec.Hex{crypto.RandomBytes(64)},
	}
	tx.Init()
	txsMap[string(tx.ID)] = tx

	p2pMock := &connMock{
		txs: txsMap,
	}
	labiMock := &abiMock{allowModule: (&sampleMod{}).Name()}
	pool.Init(
		context.Background(),
		log.DefaultLogger,
		inMemory,
		chain,
		p2pMock,
		labiMock,
	)

	assert.Equal(t, p2p.ValidationAccept, pool.transactionValidator(context.Background(), p2p.NewMessage(tx.MustEncode())))
	pool.onTransactionAnnoucement(p2p.NewEvent("127.0.0.1:4949", EventTransactionAnnouncement, tx.MustEncode()))
	assert.Len(t, pool.allTransactions, 1)

	unknownTx := tx.Copy()
	unknownTx.Module = "unknown"
	assert.Equal(t, p2p.ValidationAccept, pool.transactionValidator(context.Background(), p2p.NewMessage(unknownTx.MustEncode())))
	pool.onTransactionAnnoucement(p2p.NewEvent("127.0.0.1:4949", EventTransactionAnnouncement, unknownTx.MustEncode()))
	assert.Len(t, pool.allTransactions, 1)

	assert.Equal(t, p2p.ValidationReject, pool.transactionValidator(context.Background(), p2p.NewMessage([]byte{})))
	assert.Equal(t, p2p.ValidationReject, pool.transactionValidator(context.Background(), p2p.NewMessage(crypto.RandomBytes(200))))
	invalidTx := tx.Copy()
	invalidTx.Signatures = nil // invalid signature to make transaction invalid
	assert.Equal(t, p2p.ValidationReject, pool.transactionValidator(context.Background(), p2p.NewMessage(invalidTx.MustEncode())))
}

func TestTxPoolReorg(t *testing.T) {
	cfg := &TransactionPoolConfig{}
	cfg.SetDefault()
	pool := NewTransactionPool(cfg)
	stateMachine := statemachine.NewExecuter()
	inMemory, _ := db.NewInMemoryDB()
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 1024,
		MaxBlockCache:         20,
	})
	chain.Init(&blockchain.Block{
		Header: &blockchain.BlockHeader{
			ID: crypto.RandomBytes(32),
		},
		Assets:       []*blockchain.BlockAsset{},
		Transactions: []*blockchain.Transaction{},
	}, inMemory)
	stateMachine.AddModule(&sampleMod{})
	stateMachine.Init(log.DefaultLogger)
	p2pMock := &connMock{}
	labiMock := &abiMock{allowModule: (&sampleMod{}).Name()}
	pool.Init(
		context.Background(),
		log.DefaultLogger,
		inMemory,
		chain,
		p2pMock,
		labiMock,
	)

	txs := make([]*blockchain.Transaction, 1024)
	pk := crypto.RandomBytes(32)
	var wg sync.WaitGroup
	for i := range txs {
		if i%64 == 0 {
			pk = crypto.RandomBytes(32)
		}
		txs[i] = &blockchain.Transaction{
			SenderPublicKey: pk,
			Module:          (&sampleMod{}).Name(),
			Command:         "transfer",
			Params:          crypto.RandomBytes(20),
			Nonce:           uint64(i) % 64,
			Fee:             10000000000000,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		}
		txs[i].Init()

		wg.Add(1)
		i := i
		go func(tx *blockchain.Transaction) {
			defer wg.Done()
			added := pool.Add(txs[i])
			assert.True(t, added)
		}(txs[i])
	}
	wg.Wait()

	processables := pool.GetProcessable()
	assert.Len(t, processables, 0)
	pool.reorg()
	processables = pool.GetProcessable()
	assert.Len(t, processables, 1024)

	// Replace with nonce 30, and make everything unprocessable again
	newTx := &blockchain.Transaction{
		SenderPublicKey: txs[0].SenderPublicKey,
		Module:          (&sampleMod{}).Name(),
		Command:         "transfer",
		Params:          crypto.RandomBytes(20),
		Nonce:           30,
		Fee:             90000000000000,
		Signatures:      []codec.Hex{crypto.RandomBytes(64)},
	}
	newTx.Init()
	added := pool.Add(newTx)
	assert.True(t, added)

	// set error key to trigger command to return error
	labiMock.setAllowModule("rand")
	inMemory.Set(bytes.Join(blockchain.DBPrefixToBytes(blockchain.DBPrefixState), bytes.FromUint32(3), bytes.FromUint16(0), []byte("error-key")), []byte{})

	pool.reorg()
	processables = pool.GetProcessable()
	// set error key to trigger command to return error
	assert.LessOrEqual(t, len(processables), 1000)
	// it should remove the error transactions
	assert.LessOrEqual(t, len(pool.GetAll()), 1024)
}

type responseWriterMock struct {
	mock.Mock
	data []byte
	err  error
}

func (r *responseWriterMock) Write(d []byte) {
	r.data = d
}
func (r *responseWriterMock) Error(err error) {
	r.err = err
}

func TestTxPoolHandleGetTransaction(t *testing.T) {
	cfg := &TransactionPoolConfig{}
	cfg.SetDefault()
	pool := NewTransactionPool(cfg)
	stateMachine := statemachine.NewExecuter()
	inMemory, _ := db.NewInMemoryDB()
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 1024,
		MaxBlockCache:         20,
	})
	chain.Init(&blockchain.Block{
		Header: &blockchain.BlockHeader{
			ID: crypto.RandomBytes(32),
		},
		Assets:       []*blockchain.BlockAsset{},
		Transactions: []*blockchain.Transaction{},
	}, inMemory)
	stateMachine.AddModule(&sampleMod{})
	stateMachine.Init(log.DefaultLogger)
	p2pMock := &connMock{}
	labiMock := &abiMock{allowModule: (&sampleMod{}).Name()}
	pool.Init(
		context.Background(),
		log.DefaultLogger,
		inMemory,
		chain,
		p2pMock,
		labiMock,
	)
	pk := crypto.RandomBytes(32)
	txs := make([]*blockchain.Transaction, 1024)
	var wg sync.WaitGroup
	for i := range txs {
		if i%64 == 0 {
			pk = crypto.RandomBytes(32)
		}
		txs[i] = &blockchain.Transaction{
			SenderPublicKey: pk,
			Module:          (&sampleMod{}).Name(),
			Command:         "transfer",
			Params:          crypto.RandomBytes(20),
			Nonce:           uint64(i) % 64,
			Fee:             10000000000000,
			Signatures:      []codec.Hex{crypto.RandomBytes(64)},
		}
		txs[i].Init()

		wg.Add(1)
		i := i
		go func(tx *blockchain.Transaction) {
			defer wg.Done()
			added := pool.Add(txs[i])
			assert.True(t, added)
		}(txs[i])
	}
	wg.Wait()
	pool.reorg()

	// empty req
	req := &p2p.Request{
		PeerID:    "127.0.0.1:4949",
		Procedure: RPCEndpointGetTransactions,
		Data:      []byte{},
	}
	resp := &responseWriterMock{}
	pool.HandleRPCEndpointGetTransaction(resp, req)
	assert.Nil(t, resp.err)
	body := &GetTransactionsResponse{}
	err := body.Decode(resp.data)
	assert.NoError(t, err)
	assert.Len(t, body.Transactions, 100)

	// not empty req
	ids := make([]codec.Hex, 101)
	for i := range ids {
		ids[i] = crypto.RandomBytes(32)
	}
	req = &p2p.Request{
		PeerID:    "127.0.0.1:4949",
		Procedure: RPCEndpointGetTransactions,
		Data:      crypto.RandomBytes(10),
	}
	resp = &responseWriterMock{}
	p2pMock.On("ApplyPenalty", mock.AnythingOfType("string"), mock.AnythingOfType("int"))
	pool.HandleRPCEndpointGetTransaction(resp, req)
	assert.Nil(t, resp.data)
	assert.Nil(t, resp.err)
}
