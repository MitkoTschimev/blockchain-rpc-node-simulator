package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEVMHandler(t *testing.T) {
	conn := NewMockWSConn()

	// First create a subscription that we can unsubscribe from later
	subRequest := JSONRPCRequest{
		JsonRPC: "2.0",
		Method:  "eth_subscribe",
		Params:  []interface{}{"ethereum", "newHeads"},
		ID:      1,
	}
	subRequestData, _ := json.Marshal(subRequest)
	subResponse, _ := handleEVMRequest(subRequestData, conn, "1")
	var subResp JSONRPCResponse
	json.Unmarshal(subResponse, &subResp)
	subscriptionID := subResp.Result.(string)

	tests := []struct {
		name     string
		request  JSONRPCRequest
		validate func(t *testing.T, response []byte)
	}{
		{
			name: "eth_chainId for ethereum",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_chainId",
				Params:  []interface{}{"ethereum"},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				if resp.Result != "0x1" {
					t.Errorf("Expected chainId 0x1, got %v", resp.Result)
				}
			},
		},
		{
			name: "eth_chainId for optimism",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_chainId",
				Params:  []interface{}{"optimism"},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				if resp.Result != "0xa" {
					t.Errorf("Expected chainId 0xa, got %v", resp.Result)
				}
			},
		},
		{
			name: "eth_blockNumber for ethereum",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_blockNumber",
				Params:  []interface{}{"ethereum"},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				if resp.Result == "" {
					t.Error("Block number should not be empty")
				}
			},
		},
		{
			name: "eth_subscribe with chain",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_subscribe",
				Params:  []interface{}{"optimism", "newHeads"},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				// Verify subscription ID is a string
				subID, ok := resp.Result.(string)
				if !ok {
					t.Error("Expected subscription ID to be a string")
					return
				}
				if subID == "" {
					t.Error("Subscription ID should not be empty")
				}
			},
		},
		{
			name: "eth_unsubscribe with string ID",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_unsubscribe",
				Params:  []interface{}{subscriptionID},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				if resp.Result != true {
					t.Error("Expected unsubscribe to return true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestData, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			response, err := handleEVMRequest(requestData, conn, "1")
			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			tt.validate(t, response)
		})
	}
}

func TestSolanaHandler(t *testing.T) {
	conn := NewMockWSConn()

	// First create a subscription that we can unsubscribe from later
	subRequest := JSONRPCRequest{
		JsonRPC: "2.0",
		Method:  "slotSubscribe",
		Params:  []interface{}{},
		ID:      1,
	}
	subRequestData, _ := json.Marshal(subRequest)
	subResponse, _ := handleSolanaRequest(subRequestData, conn)
	var subResp JSONRPCResponse
	json.Unmarshal(subResponse, &subResp)
	subscriptionID := uint64(subResp.Result.(float64))

	tests := []struct {
		name     string
		request  JSONRPCRequest
		validate func(t *testing.T, response []byte)
	}{
		{
			name: "getSlot",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "getSlot",
				Params:  []interface{}{},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				slot, ok := resp.Result.(float64)
				if !ok {
					t.Error("Expected slot to be a number")
					return
				}
				if slot < 1 {
					t.Error("Slot number should be at least 1")
				}
			},
		},
		{
			name: "getVersion",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "getVersion",
				Params:  []interface{}{},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				version, ok := resp.Result.(map[string]interface{})
				if !ok {
					t.Error("Expected version to be an object")
					return
				}
				if version["solana-core"] != "1.14.10" {
					t.Error("Version should be 1.14.10")
				}
			},
		},
		{
			name: "slotSubscribe",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "slotSubscribe",
				Params:  []interface{}{},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				// Verify subscription ID is a number
				subID, ok := resp.Result.(float64)
				if !ok {
					t.Error("Expected subscription ID to be a number")
					return
				}
				if subID <= 0 {
					t.Error("Subscription ID should be positive")
				}
			},
		},
		{
			name: "slotUnsubscribe with numeric ID",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "slotUnsubscribe",
				Params:  []interface{}{float64(subscriptionID)},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				if resp.Result != true {
					t.Error("Expected unsubscribe to return true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestData, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			response, err := handleSolanaRequest(requestData, conn)
			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			tt.validate(t, response)
		})
	}
}

func TestBlockIntervals(t *testing.T) {
	// Test that block intervals are respected
	for chain, expectedInterval := range map[string]time.Duration{
		"ethereum":  12 * time.Second,
		"optimism":  2 * time.Second,
		"arbitrum":  250 * time.Millisecond,
		"avalanche": 2 * time.Second,
		"base":      2 * time.Second,
		"binance":   3 * time.Second,
	} {
		c := supportedChains[chain]
		if c.BlockInterval != expectedInterval {
			t.Errorf("Chain %s: expected interval %v, got %v", chain, expectedInterval, c.BlockInterval)
		}
	}

	// Test Solana interval
	if solanaNode.SlotInterval != 400*time.Millisecond {
		t.Errorf("Solana: expected interval 400ms, got %v", solanaNode.SlotInterval)
	}
}
