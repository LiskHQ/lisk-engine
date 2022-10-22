// Package labi_client implements ABI client using IPC protocol.
package labi_client

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/go-zeromq/zmq4"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type reqOp struct {
	id      uint64
	msgChan chan *ipcResponse
}

type ipcClient struct {
	ctx       context.Context
	logger    log.Logger
	client    zmq4.Socket
	globalID  uint64
	mutex     *sync.Mutex
	reqOp     map[uint64]*reqOp
	closeConn chan struct{}
}

type zmqMsg struct {
	body zmq4.Msg
	err  error
}

func NewIPCClient(logger log.Logger) *ipcClient {
	ctx := context.Background()
	dealer := zmq4.NewDealer(ctx, zmq4.WithID([]byte("lisk-engine_dealer")))
	client := &ipcClient{
		ctx:       ctx,
		logger:    logger,
		client:    dealer,
		reqOp:     map[uint64]*reqOp{},
		mutex:     &sync.Mutex{},
		closeConn: make(chan struct{}),
		globalID:  0,
	}
	return client
}

type ipcRequest struct {
	id     uint64 `fieldNumber:"1"`
	method string `fieldNumber:"2"`
	params []byte `fieldNumber:"3"`
}

type errObj struct {
	message string `fieldNumber:"1"`
}

type ipcResponse struct {
	id      uint64  `fieldNumber:"1"`
	success bool    `fieldNumber:"2"`
	err     *errObj `fieldNumber:"3"`
	result  []byte  `fieldNumber:"4"`
}

func (c *ipcClient) Start(path string) error {
	if err := c.client.Dial(path); err != nil {
		return err
	}
	go c.start()
	return nil
}

func (c *ipcClient) start() {
	msgReceiver := make(chan zmqMsg)
	go c.read(msgReceiver)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.closeConn:
			return
		case msg := <-msgReceiver:
			if msg.err != nil {
				c.logger.Errorf("Fail to receive message from server with %s", msg.err.Error())
				continue
			}
			r := &ipcResponse{}
			if err := r.Decode(msg.body.Frames[0]); err != nil {
				r.success = false
				r.err = &errObj{
					message: err.Error(),
				}
			}
			c.mutex.Lock()
			if op, exist := c.reqOp[r.id]; exist {
				op.msgChan <- r
				delete(c.reqOp, r.id)
			}
			c.mutex.Unlock()
		}
	}
}

func (c *ipcClient) Close() error {
	close(c.closeConn)
	return c.client.Close()
}

func (c *ipcClient) Init(req *labi.InitRequest) (*labi.InitResponse, error) {
	res := &labi.InitResponse{}
	if err := c.call("init", req, nil); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) InitStateMachine(req *labi.InitStateMachineRequest) (*labi.InitStateMachineResponse, error) {
	res := &labi.InitStateMachineResponse{}
	if err := c.call("initStateMachine", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) InitGenesisState(req *labi.InitGenesisStateRequest) (*labi.InitGenesisStateResponse, error) {
	res := &labi.InitGenesisStateResponse{}
	if err := c.call("initGenesisState", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) InsertAssets(req *labi.InsertAssetsRequest) (*labi.InsertAssetsResponse, error) {
	res := &labi.InsertAssetsResponse{}
	if err := c.call("insertAssets", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) VerifyAssets(req *labi.VerifyAssetsRequest) (*labi.VerifyAssetsResponse, error) {
	res := &labi.VerifyAssetsResponse{}
	if err := c.call("verifyAssets", req, nil); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) BeforeTransactionsExecute(req *labi.BeforeTransactionsExecuteRequest) (*labi.BeforeTransactionsExecuteResponse, error) {
	res := &labi.BeforeTransactionsExecuteResponse{}
	if err := c.call("beforeTransactionsExecute", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) AfterTransactionsExecute(req *labi.AfterTransactionsExecuteRequest) (*labi.AfterTransactionsExecuteResponse, error) {
	res := &labi.AfterTransactionsExecuteResponse{}
	if err := c.call("afterTransactionsExecute", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) VerifyTransaction(req *labi.VerifyTransactionRequest) (*labi.VerifyTransactionResponse, error) {
	res := &labi.VerifyTransactionResponse{}
	if err := c.call("verifyTransaction", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) ExecuteTransaction(req *labi.ExecuteTransactionRequest) (*labi.ExecuteTransactionResponse, error) {
	res := &labi.ExecuteTransactionResponse{}
	if err := c.call("executeTransaction", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) Commit(req *labi.CommitRequest) (*labi.CommitResponse, error) {
	res := &labi.CommitResponse{}
	if err := c.call("commit", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) Revert(req *labi.RevertRequest) (*labi.RevertResponse, error) {
	res := &labi.RevertResponse{}
	if err := c.call("revert", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) Clear(req *labi.ClearRequest) (*labi.ClearResponse, error) {
	res := &labi.ClearResponse{}
	if err := c.call("clear", nil, nil); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) Finalize(req *labi.FinalizeRequest) (*labi.FinalizeResponse, error) {
	res := &labi.FinalizeResponse{}
	if err := c.call("finalize", req, nil); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) GetMetadata(req *labi.MetadataRequest) (*labi.MetadataResponse, error) {
	res := &labi.MetadataResponse{}
	if err := c.call("getMetadata", nil, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) Query(req *labi.QueryRequest) (*labi.QueryResponse, error) {
	res := &labi.QueryResponse{}
	if err := c.call("query", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) Prove(req *labi.ProveRequest) (*labi.ProveResponse, error) {
	res := &labi.ProveResponse{}
	if err := c.call("prove", req, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ipcClient) call(method string, req codec.Encodable, res codec.Decodable) error {
	op, err := c.request(method, req)
	if err != nil {
		return err
	}
	msg := <-op.msgChan
	if !msg.success {
		return errors.New(msg.err.message)
	}
	if res != nil {
		if err := res.Decode(msg.result); err != nil {
			return err
		}
	}
	return nil
}

func (c *ipcClient) request(method string, req codec.Encodable) (*reqOp, error) {
	encodedReq := []byte{}
	if req != nil {
		var err error
		encodedReq, err = req.Encode()
		if err != nil {
			return nil, err
		}
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.globalID == math.MaxUint64 {
		c.globalID = 0
	}
	c.globalID += 1
	reqData := &ipcRequest{
		id:     c.globalID,
		method: method,
		params: encodedReq,
	}
	encodedReqData, err := reqData.Encode()
	if err != nil {
		return nil, err
	}
	if err := c.client.Send(zmq4.NewMsg(encodedReqData)); err != nil {
		return nil, err
	}
	op := &reqOp{
		id:      c.globalID,
		msgChan: make(chan *ipcResponse),
	}
	c.reqOp[c.globalID] = op
	return op, nil
}

func (c *ipcClient) read(receiver chan<- zmqMsg) {
	for {
		msg, err := c.client.Recv()
		receiver <- zmqMsg{
			body: msg,
			err:  err,
		}
	}
}
