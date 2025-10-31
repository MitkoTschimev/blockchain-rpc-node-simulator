package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	// Test Log subscription
	logSubID, err := sm.Subscribe("1", conn, "logs")
	if err != nil {
		t.Fatalf("Failed to create Log subscription: %v", err)
	}

	// Verify subscriptions exist
	if len(sm.subscriptions) != 3 {
		t.Errorf("Expected 3 subscriptions, got %d", len(sm.subscriptions))
	}

	// Test EVM notification format
	sm.BroadcastNewBlock("1", 100)
	messages := conn.GetMessages()
	if len(messages) != 2 { // Both newHeads and logs subscriptions receive block notifications
		t.Fatalf("Expected 2 EVM messages, got %d", len(messages))
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
	if !ok || evmSubIDStr != "0x1" {
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

	// Clear messages for Log test
	conn.ClearMessages()

	// Test Log notification format
	logEvent := LogEvent{
		Address:     "0x" + hex.EncodeToString(make([]byte, 20)),
		Topics:      []string{"0x" + hex.EncodeToString(make([]byte, 32))},
		Data:        "0x" + hex.EncodeToString(make([]byte, 32)),
		BlockNumber: 100,
		TxHash:      "0x" + hex.EncodeToString(make([]byte, 32)),
		TxIndex:     0,
		BlockHash:   "0x" + hex.EncodeToString(make([]byte, 32)),
		LogIndex:    0,
		Removed:     false,
	}
	sm.BroadcastNewLog("1", logEvent)
	messages = conn.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 Log message, got %d", len(messages))
	}

	var logNotification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &logNotification); err != nil {
		t.Fatalf("Failed to parse Log notification: %v", err)
	}

	// Verify Log notification format
	if logNotification.Method != "eth_subscription" {
		t.Errorf("Expected method eth_subscription, got %s", logNotification.Method)
	}

	logParams, ok := logNotification.Params.(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse Log notification params")
	}

	logSubIDStr, ok := logParams["subscription"].(string)
	if !ok || logSubIDStr != fmt.Sprintf("0x%x", logSubID) {
		t.Errorf("Expected subscription ID %x, got %v", logSubID, logParams["subscription"])
	}

	logResult, ok := logParams["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse Log result")
	}

	// Verify Log event fields
	address, ok := logResult["address"].(string)
	if !ok || address != logEvent.Address {
		t.Errorf("Expected address %s, got %v", logEvent.Address, address)
	}

	topics, ok := logResult["topics"].([]interface{})
	if !ok || len(topics) != len(logEvent.Topics) {
		t.Errorf("Expected %d topics, got %v", len(logEvent.Topics), topics)
	}

	data, ok := logResult["data"].(string)
	if !ok || data != logEvent.Data {
		t.Errorf("Expected data %s, got %v", logEvent.Data, data)
	}

	blockNum, ok := logResult["blockNumber"].(float64)
	if !ok || uint64(blockNum) != logEvent.BlockNumber {
		t.Errorf("Expected block number %d, got %v", logEvent.BlockNumber, blockNum)
	}

	// Test unsubscribe
	if err := sm.Unsubscribe(evmSubID); err != nil {
		t.Errorf("Failed to unsubscribe from EVM: %v", err)
	}
	if err := sm.Unsubscribe(solanaSubID); err != nil {
		t.Errorf("Failed to unsubscribe from Solana: %v", err)
	}
	if err := sm.Unsubscribe(logSubID); err != nil {
		t.Errorf("Failed to unsubscribe from Log: %v", err)
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

func TestSubscriptionManagerWithTransactions(t *testing.T) {
	sm := NewSubscriptionManager()
	conn := NewMockWSConn()

	// Test EVM subscription with transactions
	evmWithTxSubID, err := sm.Subscribe("1", conn, "newHeadsWithTx")
	if err != nil {
		t.Fatalf("Failed to create EVM subscription with transactions: %v", err)
	}

	// Test Solana subscription
	solanaSubID, err := sm.Subscribe("501", conn, "slotNotification")
	if err != nil {
		t.Fatalf("Failed to create Solana subscription: %v", err)
	}

	// Test Log subscription
	logSubID, err := sm.Subscribe("1", conn, "logs")
	if err != nil {
		t.Fatalf("Failed to create Log subscription: %v", err)
	}

	// Clear messages for EVM with transactions test
	conn.ClearMessages()

	// Test EVM notification format with transactions
	sm.BroadcastNewBlock("1", 101)
	messages := conn.GetMessages()
	if len(messages) != 2 { // Both newHeads and logs subscriptions receive block notifications
		t.Fatalf("Expected 2 EVM messages, got %d", len(messages))
	}

	var evmWithTxNotification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &evmWithTxNotification); err != nil {
		t.Fatalf("Failed to parse EVM notification with transactions: %v", err)
	}

	// Verify EVM notification format with transactions
	if evmWithTxNotification.Method != "eth_subscription" {
		t.Errorf("Expected method eth_subscription, got %s", evmWithTxNotification.Method)
	}

	evmWithTxParams, ok := evmWithTxNotification.Params.(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse EVM notification params")
	}

	evmWithTxSubIDStr, ok := evmWithTxParams["subscription"].(string)
	if !ok || evmWithTxSubIDStr != fmt.Sprintf("0x%x", evmWithTxSubID) {
		t.Errorf("Expected subscription ID %x, got %v", evmWithTxSubID, evmWithTxParams["subscription"])
	}

	evmWithTxResult, ok := evmWithTxParams["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse EVM result")
	}

	// Verify block fields
	blockNumber, ok := evmWithTxResult["number"].(string)
	if !ok || blockNumber != "0x65" { // 101 in hex
		t.Errorf("Expected block number 0x65, got %v", blockNumber)
	}

	// Verify transaction fields
	transactions, ok := evmWithTxResult["transactions"].([]interface{})
	if !ok || len(transactions) == 0 {
		t.Error("Expected non-empty transactions array")
	}

	// Verify first transaction fields
	firstTx, ok := transactions[0].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse first transaction")
	}

	// Check required transaction fields
	requiredTxFields := []string{"hash", "nonce", "blockHash", "blockNumber", "transactionIndex", "from", "to", "value", "gas", "gasPrice", "input", "v", "r", "s"}
	for _, field := range requiredTxFields {
		if _, ok := firstTx[field]; !ok {
			t.Errorf("Missing required transaction field: %s", field)
		}
	}

	// Test unsubscribe for EVM with transactions
	if err := sm.Unsubscribe(evmWithTxSubID); err != nil {
		t.Errorf("Failed to unsubscribe from EVM with transactions: %v", err)
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

	// Clear messages for Log test
	conn.ClearMessages()

	// Test Log notification format
	logEvent := LogEvent{
		Address:     "0x" + hex.EncodeToString(make([]byte, 20)),
		Topics:      []string{"0x" + hex.EncodeToString(make([]byte, 32))},
		Data:        "0x" + hex.EncodeToString(make([]byte, 32)),
		BlockNumber: 100,
		TxHash:      "0x" + hex.EncodeToString(make([]byte, 32)),
		TxIndex:     0,
		BlockHash:   "0x" + hex.EncodeToString(make([]byte, 32)),
		LogIndex:    0,
		Removed:     false,
	}
	sm.BroadcastNewLog("1", logEvent)
	messages = conn.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 Log message, got %d", len(messages))
	}

	var logNotification JSONRPCNotification
	if err := json.Unmarshal(messages[0], &logNotification); err != nil {
		t.Fatalf("Failed to parse Log notification: %v", err)
	}

	// Verify Log notification format
	if logNotification.Method != "eth_subscription" {
		t.Errorf("Expected method eth_subscription, got %s", logNotification.Method)
	}

	logParams, ok := logNotification.Params.(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse Log notification params")
	}

	logSubIDStr, ok := logParams["subscription"].(string)
	if !ok || logSubIDStr != fmt.Sprintf("0x%x", logSubID) {
		t.Errorf("Expected subscription ID %x, got %v", logSubID, logParams["subscription"])
	}

	logResult, ok := logParams["result"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to parse Log result")
	}

	// Verify Log event fields
	address, ok := logResult["address"].(string)
	if !ok || address != logEvent.Address {
		t.Errorf("Expected address %s, got %v", logEvent.Address, address)
	}

	topics, ok := logResult["topics"].([]interface{})
	if !ok || len(topics) != len(logEvent.Topics) {
		t.Errorf("Expected %d topics, got %v", len(logEvent.Topics), topics)
	}

	data, ok := logResult["data"].(string)
	if !ok || data != logEvent.Data {
		t.Errorf("Expected data %s, got %v", logEvent.Data, data)
	}

	blockNum, ok := logResult["blockNumber"].(float64)
	if !ok || uint64(blockNum) != logEvent.BlockNumber {
		t.Errorf("Expected block number %d, got %v", logEvent.BlockNumber, blockNum)
	}
}
