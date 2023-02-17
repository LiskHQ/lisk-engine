// Package socket implements socket communications using WS protocol.
package socket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oklog/ulid"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

var (
	ErrSocketClose      = errors.New("socket is closed")
	ErrNoResult         = errors.New("no result was given in the RPC response")
	ErrNotSupportedType = errors.New("response not supported type")
	ErrRPCTimeout       = errors.New("rPC timeout")
	ErrSocketClosed     = errors.New("socket closed already")
	idEntropy           = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const (
	tcpKeepAliveInterval       = 30 * time.Second
	ackTimeout                 = 3 * time.Second
	writeTimeout               = 3 * time.Second
	countResetInterval         = 10 * time.Second
	maxMessagePerInterval      = 1000
	pingTimeout                = 100000
	rpcRequestKey              = "rpc-request"
	rpcEventKey                = "remote-message"
	eventSocketPing            = "ping"
	eventSocketPong            = "pong"
	eventSocketHandshake       = "#handshake"
	eventSocketDisconnect      = "#disconnect"
	eventSocketMessageReceived = "eventSocketMessageReceived"
)

// Connector function for reconnection.
type Connector func() (*websocket.Conn, error)

// Socket represents a bi-directional connection to a node.
type Socket struct {
	id                  string
	conn                *websocket.Conn
	address             net.Addr
	isConnected         bool
	isServer            bool
	logger              log.Logger
	counter             *counter
	sendMutex           *sync.Mutex
	receiveMutex        *sync.Mutex
	requestHandlers     map[string]RPCHandler
	handshakeHandler    HandshakeHandler
	connector           Connector
	eventCh             chan *Message
	messageCounter      *counter
	messageCounterReset *time.Ticker

	closeConn   chan struct{}
	didQuit     chan struct{}
	reconnected chan *websocket.Conn
	readOp      chan *scRPCMessage
	readError   chan error
	requestOp   chan *requestOp
	reqSent     chan error
	respWait    map[int]*requestOp
	reqTimeout  chan *requestOp
}

type jsonError struct {
	Message string `json:"message"`
}

type handshakeResponse struct {
	ID              string `json:"id"`
	IsAuthenticated bool   `json:"isAuthenticated"`
	PingTimeout     int    `json:"pingTimeout"`
}

type scRPCMessage struct {
	CID   int             `json:"cid,omitempty"`
	RID   int             `json:"rid,omitempty"`
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *jsonError      `json:"error,omitempty"`
}

func (scm *scRPCMessage) isRequest() bool {
	return scm.CID != 0 && scm.Event == rpcRequestKey
}

func (scm *scRPCMessage) isResponse() bool {
	return scm.RID != 0
}

func (scm *scRPCMessage) isEvent() bool {
	return scm.Event == rpcEventKey
}

func (scm *scRPCMessage) isPing() bool {
	return scm.Event == eventSocketPing
}

type requestOp struct {
	cid  int
	err  error
	resp chan *scRPCMessage
}

func (rop *requestOp) wait(ctx context.Context, s *Socket) (*scRPCMessage, error) {
	select {
	case <-ctx.Done():
		select {
		case s.reqTimeout <- rop:
		case <-s.closeConn:
		}
		return nil, ctx.Err()
	case resp := <-rop.resp:
		return resp, rop.err
	case <-time.After(ackTimeout):
		return nil, ErrRPCTimeout
	case <-s.closeConn:
		return nil, ErrSocketClose
	}
}

type jsonRPCMessage struct {
	Procedure string          `json:"procedure,omitempty"`
	Data      string          `json:"data,omitempty"`
	Error     json.RawMessage `json:"error,omitempty"`
}

type jsonEventMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// NewClientSocket creates a socket as a client socket.
func NewClientSocket(ctx context.Context, conn *websocket.Conn, connector Connector, logger log.Logger) (*Socket, error) {
	socket := &Socket{
		conn:                conn,
		logger:              logger,
		connector:           connector,
		address:             conn.RemoteAddr(),
		counter:             &counter{count: 0},
		eventCh:             make(chan *Message),
		messageCounterReset: time.NewTicker(countResetInterval),
		messageCounter:      &counter{count: 0},
		requestHandlers:     make(map[string]RPCHandler),
		isConnected:         false,
		sendMutex:           &sync.Mutex{},
		receiveMutex:        &sync.Mutex{},
		closeConn:           make(chan struct{}),
		reconnected:         make(chan *websocket.Conn),
		didQuit:             make(chan struct{}),
		readError:           make(chan error),
		readOp:              make(chan *scRPCMessage),
		requestOp:           make(chan *requestOp),
		reqSent:             make(chan error),
		respWait:            make(map[int]*requestOp),
		reqTimeout:          make(chan (*requestOp)),
	}
	go socket.dispatch()
	if err := socket.sendHandshake(ctx); err != nil {
		// Fail to handshake
		socket.isConnected = false
		return nil, err
	}
	return socket, nil
}

// NewServerSocket creates a socket as a server socket.
func NewServerSocket(conn *websocket.Conn, logger log.Logger) *Socket {
	socket := &Socket{
		conn:                conn,
		logger:              logger,
		isServer:            true,
		messageCounterReset: time.NewTicker(countResetInterval),
		messageCounter:      &counter{count: 0},
		requestHandlers:     make(map[string]RPCHandler),
		address:             conn.RemoteAddr(),
		counter:             &counter{count: 0},
		eventCh:             make(chan *Message),
		isConnected:         true,
		sendMutex:           &sync.Mutex{},
		receiveMutex:        &sync.Mutex{},
		closeConn:           make(chan struct{}),
		didQuit:             make(chan struct{}),
		readError:           make(chan error),
		readOp:              make(chan *scRPCMessage),
		requestOp:           make(chan *requestOp),
		reqSent:             make(chan error),
		respWait:            make(map[int]*requestOp),
		reqTimeout:          make(chan (*requestOp)),
	}
	go socket.dispatch()
	return socket
}

// DispatchServer.
func (s *Socket) StartServerProcess() {
	if s.isServer {
		s.dispatch()
	}
}

// RegisterRPCHandlers registers a request handler.
func (s *Socket) RegisterRPCHandlers(handlers map[string]RPCHandler) {
	for key, handler := range handlers {
		s.requestHandlers[key] = handler
	}
}

// RegisterHandshakeHandler registers a handshake handler.
func (s *Socket) RegisterHandshakeHandler(handler HandshakeHandler) {
	s.handshakeHandler = handler
}

// IsServer returns true if the socket is server socket.
func (s *Socket) IsServer() bool {
	return s.isServer
}

// Subscribe returns receive only message channel.
func (s *Socket) Subscribe() <-chan *Message {
	return s.eventCh
}

// Send event to the socket.
func (s *Socket) Send(ctx context.Context, event string, data []byte) error {
	jsonMsg := &jsonEventMessage{
		Event: event,
		Data:  base64.StdEncoding.EncodeToString(data),
	}
	content, err := json.Marshal(jsonMsg)
	if err != nil {
		return err
	}
	id := s.counter.IncrementAndGet()
	reqData := &scRPCMessage{
		CID:   id,
		Event: rpcEventKey,
		Data:  content,
	}
	msg, err := json.Marshal(reqData)
	if err != nil {
		return err
	}
	op := &requestOp{
		cid:  id,
		resp: make(chan *scRPCMessage, 1),
	}
	err = s.emit(ctx, op, msg)
	if err != nil {
		return err
	}
	return nil
}

// Request a response from the socket.
func (s *Socket) Request(ctx context.Context, procedure string, data []byte) Response {
	jsonMsg := &jsonRPCMessage{
		Procedure: procedure,
		Data:      base64.StdEncoding.EncodeToString(data),
	}
	content, err := json.Marshal(jsonMsg)
	if err != nil {
		return NewResponse(nil, err)
	}
	id := s.counter.IncrementAndGet()
	reqData := &scRPCMessage{
		CID:   id,
		Event: rpcRequestKey,
		Data:  content,
	}
	msg, err := json.Marshal(reqData)
	if err != nil {
		return NewResponse(nil, err)
	}
	op := &requestOp{
		cid:  id,
		resp: make(chan *scRPCMessage, 1),
	}
	err = s.emit(ctx, op, msg)
	if err != nil {
		return NewResponse(nil, err)
	}
	switch resp, err := op.wait(ctx, s); {
	case err != nil:
		return NewResponse(nil, err)
	case resp.Error != nil:
		return NewResponse(nil, errors.New(resp.Error.Message))
	case resp.Data == nil:
		return NewResponse(nil, ErrNoResult)
	default:
		jsonMsg := &jsonRPCMessage{}
		if err := json.Unmarshal(resp.Data, jsonMsg); err != nil {
			return NewResponse(nil, err)
		}
		data, err := base64.StdEncoding.DecodeString(jsonMsg.Data)
		if err != nil {
			return NewResponse(nil, err)
		}
		return NewResponse(data, err)
	}
}

// Ping send ping and return latency.
func (s *Socket) Ping(ctx context.Context) (int64, error) {
	id := s.counter.IncrementAndGet()
	jsonMsg := &jsonRPCMessage{
		Procedure: "ping",
		Data:      "",
	}
	marhaledMsg, err := json.Marshal(jsonMsg)
	if err != nil {
		return 0, err
	}
	reqData := &scRPCMessage{
		CID:   id,
		Event: rpcRequestKey,
		Data:  marhaledMsg,
	}
	msg, err := json.Marshal(reqData)
	if err != nil {
		return 0, err
	}
	op := &requestOp{
		cid:  id,
		resp: make(chan *scRPCMessage, 1),
	}
	start := time.Now()
	err = s.emit(ctx, op, msg)
	if err != nil {
		return 0, err
	}
	_, err = op.wait(ctx, s)
	if err != nil {
		return 0, err
	}
	duration := time.Since(start)
	return duration.Milliseconds(), nil
}

// Close socket connection.
func (s *Socket) Close() {
	s.CloseWithReason(CloseIntentional)
}

// CloseWithReason closes socket connection.
func (s *Socket) CloseWithReason(code DisconnectCode) {
	reason, exist := closeCodes[code]
	if !exist {
		reason = "Closing with unknown reason"
	}
	s.logger.Debug("Closing socket with reason", code, reason)
	s.sendMutex.Lock()
	err := s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(int(code), reason))
	s.sendMutex.Unlock()
	if err != nil {
		s.logger.Errorf("Fail to close with %v", err)
		return
	}
	close(s.closeConn)
	s.isConnected = false
}

