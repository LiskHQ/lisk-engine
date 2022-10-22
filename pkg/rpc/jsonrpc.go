package rpc

import (
	"encoding/json"
	"fmt"
)

const (
	jsonRPCParseError          = -32700
	jsonRPCInvalidRequestError = -32600
	jsonRPCMethodNotFoundError = -32601
	jsonRPCInvalidParamError   = -32602
	jsonRPCInternalError       = -32603
	jsonRPCInvalidRequest      = -32000
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      int             `json:"id,string"`
	Params  json.RawMessage `json:"params"`
}

func (r *JSONRPCRequest) Validate() error {
	if r.JSONRPC != "2.0" {
		return fmt.Errorf("invalid json rpc version %s", r.JSONRPC)
	}
	if r.ID < 0 {
		return fmt.Errorf("invalid json rpc id %d", r.ID)
	}
	if r.Method == "" {
		return fmt.Errorf("json RPC method must be specified")
	}
	return nil
}

type JSONRPCErrorResponse struct {
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string                `json:"jsonrpc"`
	ID      int                   `json:"id"`
	Result  json.RawMessage       `json:"result,omitempty"`
	Error   *JSONRPCErrorResponse `json:"error,omitempty"`
}

func getErrResponse(id int, err error, errCode int) []byte {
	jsonErr := &JSONRPCErrorResponse{
		Message: err.Error(),
		Code:    errCode,
	}
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   jsonErr,
	}
	result, err := json.Marshal(resp)
	if err != nil {
		return []byte("Fail to marshal response")
	}
	return result
}

func getSuccessResponse(id int, result []byte) []byte {
	resp := &JSONRPCResponse{
		ID:      id,
		JSONRPC: "2.0",
		Result:  result,
	}
	result, err := json.Marshal(resp)
	if err != nil {
		return []byte("Fail to marshal response")
	}
	return result
}
