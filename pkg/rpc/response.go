package rpc

import (
	"encoding/json"
	"fmt"
)

type EndpointResponseWriter interface {
	Write(interface{})
	Error(error)
}

type endpointResponseWriter struct {
	data    interface{}
	err     error
	written bool
}

func NewEndpointResponseWriter() *endpointResponseWriter {
	return &endpointResponseWriter{}
}

func (a *endpointResponseWriter) Error(err error) {
	a.err = err
}

func (a *endpointResponseWriter) Write(data interface{}) {
	if a.written {
		panic(fmt.Errorf("data has already been written once"))
	}

	a.data = data
	a.written = true
}

func (a *endpointResponseWriter) Result() EndpointResponse {
	return NewEndpointResponse(a.data, a.err)
}

type EndpointResponse interface {
	Err() error
	Data() interface{}
	JSONData() ([]byte, error)
}

type endpointResponse struct {
	data interface{}
	err  error
}

func NewEndpointResponse(data interface{}, err error) EndpointResponse {
	return &endpointResponse{
		data: data,
		err:  err,
	}
}

func (a *endpointResponse) Err() error {
	return a.err
}

func (a *endpointResponse) Data() interface{} {
	return a.data
}

func (a *endpointResponse) JSONData() ([]byte, error) {
	return json.Marshal(a.data)
}
