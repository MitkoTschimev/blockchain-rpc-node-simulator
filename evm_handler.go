package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

func init() {
	// Initialize block numbers for each chain
	for _, chain := range supportedChains {
		chain.BlockNumber = 1
		chain.BlockIncrement = 0
		// Set default error probability to 0
		chain.ErrorProbability = 0
	}
	// Initialize Solana slot number
	solanaNode.SlotNumber = 1
	solanaNode.SlotIncrement = 0
}

// generateBlockHash creates a deterministic hash based on block number and chain ID
func generateBlockHash(blockNumber uint64, chainID string, seed string) string {
	// Create a unique input combining block number, chain ID, and seed
	input := fmt.Sprintf("%s-%d-%s", chainID, blockNumber, seed)
	hash := sha256.Sum256([]byte(input))
	return "0x" + hex.EncodeToString(hash[:])
}

func handleEVMRequest(message []byte, conn WSConn, chainId string) ([]byte, error) {
	// Get chain configuration
	chainName, exists := chainIdToName[chainId]
	if !exists {
		return createErrorResponse(-32602, fmt.Sprintf("Unsupported chain ID: %s", chainId), nil, nil)
	}

	chain, ok := supportedChains[chainName]
	if !ok {
		return createErrorResponse(-32602, fmt.Sprintf("Unsupported chain: %s", chainName), nil, nil)
	}

	// Simulate network latency if configured
	if chain.Latency > 0 {
		time.Sleep(chain.Latency)
	}

	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		log.Printf("Error unmarshalling message: %s", err)
		log.Printf("Message: %s", string(message))
		return createErrorResponse(-32700, "Parse error", nil, nil)
	}

	// Only log non-health check messages
	if request.Method != "getHealth" {
		log.Printf("Incoming EVM message: %s", string(message))
	}

	// Validate JSON-RPC version
	if request.JsonRPC != "2.0" {
		return createErrorResponse(-32600, "Invalid Request", nil, request.ID)
	}

	// Legacy error probability support (deprecated but maintained for backwards compatibility)
	if chain.ErrorProbability > 0 && rand.Float64() < chain.ErrorProbability {
		return createErrorResponse(-32000, "header not found", nil, request.ID)
	}

	// New configurable error simulation
	if errorConfig := ShouldSimulateError(chain.ErrorConfigs, request.Method); errorConfig != nil {
		var data interface{}
		if errorConfig.Data != "" {
			data = errorConfig.Data
		}
		return createErrorResponse(errorConfig.Code, errorConfig.Message, data, request.ID)
	}

	var result interface{}
	var err error

	switch request.Method {
	case "eth_chainId":
		result = chain.ChainID
	case "eth_blockNumber":
		result = fmt.Sprintf("0x%x", atomic.LoadUint64(&chain.BlockNumber))
	case "eth_getBalance":
		result = "0x1234567890"
	case "eth_call":
		result = "0x1234567890"
	case "getHealth":
		result = "ok"
	case "eth_accounts":
		result = []string{}
	case "net_listening":
		result = true
	case "eth_getBlockByNumber":
		// Parse block parameter if provided
		if len(request.Params) > 0 {
			blockParam, ok := request.Params[0].(string)
			if !ok {
				return createErrorResponse(-32602, "Invalid block parameter", nil, request.ID)
			}

			var blockNumber uint64
			switch blockParam {
			case "latest", "pending":
				blockNumber = atomic.LoadUint64(&chain.BlockNumber)
			case "safe":
				blockNumber = atomic.LoadUint64(&chain.SafeBlockNumber)
			case "finalized":
				blockNumber = atomic.LoadUint64(&chain.FinalizedBlockNumber)
			case "earliest":
				blockNumber = 0
			default:
				// Parse hex block number
				if len(blockParam) > 2 && blockParam[:2] == "0x" {
					blockParam = blockParam[2:]
				}
				parsedBlock, err := strconv.ParseUint(blockParam, 16, 64)
				if err != nil {
					return createErrorResponse(-32602, "Invalid block number", nil, request.ID)
				}
				blockNumber = parsedBlock
			}

			// Generate unique hashes for this block
			blockHash := generateBlockHash(blockNumber, chainId, "block")
			var parentHash string
			if blockNumber > 0 {
				parentHash = generateBlockHash(blockNumber-1, chainId, "block")
			} else {
				parentHash = "0x" + hex.EncodeToString(make([]byte, 32))
			}

			// Return a full block object
			result = map[string]interface{}{
				"number":          fmt.Sprintf("0x%x", blockNumber),
				"hash":            blockHash,
				"parentHash":      parentHash,
				"timestamp":       fmt.Sprintf("0x%x", time.Now().Unix()),
				"gasLimit":        "0x" + hex.EncodeToString(make([]byte, 32)),
				"gasUsed":         "0x" + hex.EncodeToString(make([]byte, 32)),
				"miner":           "0x" + hex.EncodeToString(make([]byte, 20)),
				"difficulty":      "0x" + hex.EncodeToString(make([]byte, 32)),
				"totalDifficulty": "0x" + hex.EncodeToString(make([]byte, 32)),
				"size":            "0x" + hex.EncodeToString(make([]byte, 32)),
				"nonce":           "0x" + hex.EncodeToString(make([]byte, 8)),
				"extraData":       "0x",
				"baseFeePerGas":   "0x" + hex.EncodeToString(make([]byte, 32)),
				"uncles":          []string{},
				"transactions":    []interface{}{},
			}
		} else {
			blockNumber := atomic.LoadUint64(&chain.BlockNumber)

			// Generate unique hashes for this block
			blockHash := generateBlockHash(blockNumber, chainId, "block")
			var parentHash string
			if blockNumber > 0 {
				parentHash = generateBlockHash(blockNumber-1, chainId, "block")
			} else {
				parentHash = "0x" + hex.EncodeToString(make([]byte, 32))
			}

			result = map[string]interface{}{
				"number":          fmt.Sprintf("0x%x", blockNumber),
				"hash":            blockHash,
				"parentHash":      parentHash,
				"timestamp":       fmt.Sprintf("0x%x", time.Now().Unix()),
				"gasLimit":        "0x" + hex.EncodeToString(make([]byte, 32)),
				"gasUsed":         "0x" + hex.EncodeToString(make([]byte, 32)),
				"miner":           "0x" + hex.EncodeToString(make([]byte, 20)),
				"difficulty":      "0x" + hex.EncodeToString(make([]byte, 32)),
				"totalDifficulty": "0x" + hex.EncodeToString(make([]byte, 32)),
				"size":            "0x" + hex.EncodeToString(make([]byte, 32)),
				"nonce":           "0x" + hex.EncodeToString(make([]byte, 8)),
				"extraData":       "0x",
				"baseFeePerGas":   "0x" + hex.EncodeToString(make([]byte, 32)),
				"uncles":          []string{},
				"transactions":    []interface{}{},
			}
		}
	case "eth_subscribe":
		if len(request.Params) < 1 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}
		subscriptionType, ok := request.Params[0].(string)
		if !ok {
			return createErrorResponse(-32602, "Invalid subscription type", nil, request.ID)
		}

		var subType string
		switch subscriptionType {
		case "newHeads":
			subType = "newHeads"
			// Validate includeTransactions parameter if provided
			if len(request.Params) > 1 {
				options, ok := request.Params[1].(map[string]interface{})
				if !ok {
					return createErrorResponse(-32602, "Invalid subscription options", nil, request.ID)
				}
				includeTx, ok := options["includeTransactions"].(bool)
				if ok && includeTx {
					// Store the preference in the subscription
					subType = "newHeadsWithTx"
				}
			}
		case "logs":
			subType = "logs"
			// Validate log filter parameters if provided
			if len(request.Params) > 1 {
				_, ok = request.Params[1].(map[string]interface{})
				if !ok {
					return createErrorResponse(-32602, "Invalid log filter parameters", nil, request.ID)
				}
			}
		default:
			return createErrorResponse(-32601, fmt.Sprintf("Unsupported subscription type: %s", subscriptionType), nil, request.ID)
		}

		subID, err := subManager.Subscribe(chainId, conn, subType)
		if err != nil {
			return createErrorResponse(-32603, err.Error(), nil, request.ID)
		}

		result = fmt.Sprintf("0x%x", subID) // Return subscription ID as hex string for EVM

	case "eth_unsubscribe":
		if len(request.Params) < 1 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}

		// Handle both decimal and hex string subscription IDs
		var subscriptionID uint64
		switch v := request.Params[0].(type) {
		case string:
			// Try parsing as decimal first
			subscriptionID, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				// If decimal parsing fails, try hex
				if len(v) > 2 && v[:2] == "0x" {
					v = v[2:]
				}
				subscriptionID, err = strconv.ParseUint(v, 16, 64)
				if err != nil {
					return createErrorResponse(-32602, "Invalid subscription ID", nil, request.ID)
				}
			}
		case float64:
			subscriptionID = uint64(v)
		default:
			return createErrorResponse(-32602, "Invalid subscription ID type", nil, request.ID)
		}

		err := subManager.Unsubscribe(subscriptionID)
		if err != nil {
			return createErrorResponse(-32603, err.Error(), nil, request.ID)
		}
		result = true

	case "eth_getLogs":
		if len(request.Params) < 1 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}

		// Parse filter object
		filterObj, ok := request.Params[0].(map[string]interface{})
		if !ok {
			return createErrorResponse(-32602, "Invalid filter object", nil, request.ID)
		}

		// Get current block number
		currentBlock := atomic.LoadUint64(&chain.BlockNumber)

		// Parse fromBlock and toBlock
		var fromBlock, toBlock uint64

		// Parse fromBlock
		if fromBlockRaw, exists := filterObj["fromBlock"]; exists {
			fromBlockStr, ok := fromBlockRaw.(string)
			if !ok {
				return createErrorResponse(-32602, "Invalid fromBlock parameter", nil, request.ID)
			}
			switch fromBlockStr {
			case "latest", "pending":
				fromBlock = currentBlock
			case "safe":
				fromBlock = atomic.LoadUint64(&chain.SafeBlockNumber)
			case "finalized":
				fromBlock = atomic.LoadUint64(&chain.FinalizedBlockNumber)
			case "earliest":
				fromBlock = 0
			default:
				// Parse hex block number
				if len(fromBlockStr) > 2 && fromBlockStr[:2] == "0x" {
					fromBlockStr = fromBlockStr[2:]
				}
				parsedBlock, err := strconv.ParseUint(fromBlockStr, 16, 64)
				if err != nil {
					return createErrorResponse(-32602, "Invalid fromBlock hex value", nil, request.ID)
				}
				fromBlock = parsedBlock
			}
		} else {
			fromBlock = 0 // Default to earliest
		}

		// Parse toBlock
		if toBlockRaw, exists := filterObj["toBlock"]; exists {
			toBlockStr, ok := toBlockRaw.(string)
			if !ok {
				return createErrorResponse(-32602, "Invalid toBlock parameter", nil, request.ID)
			}
			switch toBlockStr {
			case "latest", "pending":
				toBlock = currentBlock
			case "safe":
				toBlock = atomic.LoadUint64(&chain.SafeBlockNumber)
			case "finalized":
				toBlock = atomic.LoadUint64(&chain.FinalizedBlockNumber)
			case "earliest":
				toBlock = 0
			default:
				// Parse hex block number
				if len(toBlockStr) > 2 && toBlockStr[:2] == "0x" {
					toBlockStr = toBlockStr[2:]
				}
				parsedBlock, err := strconv.ParseUint(toBlockStr, 16, 64)
				if err != nil {
					return createErrorResponse(-32602, "Invalid toBlock hex value", nil, request.ID)
				}
				toBlock = parsedBlock
			}
		} else {
			toBlock = currentBlock // Default to latest
		}

		// Validate block range: toBlock must not be higher than current block
		if toBlock > currentBlock {
			return createErrorResponse(-32000, "invalid block range params", nil, request.ID)
		}

		// Validate fromBlock <= toBlock
		if fromBlock > toBlock {
			return createErrorResponse(-32000, "invalid block range params", nil, request.ID)
		}

		// Return empty logs array (can be extended later to return actual logs)
		result = []interface{}{}

	default:
		return createErrorResponse(-32601, "Method not found", nil, request.ID)
	}

	if err != nil {
		return createErrorResponse(-32603, err.Error(), nil, request.ID)
	}

	response := JSONRPCResponse{
		JsonRPC: "2.0",
		Result:  result,
		ID:      request.ID,
	}

	return json.Marshal(response)
}
