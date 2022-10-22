// Package router provides request routing logic for JSON RPC server [pkg/github.com/LiskHQ/lisk-engine/pkg/rpc].
package router

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/rpc"
)

const (
	delimitor = "_"
)

func NewEndpointRequest(
	context context.Context,
	logger log.Logger,
	jsonData []byte,
) *EndpointRequest {
	return &EndpointRequest{
		context: context,
		logger:  logger,
		params:  jsonData,
	}
}

type EndpointResponseWriter = rpc.EndpointResponseWriter

type EndpointRequest struct {
	context context.Context
	logger  log.Logger
	params  []byte
}

func (a *EndpointRequest) Logger() log.Logger {
	return a.logger
}

func (a *EndpointRequest) Context() context.Context {
	return a.context
}

func (a *EndpointRequest) Params() []byte {
	return a.params
}

type NotFoundHandler func(namespace, method string, w EndpointResponseWriter, r *EndpointRequest)
type EndpointHandler func(w EndpointResponseWriter, r *EndpointRequest)
type EndpointHandlers map[string]EndpointHandler
type Endpoint interface {
	Get() EndpointHandlers
}

type Router struct {
	mutex           *sync.RWMutex
	logger          log.Logger
	endpoints       map[string]map[string]EndpointHandler
	events          map[string]map[string][]chan rpc.EventContent
	notfoundHandler NotFoundHandler
}

func NewRouter() *Router {
	return &Router{
		mutex:     new(sync.RWMutex),
		endpoints: map[string]map[string]EndpointHandler{},
		events:    map[string]map[string][]chan rpc.EventContent{},
	}
}
func (b *Router) Init(stateDB *db.DB, logger log.Logger, chain *blockchain.Chain) error {
	b.logger = logger

	return nil
}

func (b *Router) RegisterEndpoint(namespace, method string, handler EndpointHandler) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	_, pathExist := b.endpoints[namespace]
	if !pathExist {
		b.endpoints[namespace] = map[string]EndpointHandler{}
	}
	_, actionExist := b.endpoints[namespace][method]
	if actionExist {
		return fmt.Errorf("endpoint %s at %s is already registered", method, namespace)
	}
	b.endpoints[namespace][method] = handler
	return nil
}

func (b *Router) RegisterNotFoundHandler(handler NotFoundHandler) {
	b.notfoundHandler = handler
}

func (b *Router) RegisterEvents(namespace, event string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	_, pathExist := b.events[namespace]
	if !pathExist {
		b.events[namespace] = map[string][]chan rpc.EventContent{}
	}
	_, actionExist := b.events[namespace][event]
	if actionExist {
		return fmt.Errorf("event %s at %s is already registered", event, namespace)
	}
	b.events[namespace][event] = []chan rpc.EventContent{}
	return nil
}

func (b *Router) Invoke(ctx context.Context, endpoint string, data []byte) rpc.EndpointResponse {
	path, target, err := splitTarget(endpoint)
	if err != nil {
		return rpc.NewEndpointResponse(nil, err)
	}
	b.mutex.RLocker().Lock()
	_, pathExist := b.endpoints[path]
	if !pathExist {
		return rpc.NewEndpointResponse(nil, fmt.Errorf("endpoint %s at %s does not exist", target, path))
	}
	handler, actionExist := b.endpoints[path][target]
	if !actionExist {
		return rpc.NewEndpointResponse(nil, fmt.Errorf("endpoint %s at %s does not exist", target, path))
	}
	b.mutex.RLocker().Unlock()
	resp := rpc.NewEndpointResponseWriter()
	req := &EndpointRequest{
		context: ctx,
		logger:  b.logger,
		params:  data,
	}
	r := make(chan bool)
	go func() {
		defer close(r)
		handler(resp, req)
	}()
	select {
	case <-r:
		return rpc.NewEndpointResponse(resp.Result().Data(), resp.Result().Err())
	case <-ctx.Done():
		return rpc.NewEndpointResponse(nil, fmt.Errorf("failed on endpoint %s at %s", target, path))
	}
}

func (b *Router) Subscribe(event string) chan rpc.EventContent {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	path, target, err := splitTarget(event)
	if err != nil {
		return nil
	}
	_, pathExist := b.events[path]
	if !pathExist {
		b.events[path] = map[string][]chan rpc.EventContent{}
	}
	_, methodExist := b.events[path][target]
	if !methodExist {
		b.events[path][target] = []chan rpc.EventContent{}
	}
	subscriber := make(chan rpc.EventContent)
	b.events[path][target] = append(b.events[path][target], subscriber)
	return subscriber
}

func (b *Router) Publish(event string, data codec.EncodeDecodable) error {
	b.mutex.RLocker().Lock()
	defer b.mutex.RLocker().Unlock()
	path, target, err := splitTarget(event)
	if err != nil {
		return err
	}
	_, pathExist := b.events[path]
	if !pathExist {
		return fmt.Errorf("event %s at %s does not exist", target, path)
	}
	events, actionExist := b.events[path][target]
	if !actionExist {
		return fmt.Errorf("event %s at %s does not exist", target, path)
	}
	for _, e := range events {
		e <- rpc.NewEventContent(event, data)
	}
	return nil
}

func (b *Router) Close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, path := range b.events {
		for _, channels := range path {
			for _, c := range channels {
				close(c)
			}
		}
	}
}

func splitTarget(target string) (string, string, error) {
	res := strings.Split(target, delimitor)
	if len(res) != 2 {
		return "", "", fmt.Errorf("endpoint or event path %s is not valid. Format should be XXX_YYY", target)
	}
	return res[0], res[1], nil
}
