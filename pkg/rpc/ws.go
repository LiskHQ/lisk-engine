package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	stringsUtil "github.com/LiskHQ/lisk-engine/pkg/collection/strings"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

var upgrader = websocket.Upgrader{}

type wsJSONRPCServer struct {
	logger      log.Logger
	httpServer  *http.Server
	connections map[string]*wsSocket
	invoker     Invoker
}

func NewWSJSONRPCServer(logger log.Logger, port int, addr string, invoker Invoker) *wsJSONRPCServer {
	server := &wsJSONRPCServer{
		logger:      logger,
		connections: make(map[string]*wsSocket),
		invoker:     invoker,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc-ws", server.handleUpgrade)
	bindingAddr := addr
	if addr == "" {
		bindingAddr = "127.0.0.1"
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", bindingAddr, port),
		Handler:           mux,
		ReadHeaderTimeout: 1 * time.Second,
	}
	server.httpServer = httpServer
	return server
}

func NewWSJSONRPCServerWithHTTPServer(logger log.Logger, invoker Invoker, mux *http.ServeMux, httpServer *http.Server) *wsJSONRPCServer {
	server := &wsJSONRPCServer{
		logger:      logger,
		connections: make(map[string]*wsSocket),
		invoker:     invoker,
	}
	mux.HandleFunc("/rpc-ws", server.handleUpgrade)

	httpServer.Handler = mux
	server.httpServer = httpServer
	return server
}

func (s *wsJSONRPCServer) Publish(method string, data []byte) {
	for _, conn := range s.connections {
		go conn.publish(method, data)
	}
}

func (s *wsJSONRPCServer) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *wsJSONRPCServer) Close() error {
	for _, conn := range s.connections {
		conn.close()
	}

	return s.httpServer.Close()
}

func (s *wsJSONRPCServer) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorf("Fail to upgrade connection with %s", err)
	}
	socket := newWSSocket(s.logger, conn, s.invoker)
	s.connections[r.RemoteAddr] = socket
	go socket.dispatch()
}

type wsMessage struct {
	msg *JSONRPCRequest
	err error
}

type wsSendingMessage struct {
	msg *JSONRPCRequest
}

type wsSocket struct {
	logger    log.Logger
	conn      *websocket.Conn
	invoker   Invoker
	receiver  chan wsMessage
	publisher chan wsSendingMessage
	closeChan chan bool
	topics    []string
}

func newWSSocket(logger log.Logger, conn *websocket.Conn, invoker Invoker) *wsSocket {
	return &wsSocket{
		conn:      conn,
		logger:    logger,
		invoker:   invoker,
		topics:    []string{},
		receiver:  make(chan wsMessage, 1),
		publisher: make(chan wsSendingMessage),
		closeChan: make(chan bool),
	}
}
func (c *wsSocket) publish(method string, data []byte) {
	for _, topic := range c.topics {
		if strings.Contains(method, topic) {
			c.publisher <- wsSendingMessage{
				msg: &JSONRPCRequest{
					ID:      0,
					JSONRPC: "2.0",
					Method:  method,
					Params:  data,
				},
			}
			return
		}
	}
}

func (c *wsSocket) dispatch() {
	defer c.conn.Close()
	go c.read()

	for {
		select {
		case <-c.closeChan:
			return
		case req := <-c.receiver:
			if req.err != nil {
				close(c.closeChan)
				break
			}
			ctx := context.Background()
			c.handleMessage(ctx, req.msg)
		case broadcast := <-c.publisher:
			c.writeJSON(broadcast.msg)
		}
	}
}
func (c *wsSocket) handleMessage(ctx context.Context, req *JSONRPCRequest) {
	if err := req.Validate(); err != nil {
		result := getErrResponse(req.ID, err, jsonRPCInvalidRequestError)
		c.write(result)
		return
	}
	if req.Method == "subscribe" {
		if err := c.handleSubscribe(ctx, req); err != nil {
			c.write(getErrResponse(req.ID, err, jsonRPCInvalidRequestError))
		} else {
			c.write(getSuccessResponse(req.ID, []byte{}))
		}
		return
	}
	if req.Method == "unsubscribe" {
		if err := c.handleUnsubscribe(ctx, req); err != nil {
			c.write(getSuccessResponse(req.ID, []byte{}))
		} else {
			c.write(getErrResponse(req.ID, err, jsonRPCInvalidRequestError))
		}
		return
	}
	c.logger.Infof("Received request from %s for method %s", c.conn.RemoteAddr(), req.Method)
	result := c.invoker.Invoke(ctx, req.Method, req.Params)
	if err := result.Err(); err != nil {
		result := getErrResponse(req.ID, err, jsonRPCInvalidRequest)
		c.write(result)
		return
	}
	resultData, err := result.JSONData()
	if err != nil {
		result := getErrResponse(req.ID, err, jsonRPCInvalidRequest)
		c.write(result)
		return
	}
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultData,
	}
	parsedResult, err := json.Marshal(resp)
	if err != nil {
		result := getErrResponse(req.ID, err, jsonRPCInvalidRequest)
		c.write(result)
		return
	}
	c.write(parsedResult)
}

type topicParams struct {
	Topics []string `json:"topics"`
}

func (c *wsSocket) handleSubscribe(ctx context.Context, req *JSONRPCRequest) error {
	params := &topicParams{}
	if err := json.Unmarshal(req.Params, params); err != nil {
		return err
	}
	if len(params.Topics) == 0 {
		return errors.New("topics to subscribe is empty")
	}
	for _, topic := range params.Topics {
		if !stringsUtil.Contain(c.topics, topic) {
			c.topics = append(c.topics, topic)
		}
	}
	return nil
}

func (c *wsSocket) handleUnsubscribe(ctx context.Context, req *JSONRPCRequest) error {
	params := &topicParams{}
	if err := json.Unmarshal(req.Params, params); err != nil {
		return err
	}
	newTopics := []string{}
	for _, topic := range c.topics {
		if !stringsUtil.Contain(params.Topics, topic) {
			newTopics = append(newTopics, topic)
		}
	}
	c.topics = newTopics
	return nil
}

func (c *wsSocket) read() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Errorf("Fail to read message with %s. Closing connection with %s", err, c.conn.RemoteAddr())
			c.receiver <- wsMessage{
				err: err,
			}
			return
		}
		req := &JSONRPCRequest{}
		if err := json.Unmarshal(msg, req); err != nil {
			c.logger.Errorf("Fail to unmarshal message with %s. Closing connection with %s", err, c.conn.RemoteAddr())
			c.receiver <- wsMessage{
				err: err,
			}
			return
		}
		if err := req.Validate(); err != nil {
			c.logger.Errorf("Fail to unmarshal message with %s. Closing connection with %s", err, c.conn.RemoteAddr())
			c.receiver <- wsMessage{
				err: err,
			}
			return
		}
		c.receiver <- wsMessage{
			msg: req,
		}
	}
}

func (c *wsSocket) writeJSON(body interface{}) {
	if err := c.conn.WriteJSON(body); err != nil {
		c.logger.Errorf("Fail to write json with %w", err)
	}
}

func (c *wsSocket) write(body []byte) {
	if err := c.conn.WriteMessage(websocket.TextMessage, body); err != nil {
		c.logger.Errorf("Fail to write message with %w", err)
	}
}

func (c *wsSocket) close() {
	close(c.closeChan)
}
