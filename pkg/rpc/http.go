package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

type httpJSONRPCServer struct {
	invoker    Invoker
	logger     log.Logger
	httpServer *http.Server
	httpMux    *http.ServeMux
}

func NewHTTPJSONServer(logger log.Logger, port int, addr string, invoker Invoker) *httpJSONRPCServer {
	server := &httpJSONRPCServer{
		invoker: invoker,
		logger:  logger,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", server.HandleRequest)
	bindingAddr := addr
	if addr == "" {
		bindingAddr = "127.0.0.1"
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", bindingAddr, port),
		Handler:           mux,
		ReadHeaderTimeout: time.Second * 1,
	}
	server.httpServer = httpServer
	server.httpMux = mux

	return server
}

func (s *httpJSONRPCServer) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *httpJSONRPCServer) Close() error {
	return s.httpServer.Close()
}

func (s *httpJSONRPCServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("Invalid method")); err != nil {
			s.logger.Errorf("Fail to write message with %w", err)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	s.logger.Debugf("Received request from %s", r.RemoteAddr)
	body := &JSONRPCRequest{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		result := getErrResponse(0, err, jsonRPCParseError)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write(result); err != nil {
			s.logger.Errorf("Fail to write message with %w", err)
		}
		return
	}
	if err := body.Validate(); err != nil {
		result := getErrResponse(body.ID, err, jsonRPCInvalidRequestError)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write(result); err != nil {
			s.logger.Errorf("Fail to write message with %w", err)
		}
		return
	}
	s.logger.Infof("Received request from %s for method %s", r.RemoteAddr, body.Method)
	result := s.invoker.Invoke(r.Context(), body.Method, body.Params)
	if err := result.Err(); err != nil {
		result := getErrResponse(body.ID, err, jsonRPCInvalidRequest)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write(result); err != nil {
			s.logger.Errorf("Fail to write message with %w", err)
		}
		return
	}
	resultData, err := result.JSONData()
	if err != nil {
		result := getErrResponse(body.ID, err, jsonRPCInvalidRequest)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write(result); err != nil {
			s.logger.Errorf("Fail to write message with %w", err)
		}
		return
	}
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      body.ID,
		Result:  resultData,
	}
	parsedResult, err := json.Marshal(resp)
	if err != nil {
		result := getErrResponse(body.ID, err, jsonRPCInvalidRequest)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write(result); err != nil {
			s.logger.Errorf("Fail to write message with %w", err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(parsedResult); err != nil {
		s.logger.Errorf("Fail to write message with %w", err)
	}
}
