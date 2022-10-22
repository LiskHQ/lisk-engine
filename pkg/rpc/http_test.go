package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

var port = 12345

func mustCreateBody(req *JSONRPCRequest) io.Reader {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(reqBytes)
}

func mustMarshal(data interface{}) string {
	reqBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(reqBytes)
}

type mockInvoker struct{}

func (i *mockInvoker) Invoke(ctx context.Context, endpoint string, data []byte) EndpointResponse {
	tx := &blockchain.Transaction{}
	if err := json.Unmarshal(data, tx); err != nil {
		return NewEndpointResponse(nil, err)
	}
	if tx.Nonce == 0 {
		return NewEndpointResponse(nil, errors.New("invalid nonce"))
	}
	return NewEndpointResponse(tx, nil)
}

func TestHTTPServer(t *testing.T) {
	chain := blockchain.NewChain(&blockchain.ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 15 * 1024,
		MaxBlockCache:         515,
	})

	database, err := db.NewInMemoryDB()
	if err != nil {
		panic(err)
	}
	invoker := &mockInvoker{}
	genesis := &blockchain.Block{
		Header: &blockchain.BlockHeader{
			Timestamp: 100,
		},
	}
	chain.Init(genesis, database)
	err = chain.AddBlock(database.NewBatch(), genesis, []*blockchain.Event{}, 0, false)
	assert.NoError(t, err)
	err = chain.PrepareCache()
	assert.NoError(t, err)

	assert.NoError(t, err)

	server := NewHTTPJSONServer(log.DefaultLogger, port, "0.0.0.0", invoker)
	defer server.Close()
	go func() {
		server.ListenAndServe()
	}()

	cli := http.DefaultClient

	testTable := []struct {
		method       string
		statusCode   int
		body         io.Reader
		expectedResp string
	}{
		{
			method:       "GET",
			body:         nil,
			statusCode:   http.StatusBadRequest,
			expectedResp: "Invalid method",
		},
		{
			method:     "POST",
			statusCode: http.StatusBadRequest,
			body:       bytes.NewReader([]byte("invalid req")),
			expectedResp: mustMarshal(&JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCErrorResponse{
					Code:    jsonRPCParseError,
					Message: "invalid character 'i' looking for beginning of value",
				},
			}),
		},
		{
			method:     "POST",
			statusCode: http.StatusBadRequest,
			body: mustCreateBody(&JSONRPCRequest{
				ID:      1,
				JSONRPC: "2.0",
				Method:  "node_echo",
				Params: json.RawMessage(mustMarshal(&blockchain.Transaction{
					Nonce: 0,
				})),
			}),
			expectedResp: mustMarshal(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &JSONRPCErrorResponse{
					Code:    jsonRPCInvalidRequest,
					Message: "invalid nonce",
				},
			}),
		},
		{
			method:     "POST",
			statusCode: http.StatusBadRequest,
			body: mustCreateBody(&JSONRPCRequest{
				ID:      1,
				JSONRPC: "1.0",
				Method:  "node_echo",
				Params: json.RawMessage(mustMarshal(&blockchain.Transaction{
					Nonce: 0,
				})),
			}),
			expectedResp: mustMarshal(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &JSONRPCErrorResponse{
					Code:    jsonRPCInvalidRequestError,
					Message: "invalid json rpc version 1.0",
				},
			}),
		},
		{
			method:     "POST",
			statusCode: http.StatusOK,
			body: mustCreateBody(&JSONRPCRequest{
				ID:      2,
				JSONRPC: "2.0",
				Method:  "node_echo",
				Params: json.RawMessage(mustMarshal(&blockchain.Transaction{
					Nonce: 100,
				})),
			}),
			expectedResp: mustMarshal(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      2,
				Result: json.RawMessage(mustMarshal(&blockchain.Transaction{
					Nonce: 100,
				})),
			}),
		},
	}

	for i, testCase := range testTable {
		t.Logf("Testing case %d", i)
		req, err := http.NewRequest(testCase.method, fmt.Sprintf("http://localhost:%d/rpc", port), testCase.body)
		assert.NoError(t, err)
		resp, err := cli.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, testCase.statusCode, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, testCase.expectedResp, string(body))
	}
}
