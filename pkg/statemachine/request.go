package statemachine

import (
	"context"
	"encoding/json"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/rpc"
)

func NewEndpointRequest(
	context context.Context,
	logger log.Logger,
	diffStore *diffdb.Database,
	chainID codec.Hex,
	lastBlockHeader *blockchain.BlockHeader,
	jsonData []byte,
) *EndpointRequest {
	return &EndpointRequest{
		context:         context,
		logger:          logger,
		jsonData:        jsonData,
		diffStore:       diffStore,
		lastBlockHeader: lastBlockHeader,
		chainID:         chainID,
	}
}

type EndpointRequest struct {
	context         context.Context
	logger          log.Logger
	jsonData        []byte
	diffStore       *diffdb.Database
	lastBlockHeader *blockchain.BlockHeader
	chainID         codec.Hex
}

func (a *EndpointRequest) Logger() log.Logger {
	return a.logger
}

func (a *EndpointRequest) Context() context.Context {
	return a.context
}

func (a *EndpointRequest) ChainID() codec.Hex {
	return a.chainID
}

func (a *EndpointRequest) LastBlockHeader() blockchain.SealedBlockHeader {
	return a.lastBlockHeader.Readonly()
}

func (a *EndpointRequest) GetStore(moduleID uint32, prefix uint16) Store {
	return a.diffStore.WithPrefix(ModuleStorePrefix(moduleID, prefix))
}

func (a *EndpointRequest) Immutable() ImmutableAPIContext {
	return &immutableAPIContext{diffStore: a.diffStore}
}

func (a *EndpointRequest) Data(data codec.EncodeDecodable) error {
	if len(a.jsonData) == 0 {
		return nil
	}
	return json.Unmarshal(a.jsonData, data)
}

type NotFoundHandler func(namespace, method string, w rpc.EndpointResponseWriter, r *EndpointRequest)
type EndpointHandler func(w rpc.EndpointResponseWriter, r *EndpointRequest)
type EndpointHandlers map[string]EndpointHandler
type Endpoint interface {
	Get() EndpointHandlers
}
