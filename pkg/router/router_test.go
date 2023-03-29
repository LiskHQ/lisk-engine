package router

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

type mockEndpoint struct {
	mock.Mock
}

func (m *mockEndpoint) handler1(w EndpointResponseWriter, r *EndpointRequest) {
	w.Write(&blockchain.Transaction{Nonce: 4})
}

func (m *mockEndpoint) handler2(w EndpointResponseWriter, r *EndpointRequest) {
	tx := &blockchain.Transaction{}
	if err := json.Unmarshal(r.Params(), tx); err != nil {
		w.Error(err)
		return
	}
	w.Write(&blockchain.Transaction{Nonce: tx.Nonce})
}

func TestRouterRegister(t *testing.T) {
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 15 * 1024,
		MaxBlockCache:         515,
	})

	database, err := db.NewInMemoryDB()
	if err != nil {
		panic(err)
	}
	router := NewRouter()
	router.Init(database, log.DefaultLogger, chain)

	endpoint := &mockEndpoint{}

	err = router.RegisterEndpoint("base", "run", endpoint.handler1)
	assert.NoError(t, err)

	err = router.RegisterEndpoint("base", "run", endpoint.handler1)
	assert.EqualError(t, err, "endpoint run at base is already registered")

	err = router.RegisterEvents("base", "emit")
	assert.NoError(t, err)

	err = router.RegisterEvents("base", "emit")
	assert.EqualError(t, err, "event emit at base is already registered")
}

func TestRouterInvoke(t *testing.T) {
	// Arrange
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 15 * 1024,
		MaxBlockCache:         515,
	})

	database, err := db.NewInMemoryDB()
	if err != nil {
		panic(err)
	}
	router := NewRouter()
	genesis := &blockchain.Block{
		Header: &blockchain.BlockHeader{
			Timestamp: 100,
		},
	}
	chain.Init(genesis, database)
	err = chain.AddBlock(database.NewBatch(), genesis, []*blockchain.Event{}, 0, false)
	assert.NoError(t, err)
	err = chain.PrepareCache()
	assert.NoError(t, err)
	router.Init(database, log.DefaultLogger, chain)

	endpoint := &mockEndpoint{}

	err = router.RegisterEndpoint("base", "handle1", endpoint.handler1)
	assert.NoError(t, err)
	err = router.RegisterEndpoint("base", "handle2", endpoint.handler2)
	assert.NoError(t, err)

	{
		resp := router.Invoke(context.Background(), "base_handle1", nil)
		tx, ok := resp.Data().(*blockchain.Transaction)
		assert.True(t, ok)
		assert.Equal(t, uint64(4), tx.Nonce)

		txBody := &blockchain.Transaction{Nonce: 1110}
		txBytes, err := json.Marshal(txBody)
		assert.NoError(t, err)

		resp = router.Invoke(context.Background(), "base_handle2", txBytes)
		tx, ok = resp.Data().(*blockchain.Transaction)
		assert.True(t, ok)
		assert.Equal(t, uint64(1110), tx.Nonce)

		resp = router.Invoke(context.Background(), "base_noMethod", nil)
		assert.EqualError(t, resp.Err(), "endpoint noMethod at base does not exist")

		resp = router.Invoke(context.Background(), "new_handle1", nil)
		assert.EqualError(t, resp.Err(), "endpoint handle1 at new does not exist")
	}
}

func TestRouterSubscribe(t *testing.T) {
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 15 * 1024,
		MaxBlockCache:         515,
	})

	database, err := db.NewInMemoryDB()
	if err != nil {
		panic(err)
	}
	router := NewRouter()
	genesis := &blockchain.Block{
		Header: &blockchain.BlockHeader{
			Timestamp: 100,
		},
	}
	chain.Init(genesis, database)
	err = chain.AddBlock(database.NewBatch(), genesis, []*blockchain.Event{}, 0, false)
	assert.NoError(t, err)
	err = chain.PrepareCache()
	assert.NoError(t, err)
	router.Init(database, log.DefaultLogger, chain)

	err = router.RegisterEvents("base", "event1")
	assert.NoError(t, err)
	err = router.RegisterEvents("base", "event2")
	assert.NoError(t, err)
	defer router.Close()

	sub1 := router.Subscribe("base_event1")
	sub2 := router.Subscribe("base_event2")

	go func() {
		<-time.After(1 * time.Millisecond)
		err := router.Publish("base_event1", &blockchain.Transaction{Nonce: 100})
		assert.NoError(t, err)
		err = router.Publish("base_event2", &blockchain.Transaction{Nonce: 200})
		assert.NoError(t, err)
	}()

	e1 := <-sub1
	tx, ok := e1.Data().(*blockchain.Transaction)
	assert.True(t, ok)
	assert.Equal(t, uint64(100), tx.Nonce)
	e2 := <-sub2
	tx, ok = e2.Data().(*blockchain.Transaction)
	assert.True(t, ok)
	assert.Equal(t, uint64(200), tx.Nonce)

	err = router.Publish("base_noEvent", nil)
	assert.EqualError(t, err, "event noEvent at base does not exist")

	err = router.Publish("new_event1", nil)
	assert.EqualError(t, err, "event event1 at new does not exist")
}
