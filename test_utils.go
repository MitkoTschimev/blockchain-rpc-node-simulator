package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

// WSConn is an interface that captures the websocket.Conn methods we use
type WSConn interface {
	WriteMessage(messageType int, data []byte) error
	Close() error
	GetMessages() [][]byte
	ClearMessages()
}

// MockWSConn implements WSConn for testing
type MockWSConn struct {
	messages [][]byte
	closed   bool
	mu       sync.RWMutex
}

func NewMockWSConn() *MockWSConn {
	return &MockWSConn{
		messages: make([][]byte, 0),
	}
}

func (m *MockWSConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return websocket.ErrCloseSent
	}
	m.messages = append(m.messages, data)
	return nil
}

func (m *MockWSConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func (m *MockWSConn) GetMessages() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent race conditions
	messages := make([][]byte, len(m.messages))
	for i, msg := range m.messages {
		msgCopy := make([]byte, len(msg))
		copy(msgCopy, msg)
		messages[i] = msgCopy
	}
	return messages
}

func (m *MockWSConn) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.closed
}

func (m *MockWSConn) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = nil
}
