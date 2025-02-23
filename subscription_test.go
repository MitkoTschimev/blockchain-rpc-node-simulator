package main

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionManager(t *testing.T) {
	sm := NewSubscriptionManager()

	// Test subscription creation
	conn := NewMockWSConn()
	subID, err := sm.Subscribe("evm", conn, "newHeads")
	if err != nil {
		t.Errorf("Failed to create subscription: %v", err)
	}
	if subID == "" {
		t.Error("Subscription ID should not be empty")
	}

	// Test duplicate subscription
	subID2, err := sm.Subscribe("evm", conn, "newHeads")
	if err != nil {
		t.Error("Should allow multiple subscriptions from same connection")
	}
	if subID == subID2 {
		t.Error("Subscription IDs should be unique")
	}

	// Test broadcast
	sm.BroadcastNewBlock(1000)
	messages := conn.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Verify message format
	var notification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &notification); err != nil {
		t.Errorf("Invalid notification format: %v", err)
	}
	if notification.Method != "eth_subscription" {
		t.Errorf("Expected eth_subscription method, got %s", notification.Method)
	}

	// Test unsubscribe
	if err := sm.Unsubscribe(subID); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}

	// Test unsubscribe of non-existent subscription
	if err := sm.Unsubscribe("non-existent"); err == nil {
		t.Error("Should return error for non-existent subscription")
	}

	// Test drop all connections
	count := sm.DropAllConnections()
	if count != 1 { // We still have one subscription left
		t.Errorf("Expected 1 connection to be dropped, got %d", count)
	}
	if !conn.IsClosed() {
		t.Error("Connection should be closed after DropAllConnections")
	}
}

func TestSubscriptionManagerConcurrent(t *testing.T) {
	sm := NewSubscriptionManager()
	done := make(chan bool)

	// Start a goroutine that continuously broadcasts blocks
	go func() {
		for i := 0; i < 100; i++ {
			sm.BroadcastNewBlock(uint64(i))
		}
		done <- true
	}()

	// Start goroutines that add and remove subscriptions
	for i := 0; i < 10; i++ {
		go func() {
			conn := NewMockWSConn()
			subID, _ := sm.Subscribe("evm", conn, "newHeads")
			sm.Unsubscribe(subID)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 11; i++ {
		<-done
	}
}
