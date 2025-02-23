package main

import (
	"encoding/json"
	"testing"
)

func TestEVMHandler(t *testing.T) {
	conn := NewMockWSConn()

	tests := []struct {
		name     string
		request  JSONRPCRequest
		validate func(t *testing.T, response []byte)
	}{
		{
			name: "eth_chainId",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_chainId",
				Params:  []interface{}{},
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
			name: "eth_blockNumber",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_blockNumber",
				Params:  []interface{}{},
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
			name: "eth_subscribe",
			request: JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_subscribe",
				Params:  []interface{}{"newHeads"},
				ID:      1,
			},
			validate: func(t *testing.T, response []byte) {
				var resp JSONRPCResponse
				if err := json.Unmarshal(response, &resp); err != nil {
					t.Errorf("Failed to parse response: %v", err)
					return
				}
				if resp.Result == "" {
					t.Error("Subscription ID should not be empty")
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

			response, err := handleEVMRequest(requestData, conn)
			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			tt.validate(t, response)
		})
	}
}

func TestSolanaHandler(t *testing.T) {
	conn := NewMockWSConn()

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
				if resp.Result == nil {
					t.Error("Slot number should not be nil")
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
				if version["solana-core"] == nil {
					t.Error("Version should include solana-core")
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
				if resp.Result == "" {
					t.Error("Subscription ID should not be empty")
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
