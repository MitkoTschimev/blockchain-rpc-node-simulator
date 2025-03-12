package main

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestSubscriptionManager(t *testing.T) {
	sm := NewSubscriptionManager()
	conn := NewMockWSConn()

	// Test EVM subscription
	evmSubID, err := sm.Subscribe("1", conn, "newHeads")
	if err != nil {
		t.Fatalf("Failed to create EVM subscription: %v", err)
	}

	// Test Solana subscription
	solanaSubID, err := sm.Subscribe("501", conn, "slotNotification")
	if err != nil {
		t.Fatalf("Failed to create Solana subscription: %v", err)
	}

	// Verify subscriptions exist
	if len(sm.subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(sm.subscriptions))
	}

	// Test EVM notification format
	sm.BroadcastNewBlock("1", 100)
	messages := conn.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 EVM message, got %d", len(messages))
	}

	var evmNotification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &evmNotification); err != nil {
		t.Fatalf("Failed to parse EVM notification: %v", err)
	}

	// Verify EVM notification format
	if evmNotification.Method != "eth_subscription" {
		t.Errorf("Expected method eth_subscription, got %s", evmNotification.Method)
	}

	evmParams, ok := evmNotification.Params.(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse EVM notification params")
	}

	evmSubIDStr, ok := evmParams["subscription"].(string)
	if !ok || evmSubIDStr != "1" {
		t.Errorf("Expected subscription ID '1', got %v", evmParams["subscription"])
	}

	evmResult, ok := evmParams["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse EVM result")
	}

	blockNumber, ok := evmResult["number"].(string)
	if !ok || blockNumber != "0x64" { // 100 in hex
		t.Errorf("Expected block number 0x64, got %v", blockNumber)
	}

	// Clear messages for Solana test
	conn.ClearMessages()

	// Test Solana notification format
	sm.BroadcastNewBlock("501", 100)
	messages = conn.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 Solana message, got %d", len(messages))
	}

	var solanaNotification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &solanaNotification); err != nil {
		t.Fatalf("Failed to parse Solana notification: %v", err)
	}

	// Verify Solana notification format
	if solanaNotification.Method != "slotNotification" {
		t.Errorf("Expected method slotNotification, got %s", solanaNotification.Method)
	}

	solanaParams, ok := solanaNotification.Params.(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse Solana notification params")
	}

	solanaSubIDFloat, ok := solanaParams["subscription"].(float64)
	if !ok || uint64(solanaSubIDFloat) != solanaSubID {
		t.Errorf("Expected subscription ID %d, got %v", solanaSubID, solanaParams["subscription"])
	}

	solanaResult, ok := solanaParams["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse Solana result")
	}

	// Verify Solana slot notification fields
	parent, ok := solanaResult["parent"].(float64)
	if !ok || uint64(parent) != 99 {
		t.Errorf("Expected parent slot 99, got %v", parent)
	}

	root, ok := solanaResult["root"].(float64)
	if !ok || uint64(root) != 97 { // 100 - 3
		t.Errorf("Expected root slot 97, got %v", root)
	}

	slot, ok := solanaResult["slot"].(float64)
	if !ok || uint64(slot) != 100 {
		t.Errorf("Expected slot 100, got %v", slot)
	}

	// Test unsubscribe
	if err := sm.Unsubscribe(evmSubID); err != nil {
		t.Errorf("Failed to unsubscribe from EVM: %v", err)
	}
	if err := sm.Unsubscribe(solanaSubID); err != nil {
		t.Errorf("Failed to unsubscribe from Solana: %v", err)
	}

	if len(sm.subscriptions) != 0 {
		t.Errorf("Expected 0 subscriptions after unsubscribe, got %d", len(sm.subscriptions))
	}
}

func TestSubscriptionManagerConcurrent(t *testing.T) {
	sm := NewSubscriptionManager()
	conn := NewMockWSConn()

	// Create subscriptions with guaranteed even distribution
	var wg sync.WaitGroup
	evmSubs := make(chan struct{}, 5) // Limit to 5 EVM subscriptions
	solSubs := make(chan struct{}, 5) // Limit to 5 Solana subscriptions

	// Create 5 EVM subscriptions
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := sm.Subscribe("1", conn, "newHeads")
			if err != nil {
				t.Errorf("Failed to create EVM subscription: %v", err)
				return
			}
			evmSubs <- struct{}{}
		}()
	}

	// Create 5 Solana subscriptions
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := sm.Subscribe("501", conn, "slotNotification")
			if err != nil {
				t.Errorf("Failed to create Solana subscription: %v", err)
				return
			}
			solSubs <- struct{}{}
		}()
	}

	wg.Wait()
	close(evmSubs)
	close(solSubs)

	// Verify we have exactly 5 subscriptions of each type
	if len(evmSubs) != 5 {
		t.Errorf("Expected 5 EVM subscriptions, got %d", len(evmSubs))
	}
	if len(solSubs) != 5 {
		t.Errorf("Expected 5 Solana subscriptions, got %d", len(solSubs))
	}

	if len(sm.subscriptions) != 10 {
		t.Errorf("Expected 10 total subscriptions, got %d", len(sm.subscriptions))
	}

	// Use channels to synchronize broadcasts
	evmBroadcasts := make(chan struct{}, 5)
	solBroadcasts := make(chan struct{}, 5)

	// Test concurrent broadcasts
	for i := 0; i < 5; i++ {
		wg.Add(2) // One for each chain type
		go func(i int) {
			defer wg.Done()
			sm.BroadcastNewBlock("1", uint64(i))
			evmBroadcasts <- struct{}{}
		}(i)
		go func(i int) {
			defer wg.Done()
			sm.BroadcastNewBlock("501", uint64(i))
			solBroadcasts <- struct{}{}
		}(i)
	}

	wg.Wait()
	close(evmBroadcasts)
	close(solBroadcasts)

	// Verify all broadcasts were sent
	if len(evmBroadcasts) != 5 {
		t.Errorf("Expected 5 EVM broadcasts, got %d", len(evmBroadcasts))
	}
	if len(solBroadcasts) != 5 {
		t.Errorf("Expected 5 Solana broadcasts, got %d", len(solBroadcasts))
	}

	// Give a small amount of time for all messages to be processed
	time.Sleep(50 * time.Millisecond)

	messages := conn.GetMessages()
	// Each broadcast should go to 5 subscriptions of its chain type
	// 5 broadcasts * 5 subscriptions * 2 chain types = 50 messages
	expectedMessages := 50
	if len(messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(messages))

		// Count message types for debugging
		evmCount := 0
		solCount := 0
		for _, msg := range messages {
			var notification JSONRPCNotification
			if err := json.Unmarshal(msg, &notification); err != nil {
				t.Errorf("Failed to parse message: %v", err)
				continue
			}
			if notification.Method == "eth_subscription" {
				evmCount++
			} else if notification.Method == "slotNotification" {
				solCount++
			}
		}
		t.Logf("Message breakdown - EVM: %d, Solana: %d", evmCount, solCount)
	}
}
