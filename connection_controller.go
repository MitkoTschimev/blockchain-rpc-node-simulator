package main

import (
	"sync/atomic"
	"time"
)

var (
	// 0 means accepting connections, 1 means blocking
	connectionBlocked uint32 = 0
)

// BlockConnections blocks new connections for the specified duration
func BlockConnections(duration time.Duration) {
	atomic.StoreUint32(&connectionBlocked, 1)

	go func() {
		time.Sleep(duration)
		atomic.StoreUint32(&connectionBlocked, 0)
	}()
}

// IsBlocked returns true if connections are currently blocked
func IsBlocked() bool {
	return atomic.LoadUint32(&connectionBlocked) == 1
}
