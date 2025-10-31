package main

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
)

// TestSafeAndFinalizedBlocks tests safe and finalized block functionality
func TestSafeAndFinalizedBlocks(t *testing.T) {
	// Create a test chain
	chain := &EVMChain{
		Name:                 "test",
		ChainID:              "1",
		BlockNumber:          100,
		SafeBlockNumber:      68, // 100 - 32
		FinalizedBlockNumber: 36, // 100 - 64
		LogsPerBlock:         5,
	}

	// Add chain to supported chains
	supportedChains["test"] = chain
	chainIdToName["1"] = "test"

	tests := []struct {
		name          string
		method        string
		blockParam    string
		expectedBlock uint64
	}{
		{
			name:          "eth_getBlockByNumber with 'latest'",
			method:        "eth_getBlockByNumber",
			blockParam:    "latest",
			expectedBlock: 100,
		},
		{
			name:          "eth_getBlockByNumber with 'safe'",
			method:        "eth_getBlockByNumber",
			blockParam:    "safe",
			expectedBlock: 68,
		},
		{
			name:          "eth_getBlockByNumber with 'finalized'",
			method:        "eth_getBlockByNumber",
			blockParam:    "finalized",
			expectedBlock: 36,
		},
		{
			name:          "eth_getBlockByNumber with 'earliest'",
			method:        "eth_getBlockByNumber",
			blockParam:    "earliest",
			expectedBlock: 0,
		},
		{
			name:          "eth_getBlockByNumber with hex value",
			method:        "eth_getBlockByNumber",
			blockParam:    "0x32", // 50 in decimal
			expectedBlock: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			request := JSONRPCRequest{
				JsonRPC: "2.0",
				Method:  tt.method,
				Params:  []interface{}{tt.blockParam},
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

			// Check for errors
			if rpcResponse.Error != nil {
				t.Fatalf("Unexpected error: %v", rpcResponse.Error)
			}

			// Verify result
			resultStr, ok := rpcResponse.Result.(string)
			if !ok {
				t.Fatalf("Expected result to be string, got %T", rpcResponse.Result)
			}

			// Parse hex result
			var resultBlock uint64
			if len(resultStr) > 2 && resultStr[:2] == "0x" {
				resultStr = resultStr[2:]
			}
			_, err = fmt.Sscanf(resultStr, "%x", &resultBlock)
			if err != nil {
				t.Fatalf("Failed to parse result as hex: %v", err)
			}

			if resultBlock != tt.expectedBlock {
				t.Errorf("Expected block %d, got %d", tt.expectedBlock, resultBlock)
			}
		})
	}
}

// TestEthGetLogsWithSafeAndFinalized tests eth_getLogs with safe and finalized blocks
func TestEthGetLogsWithSafeAndFinalized(t *testing.T) {
	// Create a test chain
	chain := &EVMChain{
		Name:                 "test",
		ChainID:              "1",
		BlockNumber:          100,
		SafeBlockNumber:      68,
		FinalizedBlockNumber: 36,
		LogsPerBlock:         5,
	}

	supportedChains["test"] = chain
	chainIdToName["1"] = "test"

	tests := []struct {
		name           string
		fromBlock      string
		toBlock        string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:        "Valid range: safe to latest",
			fromBlock:   "safe",
			toBlock:     "latest",
			expectError: false,
		},
		{
			name:        "Valid range: finalized to safe",
			fromBlock:   "finalized",
			toBlock:     "safe",
			expectError: false,
		},
		{
			name:        "Valid range: finalized to latest",
			fromBlock:   "finalized",
			toBlock:     "latest",
			expectError: false,
		},
		{
			name:        "Valid range: earliest to finalized",
			fromBlock:   "earliest",
			toBlock:     "finalized",
			expectError: false,
		},
		{
			name:        "Valid range: earliest to safe",
			fromBlock:   "earliest",
			toBlock:     "safe",
			expectError: false,
		},
		{
			name:           "Invalid: toBlock higher than latest",
			fromBlock:      "safe",
			toBlock:        "0xFF", // 255 > 100
			expectError:    true,
			expectedErrMsg: "invalid block range params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create filter
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
				} else if rpcResponse.Error.Message != tt.expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.expectedErrMsg, rpcResponse.Error.Message)
				}
			} else {
				if rpcResponse.Error != nil {
					t.Errorf("Unexpected error: %v", rpcResponse.Error)
				}
			}
		})
	}
}

// TestBlockIncrementUpdatesSafeAndFinalized tests that safe and finalized blocks update correctly
func TestBlockIncrementUpdatesSafeAndFinalized(t *testing.T) {
	chain := &EVMChain{
		Name:                 "test",
		ChainID:              "1",
		BlockNumber:          100,
		SafeBlockNumber:      0,
		FinalizedBlockNumber: 0,
		LogsPerBlock:         0,
	}

	// Simulate block increment
	newBlock := atomic.AddUint64(&chain.BlockNumber, 1) // Now at 101

	// Update safe block (latest - 32)
	if newBlock > 32 {
		atomic.StoreUint64(&chain.SafeBlockNumber, newBlock-32)
	}

	// Update finalized block (latest - 64)
	if newBlock > 64 {
		atomic.StoreUint64(&chain.FinalizedBlockNumber, newBlock-64)
	}

	// Verify
	latest := atomic.LoadUint64(&chain.BlockNumber)
	safe := atomic.LoadUint64(&chain.SafeBlockNumber)
	finalized := atomic.LoadUint64(&chain.FinalizedBlockNumber)

	if latest != 101 {
		t.Errorf("Expected latest block 101, got %d", latest)
	}
	if safe != 69 { // 101 - 32
		t.Errorf("Expected safe block 69, got %d", safe)
	}
	if finalized != 37 { // 101 - 64
		t.Errorf("Expected finalized block 37, got %d", finalized)
	}
}

// TestBlockIncrementWithLowBlockNumber tests safe/finalized with low block numbers
func TestBlockIncrementWithLowBlockNumber(t *testing.T) {
	chain := &EVMChain{
		Name:                 "test",
		ChainID:              "1",
		BlockNumber:          10,
		SafeBlockNumber:      0,
		FinalizedBlockNumber: 0,
		LogsPerBlock:         0,
	}

	// Simulate block increment
	newBlock := atomic.AddUint64(&chain.BlockNumber, 1) // Now at 11

	// Update safe block (latest - 32)
	if newBlock > 32 {
		atomic.StoreUint64(&chain.SafeBlockNumber, newBlock-32)
	} else {
		atomic.StoreUint64(&chain.SafeBlockNumber, 0)
	}

	// Update finalized block (latest - 64)
	if newBlock > 64 {
		atomic.StoreUint64(&chain.FinalizedBlockNumber, newBlock-64)
	} else {
		atomic.StoreUint64(&chain.FinalizedBlockNumber, 0)
	}

	// Verify - both should be 0 since block is too low
	safe := atomic.LoadUint64(&chain.SafeBlockNumber)
	finalized := atomic.LoadUint64(&chain.FinalizedBlockNumber)

	if safe != 0 {
		t.Errorf("Expected safe block 0, got %d", safe)
	}
	if finalized != 0 {
		t.Errorf("Expected finalized block 0, got %d", finalized)
	}
}
