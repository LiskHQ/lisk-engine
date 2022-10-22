package engine

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

const (
	RPCEventChainNewBlock          = "chain_newBlock"
	RPCEventChainDeleteBlock       = "chain_deleteBlock"
	RPCEventChainForked            = "chain_forked"
	RPCEventChainValidatorsChanged = "chain_validatorsChanged"
	RPCEventNetworkNewBlock        = "network_newBlock"
	RPCEventNetworkNewTransaction  = "network_newTransaction"
	RPCEventTxpoolNewTransaction   = "txpool_newTransaction"
)

type EventBlockHeader struct {
	BlockHeader *blockchain.BlockHeader `json:"blockHeader"`
}

type EventTransactionIDs struct {
	TransactionIDs []codec.Hex `json:"transactionIDs"`
}

type EventTransaction struct {
	Transaction *blockchain.Transaction `json:"transaction"`
}

type EventValidatorChange struct {
	Validators           []*labi.Validator `json:"nextValidators"`
	PrecommitThreshold   uint64            `json:"precommitThreshold,string"`
	CertificateThreshold uint64            `json:"certificateThreshold,string"`
}
