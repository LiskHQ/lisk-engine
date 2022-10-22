package consensus

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

const (
	EventNetworkBlockNew  = "EventNetworkBlockNew"
	EventBlockNew         = "EventBlockNew"
	EventBlockDelete      = "EventBlockDelete"
	EventBlockFinalize    = "EventBlockFinalize"
	EventChainFork        = "EventChainFork"
	EventValidatorsChange = "EventValidatorsChange"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type EventNetworkBlockNewMessage struct {
	Block *blockchain.Block `fieldNumber:"1"`
}

type EventBlockNewMessage struct {
	Block  *blockchain.Block   `fieldNumber:"1"`
	Events []*blockchain.Event `fieldNumber:"2"`
}

type EventBlockDeleteMessage struct {
	Block *blockchain.Block `fieldNumber:"1"`
}

type EventChainForkMessage struct {
	Block *blockchain.Block `fieldNumber:"1"`
}

type EventChangeValidator struct {
	NextValidators       []*labi.Validator `fieldNumber:"1"`
	PrecommitThreshold   uint64            `fieldNumber:"2"`
	CertificateThreshold uint64            `fieldNumber:"3"`
}

type EventBlockFinalizeMessage struct {
	Original uint32                  `fieldNumber:"1"`
	Next     uint32                  `fieldNumber:"2"`
	Trigger  *blockchain.BlockHeader `fieldNumber:"3"`
}
