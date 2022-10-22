package txpool

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const (
	broadcastReleaseLimit = 100
	broadcastInterval     = 5 * time.Second
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type PostTransactionAnnouncementEvent struct {
	TransactionIDs []codec.Hex `json:"transactionIDs" fieldNumber:"1"`
}

func (e *PostTransactionAnnouncementEvent) Validate() error {
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

type Broadcaster struct {
	mutex   *sync.Mutex
	idMap   map[string]bool
	queue   []codec.Hex
	closeCh chan bool
	ticker  *time.Ticker

	// instance
	ctx    context.Context
	conn   p2pConnection
	logger log.Logger
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		mutex:   new(sync.Mutex),
		idMap:   map[string]bool{},
		queue:   []codec.Hex{},
		closeCh: make(chan bool),
	}
}

func (b *Broadcaster) Start(ctx context.Context, logger log.Logger, conn p2pConnection) {
	b.ctx = ctx
	b.logger = logger
	b.conn = conn
	b.ticker = time.NewTicker(broadcastInterval)
	for {
		select {
		case <-b.ticker.C:
			b.broadacast()
		case <-b.ctx.Done():
			return
		case <-b.closeCh:
			return
		}
	}
}

func (b *Broadcaster) Stop() {
	b.ticker.Stop()
	close(b.closeCh)
}

func (b *Broadcaster) Enqueue(id codec.Hex) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	_, exist := b.idMap[string(id)]
	if exist {
		return
	}
	b.idMap[string(id)] = true
	b.queue = append(b.queue, id)
}

func (b *Broadcaster) Remove(id codec.Hex) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	_, exist := b.idMap[string(id)]
	if !exist {
		return
	}
	delete(b.idMap, string(id))
	newList := []codec.Hex{}
	for _, queuedID := range b.queue {
		if !bytes.Equal(id, queuedID) {
			newList = append(newList, queuedID)
		}
	}
	b.queue = newList
}

func (b *Broadcaster) broadacast() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.queue) == 0 {
		return
	}
	limit := len(b.queue)
	if len(b.queue) > broadcastReleaseLimit {
		limit = broadcastReleaseLimit
	}
	broadcasting := b.queue[:limit]
	b.queue = b.queue[limit:]
	list := make([]codec.Hex, len(broadcasting))
	for i, id := range broadcasting {
		delete(b.idMap, string(id))
		list[i] = id
	}
	data := &PostTransactionAnnouncementEvent{
		TransactionIDs: list,
	}
	encoded, err := data.Encode()
	if err != nil {
		b.logger.Error("Fail to encode PostTransactionAnnouncementEvent")
		return
	}
	b.conn.Broadcast(b.ctx, RPCEventPostTransactionAnnouncement, encoded)
}
