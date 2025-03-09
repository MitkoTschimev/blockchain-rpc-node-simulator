package main

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionManager(t *testing.T) {
	sm := NewSubscriptionManager()
	conn := NewMockWSConn()

	// Test EVM subscription
	evmSubID, err := sm.Subscribe("ethereum", conn, "newHeads")
	if err != nil {
		t.Fatalf("Failed to create EVM subscription: %v", err)
	}

	// Test Solana subscription
	solanaSubID, err := sm.Subscribe("solana", conn, "slotNotification")
	if err != nil {
		t.Fatalf("Failed to create Solana subscription: %v", err)
	}

	// Verify subscriptions exist
	if len(sm.subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(sm.subscriptions))
	}

	// Test EVM notification format
	sm.BroadcastNewBlock("ethereum", 100)
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

	chain, ok := evmResult["chain"].(string)
	if !ok || chain != "ethereum" {
		t.Errorf("Expected chain ethereum, got %v", chain)
	}

	// Clear messages for Solana test
	conn.ClearMessages()

	// Test Solana notification format
	sm.BroadcastNewBlock("solana", 100)
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

	// Create multiple subscriptions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			var err error
			if i%2 == 0 {
				_, err = sm.Subscribe("ethereum", conn, "newHeads")
			} else {
				_, err = sm.Subscribe("solana", conn, "slotNotification")
			}
			if err != nil {
				t.Errorf("Failed to create subscription: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all subscriptions
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(sm.subscriptions) != 10 {
		t.Errorf("Expected 10 subscriptions, got %d", len(sm.subscriptions))
	}

	// Test concurrent broadcasts
	for i := 0; i < 10; i++ {
		go func(i int) {
			if i%2 == 0 {
				sm.BroadcastNewBlock("ethereum", uint64(i))
			} else {
				sm.BroadcastNewBlock("solana", uint64(i))
			}
			done <- true
		}(i)
	}

	// Wait for all broadcasts
	for i := 0; i < 10; i++ {
		<-done
	}

	messages := conn.GetMessages()
	if len(messages) != 50 { // 5 ethereum + 5 solana broadcasts * 5 subscriptions each
		t.Errorf("Expected 50 messages, got %d", len(messages))
	}
}