func (s *Socket) dispatch() {
	var (
		lastOp        *requestOp
		requestOpLock = s.requestOp
		reading       = true
	)
	defer func() {
		s.messageCounterReset.Stop()
		if reading {
			s.conn.Close()
			s.drainRead()
		}
		// close(s.eventCh)
		// close(s.didQuit)
	}()

	go s.read() //nolint: errcheck // reading cycle is handled in channel

	for {
		select {
		case <-s.closeConn:
			return
		case <-s.messageCounterReset.C:
			s.messageCounter.Reset()
		case message := <-s.readOp:
			switch {
			case message.isRequest():
				s.handleRequest(message)
			case message.isResponse():
				s.handleResponse(message)
			case message.isEvent():
				s.handleEvent(message)
			case message.isPing():
				s.handlePing(message)
			default:
				s.handleMessage(message)
			}
		case err := <-s.readError:
			s.closeRequestOps(err)
			s.sendEventWithoutBlocking(newErrorMessage(err))
			s.conn.Close()
			reading = false
		case <-s.reconnected:
			s.logger.Debug("Reconnected")
			if reading {
				s.conn.Close()
				s.drainRead()
			}
			go s.read() //nolint: errcheck // reading cycle is handled in channel
			reading = true
		case op := <-requestOpLock:
			requestOpLock = nil
			lastOp = op
			s.respWait[op.cid] = op
		case err := <-s.reqSent:
			if err != nil {
				s.logger.Error("Send error", err)
				delete(s.respWait, lastOp.cid)
			}
			requestOpLock = s.requestOp
			lastOp = nil
		case op := <-s.reqTimeout:
			delete(s.respWait, op.cid)
		}
	}
}

