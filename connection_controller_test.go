package main

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectionBlocking(t *testing.T) {
	// Reset state
	atomic.StoreUint32(&connectionBlocked, 0)

	// Test initial state
	if IsBlocked() {
		t.Error("Connection should not be blocked initially")
	}

	// Test blocking
	BlockConnections(100 * time.Millisecond)
	if !IsBlocked() {
		t.Error("Connection should be blocked after BlockConnections call")
	}

	// Test automatic unblocking
	time.Sleep(150 * time.Millisecond) // Wait a bit longer than block duration
	if IsBlocked() {
		t.Error("Connection should be unblocked after duration")
	}
}

func TestConcurrentBlocking(t *testing.T) {
	// Reset state
	atomic.StoreUint32(&connectionBlocked, 0)

	// Start multiple goroutines that block/unblock
	for i := 0; i < 10; i++ {
		go func() {
			BlockConnections(50 * time.Millisecond)
		}()
	}

	// Verify that we can check status without race conditions
	for i := 0; i < 20; i++ {
		IsBlocked()
		time.Sleep(10 * time.Millisecond)
	}
}
