package framework

import (
	"context"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/rpc"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type PluginChannel interface {
	Invoke(ctx context.Context, method string, data codec.EncodeDecodable) rpc.EndpointResponse
	Subscribe(event string) chan rpc.EventContent
	Publish(event string, data codec.EncodeDecodable) error
}

type StateStoreGetter func() *diffdb.Database

type PluginConfig struct {
	Context  context.Context
	Logger   log.Logger
	Channel  PluginChannel
	Config   []byte
	DataPath string
}

type Plugin interface {
	Endpoint() statemachine.Endpoint
	Name() string
	Init(cfg *PluginConfig) error
	Start()
	Stop()
}

type Module interface {
	statemachine.Module
	Endpoint() statemachine.Endpoint
	Init(cfg []byte) error
}
