package txpool

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

type abiMock struct {
	mock.Mock
	allowModule string
}

func (m *abiMock) setAllowModule(mod string) {
	m.allowModule = mod
}

func (m *abiMock) VerifyTransaction(req *labi.VerifyTransactionRequest) (*labi.VerifyTransactionResponse, error) {
	result := labi.TxVeirfyResultOk
	if req.Transaction.Module != m.allowModule {
		result = labi.TxVeirfyResultInvalid
	}
	return &labi.VerifyTransactionResponse{
		Result: result,
	}, nil
}

type connMock struct {
	mock.Mock
	txs map[string]*blockchain.Transaction
}
type respMock struct {
	mock.Mock
	data []byte
	err  error
}

func (r *respMock) Err() error {
	return r.err
}
func (r *respMock) Data() []byte {
	return r.data
}

func (c *connMock) Broadcast(ctx context.Context, event string, data []byte) {
}
func (c *connMock) RegisterRPCHandler(endpoint string, handler p2p.RPCHandler) error { return nil }
func (c *connMock) RegisterEventHandler(name string, handler p2p.EventHandler) error { return nil }
func (c *connMock) ApplyPenalty(peerID string, score int)                            {}
func (c *connMock) RequestFrom(ctx context.Context, peerID string, procedure string, data []byte) p2p.Response {
	if procedure == RPCEndpointGetTransactions {
		req := &GetTransactionsRequest{}
		if err := req.Decode(data); err != nil {
			return &respMock{
				err: err,
			}
		}
		txs := []*blockchain.Transaction{}
		for _, id := range req.TransactionIDs {
			if tx, exist := c.txs[string(id)]; exist {
				txs = append(txs, tx)
			}
		}
		resp := &GetTransactionsResponse{
			Transactions: txs,
		}
		return &respMock{
			data: resp.MustEncode(),
		}
	}
	return &respMock{
		err: errors.New("invalid req"),
	}
}

func TestBroadcaster(t *testing.T) {
	cMock := &connMock{}
	broadcaster := NewBroadcaster()
	broadcaster.ctx = context.Background()
	broadcaster.logger = log.DefaultLogger
	broadcaster.conn = cMock

	ids := make([]codec.Hex, 200)
	for i := range ids {
		ids[i] = crypto.RandomBytes(32)
	}

	var wg sync.WaitGroup
	// enqueue all twice
	for _, id := range ids {
		id := id
		wg.Add(2)
		go func() {
			defer wg.Done()
			broadcaster.Enqueue(id)
		}()
		go func() {
			defer wg.Done()
			broadcaster.Enqueue(id)
		}()
	}
	wg.Wait()
	assert.Len(t, broadcaster.idMap, len(ids))

	// Remove last half
	for _, id := range ids[150:] {
		id := id
		wg.Add(2)
		go func() {
			defer wg.Done()
			broadcaster.Remove(id)
		}()
		go func() {
			defer wg.Done()
			broadcaster.Remove(id)
		}()
	}
	wg.Wait()
	assert.Len(t, broadcaster.idMap, 150)

	// broadcast
	cMock.On("Broadcast", broadcaster.ctx, RPCEventPostTransactionAnnouncement, mock.Anything)
	broadcaster.broadacast()
	cMock.MethodCalled("Broadcast", broadcaster.ctx, RPCEventPostTransactionAnnouncement, mock.Anything)
	assert.Len(t, broadcaster.idMap, 50)

	broadcaster.broadacast()
	assert.Len(t, broadcaster.idMap, 0)
}

func TestBroadcastStartStop(t *testing.T) {
	cMock := &connMock{}
	broadcaster := NewBroadcaster()
	go func() {
		<-time.After(1 * time.Second)
		broadcaster.Stop()
	}()
	broadcaster.Start(context.Background(), log.DefaultLogger, cMock)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(1 * time.Second)
		cancel()
	}()
	broadcaster.Start(ctx, log.DefaultLogger, cMock)
}

func TestPostTransactionAnnouncementEvent(t *testing.T) {
	ids := make([]codec.Hex, 101)
	for i := range ids {
		ids[i] = crypto.RandomBytes(32)
	}

	assert.EqualError(t, (&PostTransactionAnnouncementEvent{
		TransactionIDs: ids[:101],
	}).Validate(), "broadcast release cannot be greater than 100, but received 101")

	ids = ids[:99]
	ids = append(ids, crypto.RandomBytes(20))
	assert.EqualError(t, (&PostTransactionAnnouncementEvent{
		TransactionIDs: ids,
	}).Validate(), "transaction ID must be 32 bytes long")

	ids = ids[:99]
	ids = append(ids, crypto.RandomBytes(32))
	assert.NoError(t, (&PostTransactionAnnouncementEvent{
		TransactionIDs: ids,
	}).Validate())
}
