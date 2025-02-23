package main

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionManager(t *testing.T) {
	sm := NewSubscriptionManager()

	// Test EVM subscription
	evmConn := NewMockWSConn()
	evmSubID, err := sm.Subscribe("evm", evmConn, "newHeads")
	if err != nil {
		t.Errorf("Failed to create EVM subscription: %v", err)
	}
	if evmSubID == 0 {
		t.Error("Subscription ID should not be zero")
	}

	// Test Solana subscription
	solanaConn := NewMockWSConn()
	solanaSubID, err := sm.Subscribe("solana", solanaConn, "slotNotification")
	if err != nil {
		t.Error("Should allow multiple subscriptions from same connection")
	}
	if solanaSubID == 0 {
		t.Error("Subscription ID should not be zero")
	}

	// Test broadcast
	sm.BroadcastNewBlock(1000)

	// Verify EVM notification format
	evmMessages := evmConn.GetMessages()
	if len(evmMessages) != 1 {
		t.Errorf("Expected 1 EVM message, got %d", len(evmMessages))
	} else {
		var notification JSONRPCNotification
		if err := json.Unmarshal(evmMessages[0], &notification); err != nil {
			t.Errorf("Invalid EVM notification format: %v", err)
		} else {
			params, ok := notification.Params.(map[string]interface{})
			if !ok {
				t.Error("Expected params to be an object")
			} else {
				// Verify subscription ID is string
				subID, ok := params["subscription"].(string)
				if !ok {
					t.Error("EVM subscription ID should be a string")
				}
				if subID == "" {
					t.Error("EVM subscription ID should not be empty")
				}
			}
		}
	}

	// Verify Solana notification format
	solanaMessages := solanaConn.GetMessages()
	if len(solanaMessages) != 1 {
		t.Errorf("Expected 1 Solana message, got %d", len(solanaMessages))
	} else {
		var notification JSONRPCNotification
		if err := json.Unmarshal(solanaMessages[0], &notification); err != nil {
			t.Errorf("Invalid Solana notification format: %v", err)
		} else {
			params, ok := notification.Params.(map[string]interface{})
			if !ok {
				t.Error("Expected params to be an object")
			} else {
				// Verify subscription ID is number
				subID, ok := params["subscription"].(float64)
				if !ok {
					t.Error("Solana subscription ID should be a number")
				}
				if subID <= 0 {
					t.Error("Solana subscription ID should be positive")
				}

				// Verify slot notification format
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					t.Error("Expected result to be an object")
				} else {
					if _, ok := result["parent"].(float64); !ok {
						t.Error("Expected parent to be a number")
					}
					if _, ok := result["root"].(float64); !ok {
						t.Error("Expected root to be a number")
					}
					if _, ok := result["slot"].(float64); !ok {
						t.Error("Expected slot to be a number")
					}
				}
			}
		}
	}

	// Test unsubscribe
	if err := sm.Unsubscribe(evmSubID); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}

	// Test unsubscribe of non-existent subscription
	if err := sm.Unsubscribe(999); err == nil {
		t.Error("Should return error for non-existent subscription")
	}

	// Test drop all connections
	count := sm.DropAllConnections()
	if count != 1 { // Only Solana subscription should remain
		t.Errorf("Expected 1 connection to be dropped, got %d", count)
	}
	if !solanaConn.IsClosed() {
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
