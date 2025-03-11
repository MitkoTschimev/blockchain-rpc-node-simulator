package main

import (
	"sync"
)

// ConnectionTracker keeps track of active connections per chain
type ConnectionTracker struct {
	connections sync.Map // maps chainId to connection count (string -> int)
}

// NewConnectionTracker creates a new connection tracker
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{}
}

// AddConnection increments the connection count for a chain
func (ct *ConnectionTracker) AddConnection(chainId string) int {
	for {
		value, loaded := ct.connections.Load(chainId)
		if !loaded {
			if _, loaded := ct.connections.LoadOrStore(chainId, 1); !loaded {
				return 1
			}
			continue
		}
		count := value.(int)
		if ct.connections.CompareAndSwap(chainId, count, count+1) {
			return count + 1
		}
	}
}

// RemoveConnection decrements the connection count for a chain
func (ct *ConnectionTracker) RemoveConnection(chainId string) int {
	for {
		value, loaded := ct.connections.Load(chainId)
		if !loaded {
			return 0
		}
		count := value.(int)
		if count <= 1 {
			ct.connections.Delete(chainId)
			return 0
		}
		if ct.connections.CompareAndSwap(chainId, count, count-1) {
			return count - 1
		}
	}
}

// GetConnections returns a copy of the current connection counts
func (ct *ConnectionTracker) GetConnections() map[string]int {
	counts := make(map[string]int)
	ct.connections.Range(func(key, value interface{}) bool {
		if chainId, ok := key.(string); ok {
			if count, ok := value.(int); ok {
				counts[chainId] = count
			}
		}
		return true
	})
	return counts
}

// GetConnectionCount returns the current connection count for a specific chain
func (ct *ConnectionTracker) GetConnectionCount(chainId string) int {
	if value, ok := ct.connections.Load(chainId); ok {
		if count, ok := value.(int); ok {
			return count
		}
	}
	return 0
}

// HasConnections returns true if there are any active connections for the given chain
func (ct *ConnectionTracker) HasConnections(chainId string) bool {
	_, exists := ct.connections.Load(chainId)
	return exists
}
