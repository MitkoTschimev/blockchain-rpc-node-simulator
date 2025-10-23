package main

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

// TestSolanaSlotNotificationBroadcast tests that Solana slot notifications are properly broadcast
func TestSolanaSlotNotificationBroadcast(t *testing.T) {
	// Create a mock WebSocket connection
	mockConn := NewMockWSConn()

	// Create a new subscription manager
	sm := NewSubscriptionManager()

	// Subscribe to Solana slot notifications
	subID, err := sm.Subscribe("501", mockConn, "slotNotification")
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Broadcast a new block/slot
	slotNumber := uint64(123)
	sm.BroadcastNewBlock("501", slotNumber)

	// Wait a bit for the message to be sent
	time.Sleep(100 * time.Millisecond)

	// Check that the mock connection received a message
	messages := mockConn.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected to receive a slot notification, but got none")
	}

	// Parse the notification
	var notification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &notification); err != nil {
		t.Fatalf("Failed to unmarshal notification: %v", err)
	}

	// Verify the notification structure
	if notification.Method != "slotNotification" {
		t.Errorf("Expected method 'slotNotification', got '%s'", notification.Method)
	}

	// Check the params
	params, ok := notification.Params.(map[string]interface{})
	if !ok {
		t.Fatal("Expected params to be a map")
	}

	// Verify subscription ID
	subscription, ok := params["subscription"].(float64)
	if !ok {
		t.Fatalf("Expected subscription to be a number, got %T", params["subscription"])
	}
	if uint64(subscription) != subID {
		t.Errorf("Expected subscription ID %d, got %d", subID, uint64(subscription))
	}

	// Verify the result contains slot information
	result, ok := params["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	slot, ok := result["slot"].(float64)
	if !ok {
		t.Fatalf("Expected slot to be a number, got %T", result["slot"])
	}
	if uint64(slot) != slotNumber {
		t.Errorf("Expected slot %d, got %d", slotNumber, uint64(slot))
	}

	t.Logf("Successfully received slot notification for slot %d", slotNumber)
}

// TestSolanaVsEVMBroadcast ensures that Solana and EVM broadcasts don't interfere
func TestSolanaVsEVMBroadcast(t *testing.T) {
	// Create mock connections
	solanaConn := NewMockWSConn()
	evmConn := NewMockWSConn()

	// Create a new subscription manager
	sm := NewSubscriptionManager()

	// Create a test EVM chain
	testChain := &EVMChain{
		Name:        "test",
		ChainID:     "1",
		BlockNumber: 100,
	}
	supportedChains["test"] = testChain

	// Subscribe to Solana
	solanaSubID, err := sm.Subscribe("501", solanaConn, "slotNotification")
	if err != nil {
		t.Fatalf("Failed to create Solana subscription: %v", err)
	}

	// Subscribe to EVM
	evmSubID, err := sm.Subscribe("1", evmConn, "newHeads")
	if err != nil {
		t.Fatalf("Failed to create EVM subscription: %v", err)
	}

	// Broadcast to Solana
	sm.BroadcastNewBlock("501", 200)
	time.Sleep(50 * time.Millisecond)

	// Verify Solana conn received message
	solanaMessages := solanaConn.GetMessages()
	if len(solanaMessages) != 1 {
		t.Errorf("Expected Solana to receive 1 message, got %d", len(solanaMessages))
	}

	// Verify EVM conn did NOT receive message
	evmMessages := evmConn.GetMessages()
	if len(evmMessages) != 0 {
		t.Errorf("Expected EVM to receive 0 messages from Solana broadcast, got %d", len(evmMessages))
	}

	// Clear messages
	solanaConn.ClearMessages()
	evmConn.ClearMessages()

	// Broadcast to EVM
	atomic.StoreUint64(&testChain.BlockNumber, 300)
	sm.BroadcastNewBlock("1", 300)
	time.Sleep(50 * time.Millisecond)

	// Verify EVM conn received message
	evmMessages = evmConn.GetMessages()
	if len(evmMessages) != 1 {
		t.Errorf("Expected EVM to receive 1 message, got %d", len(evmMessages))
	}

	// Verify Solana conn did NOT receive message
	solanaMessages = solanaConn.GetMessages()
	if len(solanaMessages) != 0 {
		t.Errorf("Expected Solana to receive 0 messages from EVM broadcast, got %d", len(solanaMessages))
	}

	t.Logf("Solana subscription ID: %d", solanaSubID)
	t.Logf("EVM subscription ID: %d", evmSubID)
	t.Log("Broadcast isolation verified successfully")
}