func (s *Socket) closeRequestOps(err error) {
	didClose := make(map[*requestOp]bool)

	for id, op := range s.respWait {
		// Remove the op so that later calls will not close op.resp again.
		delete(s.respWait, id)

		if !didClose[op] {
			op.err = err
			close(op.resp)
			didClose[op] = true
		}
	}
}

func (s *Socket) read() error {
	for {
		s.receiveMutex.Lock()
		msgType, msg, err := s.conn.ReadMessage()
		s.receiveMutex.Unlock()
		if err != nil {
			s.readError <- err
			return err
		}
		if string(msg) == "#1" {
			err := s.write([]byte("#2"))
			s.reqSent <- err
			continue
		}
		if msgType != websocket.TextMessage {
			s.logger.Error("Websocket received wrong type", msgType, ErrNotSupportedType)
			s.readError <- ErrNotSupportedType
			return ErrNotSupportedType
		}
		res := &scRPCMessage{}
		if err := json.Unmarshal(msg, res); err != nil {
			s.logger.Error("Failed to parse response", ErrNotSupportedType)
			s.readError <- err
			return err
		}
		s.readOp <- res
	}
}

func (s *Socket) sendHandshake(ctx context.Context) error {
	id := s.counter.IncrementAndGet()
	handshakeEvent := &scRPCMessage{
		Event: eventSocketHandshake,
		CID:   id,
	}
	handshakeEventData, parseErr := json.Marshal(handshakeEvent)
	if parseErr != nil {
		s.logger.Error("Failed to parse handshake model", parseErr)
	}
	op := &requestOp{
		cid:  id,
		resp: make(chan *scRPCMessage, 1),
	}
	err := s.emit(ctx, op, handshakeEventData)
	if err != nil {
		return err
	}
	switch resp, err := op.wait(ctx, s); {
	case err != nil:
		return err
	case resp.Error != nil:
		return errors.New(resp.Error.Message)
	case resp.Data == nil:
		return ErrNoResult
	default:
		jsonMsg := &handshakeResponse{}
		if err := json.Unmarshal(resp.Data, jsonMsg); err != nil {
			return err
		}
		s.id = jsonMsg.ID
		s.isConnected = true
		return nil
	}
}

