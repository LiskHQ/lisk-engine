package txpool

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

const (
	EventTransactionNew          = "EventTransactionNew"
	EventTransactionAnnouncement = "EventTransactionAnnouncement"
)

type EventNewTransactionMessage struct {
	Transaction *blockchain.Transaction `json:"transaction"`
}

type EventNewTransactionAnnouncementMessage struct {
	TransactionIDs []codec.Hex `json:"transactionIDs"`
}
