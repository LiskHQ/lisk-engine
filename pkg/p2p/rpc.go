package p2p

// ResponseWriter is an interface for handler to write.
type ResponseWriter interface {
	Write([]byte)
	Error(error)
}

// responseWriter is a struct that implements ResponseWriter interface.
type responseWriter struct {
	data []byte
	err  error
}

// Write writes data as a response.
func (w *responseWriter) Write(data []byte) {
	w.data = data
}

// Error writes error as a response.
func (w *responseWriter) Error(err error) {
	w.err = err
}

// Event holds event message from a peer.
type Response struct {
	timestamp int64  // Unix time when the message was received.
	peerID    string // ID of peer that created the response message.
	data      []byte // Response data.
	err       error  // Error message in case of an error.
}

// newResponse creates a new Response struct.
func newResponse(timestamp int64, peerID string, data []byte, err error) *Response {
	return &Response{
		timestamp: timestamp,
		peerID:    peerID,
		data:      data,
		err:       err,
	}
}

// Timestamp returns response timestamp.
func (r *Response) Timestamp() int64 {
	return r.timestamp
}

// PeerID returns sender peer ID.
func (r *Response) PeerID() string {
	return r.peerID
}

// Data returns response payload.
func (r *Response) Data() []byte {
	return r.data
}

// Error returns response error.
func (r *Response) Error() error {
	return r.err
}

// RPCHandler is a handler function for RPC request received from a peer.
type RPCHandler func(w ResponseWriter, req *RequestMsg)