func (s *Socket) emit(ctx context.Context, op *requestOp, data []byte) error {
	select {
	case s.requestOp <- op:
		err := s.write(data)
		s.reqSent <- err
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-s.didQuit:
		return ErrSocketClosed
	}
}

func (s *Socket) write(data []byte) error {
	s.sendMutex.Lock()
	err := s.conn.WriteMessage(websocket.TextMessage, data)
	s.sendMutex.Unlock()
	if err != nil {
		s.logger.Debug("Fail to write", err)
	}
	return err
}

func (s *Socket) handleResponse(msg *scRPCMessage) {
	op := s.respWait[msg.RID]
	if op == nil {
		s.logger.Debug("unassociated response", msg.RID)
		return
	}
	delete(s.respWait, msg.RID)
	op.resp <- msg
}

func (s *Socket) handleEvent(msg *scRPCMessage) {
	count := s.messageCounter.IncrementAndGet()
	if count > maxMessagePerInterval {
		s.eventCh <- newLimitExceededMessage()
		return
	}
	jsonMsg := &jsonEventMessage{}
	if err := json.Unmarshal(msg.Data, jsonMsg); err != nil {
		s.logger.Errorf("event message unmarshal error %v %s", err, msg.Data)
		s.sendEventWithoutBlocking(newErrorMessage(err))
		return
	}
	data, err := base64.StdEncoding.DecodeString(jsonMsg.Data)
	if err != nil {
		s.logger.Errorf("event data unmarshal error %v %s", err, msg.Data)
		s.sendEventWithoutBlocking(newErrorMessage(err))
		return
	}
	s.sendEventWithoutBlocking(newEventMessage(jsonMsg.Event, data))
}

func (s *Socket) handleMessage(msg *scRPCMessage) {
	count := s.messageCounter.IncrementAndGet()
	if count > maxMessagePerInterval {
		s.eventCh <- newLimitExceededMessage()
		return
	}
	s.logger.Debug("Event message received", msg.Event, string(msg.Data))
	if msg.Event == eventSocketHandshake {
		s.handleHandshake(msg)
		return
	}
	if msg.Event == eventSocketDisconnect {
		s.sendEventWithoutBlocking(newClosedMessage())
		s.Close()
		return
	}
	s.sendEventWithoutBlocking(newGeneralMessage(msg.Event, msg.Data))
}

