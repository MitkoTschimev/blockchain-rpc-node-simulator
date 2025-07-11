package main

import (
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

	// Randomly return header not found error based on probability
	if chain.ErrorProbability > 0 && rand.Float64() < chain.ErrorProbability {
		return createErrorResponse(-32000, "header not found", nil, request.ID)
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
		result = fmt.Sprintf("0x%x", atomic.LoadUint64(&chain.BlockNumber))
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
