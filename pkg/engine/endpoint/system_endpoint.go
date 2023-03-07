package endpoint

import (
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/router"
	"github.com/LiskHQ/lisk-engine/pkg/txpool"
)

type systemEndpoint struct {
	config        *config.Config
	chain         *blockchain.Chain
	consensusExec *consensus.Executer
	p2pConn       *p2p.Connection
	txPool        *txpool.TransactionPool
	abi           labi.ABI
}

func NewSystemEndpoint(
	config *config.Config,
	chain *blockchain.Chain,
	consensusExec *consensus.Executer,
	p2pConn *p2p.Connection,
	txPool *txpool.TransactionPool,
	abi labi.ABI,
) *systemEndpoint {
	return &systemEndpoint{
		config:        config,
		chain:         chain,
		consensusExec: consensusExec,
		p2pConn:       p2pConn,
		txPool:        txPool,
		abi:           abi,
	}
}

func (a *systemEndpoint) Endpoint() router.EndpointHandlers {
	return map[string]router.EndpointHandler{
		"getNodeInfo": a.HandleGetNodeInfo,
	}
}

type GetNodeInfoResponse struct {
	CurrentTimeslot          uint32                `json:"currentTimeslot" fieldNumber:"1"`
	CurrentTimeslotBeginning uint32                `json:"currentTimeslotBeginning" fieldNumber:"2"`
	LastTimeslot             uint32                `json:"lastTimeslot" fieldNumber:"3"`
	ChainID                  codec.Hex             `json:"chainID" fieldNumber:"4"`
	FinalizedHeight          uint32                `json:"finalizedHeight" fieldNumber:"5"`
	Height                   uint32                `json:"height" fieldNumber:"6"`
	Syncing                  bool                  `json:"syncing" fieldNumber:"7"`
	GenesisConfig            *config.GenesisConfig `json:"genesisConfig" fieldNumber:"8"`
	NetworkVersion           string                `json:"networkVersion" fieldNumber:"9"`
}

func (a *systemEndpoint) HandleGetNodeInfo(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	lastBlock := a.chain.LastBlock()
	chainID := a.chain.ChainID()
	currentTimeslot := a.consensusExec.GetSlotNumber(uint32(time.Now().Unix()))
	lastTimeslot := a.consensusExec.GetSlotNumber(lastBlock.Header.Timestamp)
	currentTimeslotBeginning := a.consensusExec.GetSlotTime(currentTimeslot)
	finalizedHeight, err := a.chain.DataAccess().GetFinalizedHeight()
	syncing := a.consensusExec.Syncing()
	if err != nil {
		w.Error(err)
		return
	}
	resp := &GetNodeInfoResponse{
		CurrentTimeslot:          uint32(currentTimeslot),
		Height:                   lastBlock.Header.Height,
		CurrentTimeslotBeginning: currentTimeslotBeginning,
		LastTimeslot:             uint32(lastTimeslot),
		ChainID:                  chainID,
		FinalizedHeight:          finalizedHeight,
		Syncing:                  syncing,
		GenesisConfig:            a.config.Genesis,
		NetworkVersion:           a.p2pConn.Version(),
	}
	if err := w.Write(resp); err != nil {
		w.Error(err)
		return
	}
}