func (s *Socket) handleRequest(msg *scRPCMessage) {
	count := s.messageCounter.IncrementAndGet()
	if count > maxMessagePerInterval {
		s.eventCh <- newLimitExceededMessage()
		return
	}
	jsonMsg := &jsonRPCMessage{}
	if err := json.Unmarshal(msg.Data, jsonMsg); err != nil {
		s.reqSent <- fmt.Errorf("failed to unmarshal request data")
		return
	}
	handler, exist := s.requestHandlers[jsonMsg.Procedure]
	if !exist {
		err := fmt.Errorf("procedure %s does not exist", jsonMsg.Procedure)
		s.reqSent <- err
		s.sendEventWithoutBlocking(newErrorMessage(err))
		return
	}
	s.logger.Debug("Handling procedure", jsonMsg.Procedure, s.address.String())
	req := &Request{
		ctx: context.Background(),
	}
	req.PeerID = s.address.String()
	req.Procedure = jsonMsg.Procedure
	reqData, err := base64.StdEncoding.DecodeString(jsonMsg.Data)
	if err != nil {
		s.logger.Errorf("Failed to encode response with %v", err)
		return
	}
	req.Data = reqData
	responseW := &responseWriter{}
	// TODO: Update this to handle in goroutine
	handler(responseW, req)
	jsonResp := &jsonRPCMessage{
		Data: base64.StdEncoding.EncodeToString(responseW.data),
	}

	marshaledData, err := json.Marshal(jsonResp)
	if err != nil {
		s.logger.Errorf("Failed to encode response with %v", err)
		return
	}

	response := &scRPCMessage{
		RID:  msg.CID,
		Data: marshaledData,
	}
	if responseW.err != nil {
		response.Error = &jsonError{
			Message: responseW.err.Error(),
		}
	}
	data, err := json.Marshal(response)
	if err != nil {
		s.logger.Errorf("Failed to marshal response with %v", err)
		return
	}
	if err := s.write(data); err != nil {
		s.logger.Errorf("Failed to write response with %v", err)
		return
	}
}

func (s *Socket) handleHandshake(msg *scRPCMessage) {
	response := &scRPCMessage{
		RID: msg.CID,
	}
	if err := s.handshakeHandler(s.conn); err != nil {
		response.Error = &jsonError{
			Message: err.Error(),
		}
		if data, err := json.Marshal(response); err != nil {
			s.logger.Error("Failed to marshal response", err)
		} else {
			err := s.write(data)
			s.reqSent <- err
		}
		s.Close()
		return
	}
	s.id = ulid.MustNew(ulid.Timestamp(time.Now()), idEntropy).String()
	handshakeStruct := &handshakeResponse{
		s.id,
		false,
		pingTimeout,
	}
	handshakeResp, err := json.Marshal(handshakeStruct)
	if err != nil {
		s.logger.Error("Failed to marshal handshake", err)
		return
	}
	response.Data = handshakeResp
	if data, err := json.Marshal(response); err != nil {
		s.logger.Error("Failed to marshal response", err)
	} else {
		err := s.write(data)
		s.reqSent <- err
	}
}

func (s *Socket) handlePing(msg *scRPCMessage) {
	response := &scRPCMessage{
		RID: msg.CID,
	}
	jsonData := &jsonRPCMessage{
		Data: "pong",
	}
	marshaled, err := json.Marshal(jsonData)
	if err != nil {
		s.logger.Errorf("Failed to marshal ping response with %v", err)
		return
	}
	response.Data = marshaled
	data, err := json.Marshal(response)
	if err != nil {
		s.logger.Errorf("Failed to marshal ping response with %v", err)
		return
	}
	if err = s.write(data); err != nil {
		s.logger.Errorf("Failed to write ping response with %v", err)
	}
}

// drainRead wait until there is an error.
func (s *Socket) drainRead() {
	for {
		select {
		case <-s.readOp:
		case <-s.readError:
			return
		}
	}
}

// drainRead wait until there is an error.
func (s *Socket) sendEventWithoutBlocking(msg *Message) {
	select {
	case s.eventCh <- msg:
	default:
		s.logger.Debug("Message is blocked or has no subscriber")
	}
}
