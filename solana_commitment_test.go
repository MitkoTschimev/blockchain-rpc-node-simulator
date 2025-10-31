package main

import (
	"encoding/json"
	"sync/atomic"
	"testing"
)

func TestSolanaGetSlotWithCommitment(t *testing.T) {
	// Set current slot to 100
	atomic.StoreUint64(&solanaNode.SlotNumber, 100)

	tests := []struct {
		name           string
		commitment     string
		expectedSlot   uint64
		withParams     bool
	}{
		{
			name:         "No params - defaults to processed (latest)",
			commitment:   "",
			expectedSlot: 100,
			withParams:   false,
		},
		{
			name:         "Processed commitment - latest slot",
			commitment:   "processed",
			expectedSlot: 100,
			withParams:   true,
		},
		{
			name:         "Confirmed commitment - slot - 1",
			commitment:   "confirmed",
			expectedSlot: 99,
			withParams:   true,
		},
		{
			name:         "Finalized commitment - slot - 3",
			commitment:   "finalized",
			expectedSlot: 97,
			withParams:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request JSONRPCRequest
			request.JsonRPC = "2.0"
			request.Method = "getSlot"
			request.ID = 1

			if tt.withParams {
				request.Params = []interface{}{
					map[string]interface{}{
						"commitment": tt.commitment,
					},
				}
			}

			requestBytes, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			response, err := handleSolanaRequest(requestBytes, nil)
			if err != nil {
				t.Fatalf("handleSolanaRequest failed: %v", err)
			}

			var rpcResponse JSONRPCResponse
			if err := json.Unmarshal(response, &rpcResponse); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if rpcResponse.Error != nil {
				t.Fatalf("Unexpected error: %v", rpcResponse.Error)
			}

			slot, ok := rpcResponse.Result.(float64)
			if !ok {
				t.Fatalf("Expected result to be a number, got %T", rpcResponse.Result)
			}

			if uint64(slot) != tt.expectedSlot {
				t.Errorf("Expected slot %d, got %d", tt.expectedSlot, uint64(slot))
			}
		})
	}
}

func TestSolanaGetSlotWithLowSlotNumber(t *testing.T) {
	// Set current slot to 2 (less than finalized offset of 3)
	atomic.StoreUint64(&solanaNode.SlotNumber, 2)

	tests := []struct {
		name         string
		commitment   string
		expectedSlot uint64
	}{
		{
			name:         "Finalized with low slot returns 0",
			commitment:   "finalized",
			expectedSlot: 0,
		},
		{
			name:         "Confirmed with slot 2 returns 1",
			commitment:   "confirmed",
			expectedSlot: 1,
		},
		{
			name:         "Processed with slot 2 returns 2",
			commitment:   "processed",
			expectedSlot: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "getSlot",
				Params: []interface{}{
					map[string]interface{}{
						"commitment": tt.commitment,
					},
				},
				ID: 1,
			}

			requestBytes, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			response, err := handleSolanaRequest(requestBytes, nil)
			if err != nil {
				t.Fatalf("handleSolanaRequest failed: %v", err)
			}

			var rpcResponse JSONRPCResponse
			if err := json.Unmarshal(response, &rpcResponse); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			slot, ok := rpcResponse.Result.(float64)
			if !ok {
				t.Fatalf("Expected result to be a number, got %T", rpcResponse.Result)
			}

			if uint64(slot) != tt.expectedSlot {
				t.Errorf("Expected slot %d, got %d", tt.expectedSlot, uint64(slot))
			}
		})
	}
}
