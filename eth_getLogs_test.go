package main

import (
	"encoding/json"
	"sync/atomic"
	"testing"
)

func TestEthGetLogsBlockRangeValidation(t *testing.T) {
	// Create a test chain
	chain := &EVMChain{
		Name:         "test",
		ChainID:      "1",
		BlockNumber:  100, // Current block is 100
		LogsPerBlock: 5,
	}

	// Add chain to supported chains for testing
	supportedChains["test"] = chain
	chainIdToName["1"] = "test"

	tests := []struct {
		name           string
		fromBlock      string
		toBlock        string
		currentBlock   uint64
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "Valid range: fromBlock < toBlock < currentBlock",
			fromBlock:    "0x1",
			toBlock:      "0x5",
			currentBlock: 100,
			expectError:  false,
		},
		{
			name:         "Valid range: toBlock = latest",
			fromBlock:    "0x1",
			toBlock:      "latest",
			currentBlock: 100,
			expectError:  false,
		},
		{
			name:         "Valid range: toBlock = currentBlock",
			fromBlock:    "0x1",
			toBlock:      "0x64", // 0x64 = 100
			currentBlock: 100,
			expectError:  false,
		},
		{
			name:           "Invalid: toBlock > currentBlock",
			fromBlock:      "0x1",
			toBlock:        "0xFF", // 255 > 100
			currentBlock:   100,
			expectError:    true,
			expectedErrMsg: "invalid block range params",
		},
		{
			name:           "Invalid: fromBlock > toBlock",
			fromBlock:      "0x64", // 100
			toBlock:        "0x32", // 50
			currentBlock:   100,
			expectError:    true,
			expectedErrMsg: "invalid block range params",
		},
		{
			name:         "Valid: fromBlock = toBlock",
			fromBlock:    "0x32",
			toBlock:      "0x32",
			currentBlock: 100,
			expectError:  false,
		},
		{
			name:         "Valid: earliest to latest",
			fromBlock:    "earliest",
			toBlock:      "latest",
			currentBlock: 100,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the current block number
			atomic.StoreUint64(&chain.BlockNumber, tt.currentBlock)

			// Create request
			filter := map[string]interface{}{
				"fromBlock": tt.fromBlock,
				"toBlock":   tt.toBlock,
			}

			request := JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  "eth_getLogs",
				Params:  []interface{}{filter},
				ID:      1,
			}

			requestBytes, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Call handleEVMRequest
			response, err := handleEVMRequest(requestBytes, nil, "1")
			if err != nil {
				t.Fatalf("handleEVMRequest returned error: %v", err)
			}

			// Parse response
			var rpcResponse JSONRPCResponse
			if err := json.Unmarshal(response, &rpcResponse); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Check if error is expected
			if tt.expectError {
				if rpcResponse.Error == nil {
					t.Errorf("Expected error but got none")
				} else {
					if rpcResponse.Error.Code != -32000 {
						t.Errorf("Expected error code -32000, got %d", rpcResponse.Error.Code)
					}
					if rpcResponse.Error.Message != tt.expectedErrMsg {
						t.Errorf("Expected error message '%s', got '%s'", tt.expectedErrMsg, rpcResponse.Error.Message)
					}
				}
			} else {
				if rpcResponse.Error != nil {
					t.Errorf("Unexpected error: %v", rpcResponse.Error)
				}
				if rpcResponse.Result == nil {
					t.Errorf("Expected result, got nil")
				}
			}
		})
	}
}
