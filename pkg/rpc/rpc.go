// Package rpc provides JSON RPC server for IPC. HTTP and WS protocols.
package rpc

import (
	"context"
	"errors"
	"strings"

	stringsUtil "github.com/LiskHQ/lisk-engine/pkg/collection/strings"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const (
	RPCTypeHTTP = "http"
	RPCTypeWS   = "ws"
	RPCIPC      = "ipc"
)

type Invoker interface {
	Invoke(ctx context.Context, endpoint string, data []byte) EndpointResponse
}

type RPCServer struct {
	done       chan bool
	httpServer *httpJSONRPCServer
	wsServer   *wsJSONRPCServer
}

func NewRPCServer(logger log.Logger, enabled []string, invoker Invoker, port int, address, path string) *RPCServer {
	server := &RPCServer{
		done: make(chan bool),
	}
	if stringsUtil.Contain(enabled, RPCTypeHTTP) {
		logger.Infof("Starting HTTP RPC server at %d on %s", port, address)
		server.httpServer = NewHTTPJSONServer(logger, port, address, invoker)
	}
	if stringsUtil.Contain(enabled, RPCTypeWS) {
		logger.Infof("Starting WS RPC server at %d on %s", port, address)
		if server.httpServer != nil {
			server.wsServer = NewWSJSONRPCServerWithHTTPServer(logger, invoker, server.httpServer.httpMux, server.httpServer.httpServer)
		} else {
			server.wsServer = NewWSJSONRPCServer(logger, port, address, invoker)
		}
	}
	return server
}

func (s *RPCServer) ListenAndServe() error {
	if s.httpServer != nil {
		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil {
				s.Close()
			}
		}()
	}
	if s.wsServer != nil {
		go func() {
			if err := s.wsServer.ListenAndServe(); err != nil {
				s.Close()
			}
		}()
	}
	<-s.done
	return nil
}

func (s *RPCServer) Publish(method string, data []byte) {
	if s.wsServer != nil {
		s.wsServer.Publish(method, data)
	}
}

func (s *RPCServer) Close() error {
	close(s.done)
	errs := []error{}
	if s.httpServer != nil {
		if err := s.httpServer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.wsServer != nil {
		if err := s.wsServer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		errStr := make([]string, len(errs))
		for i, err := range errs {
			errStr[i] = err.Error()
		}
		return errors.New(strings.Join(errStr, ","))
	}
	return nil
}
