package main

import (
	"github.com/gorilla/websocket"
)

// WSConn is an interface that captures the websocket.Conn methods we use
type WSConn interface {
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// MockWSConn implements WSConn for testing
type MockWSConn struct {
	messages [][]byte
	closed   bool
}

func NewMockWSConn() *MockWSConn {
	return &MockWSConn{
		messages: make([][]byte, 0),
	}
}

func (m *MockWSConn) WriteMessage(messageType int, data []byte) error {
	if m.closed {
		return websocket.ErrCloseSent
	}
	m.messages = append(m.messages, data)
	return nil
}

func (m *MockWSConn) Close() error {
	m.closed = true
	return nil
}

func (m *MockWSConn) GetMessages() [][]byte {
	return m.messages
}

func (m *MockWSConn) IsClosed() bool {
	return m.closed
}
