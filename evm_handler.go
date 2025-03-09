package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"
)

type EVMChain struct {
	Name           string
	ChainID        string // hex string
	BlockNumber    uint64
	BlockIncrement uint32 // 0 = running, 1 = paused
	BlockInterval  time.Duration
}

var (
	supportedChains = map[string]*EVMChain{
		"ethereum": {
			Name:          "ethereum",
			ChainID:       "0x1", // 1 Mainnet
			BlockInterval: 12 * time.Second,
		},
		"optimism": {
			Name:          "optimism",
			ChainID:       "0xa", // 10
			BlockInterval: 2 * time.Second,
		},
		"arbitrum": {
			Name:          "arbitrum",
			ChainID:       "0xa4b1", // 42161
			BlockInterval: 250 * time.Millisecond,
		},
		"avalanche": {
			Name:          "avalanche",
			ChainID:       "0xa86a", // 43114
			BlockInterval: 2 * time.Second,
		},
		"base": {
			Name:          "base",
			ChainID:       "0x2105", // 8453
			BlockInterval: 2 * time.Second,
		},
		"binance": {
			Name:          "binance",
			ChainID:       "0x38", // 56
			BlockInterval: 3 * time.Second,
		},
	}
)

func init() {
	// Initialize block numbers for each chain
	for _, chain := range supportedChains {
		chain.BlockNumber = 1
		chain.BlockIncrement = 0
	}
}

func handleEVMRequest(message []byte, conn WSConn) ([]byte, error) {
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

	// Get chain from params or default to ethereum
	chainName := "ethereum"
	if len(request.Params) > 0 {
		if chainParam, ok := request.Params[0].(string); ok {
			chainName = chainParam
		}
	}

	chain, ok := supportedChains[chainName]
	if !ok {
		return createErrorResponse(-32602, fmt.Sprintf("Unsupported chain: %s", chainName), nil, request.ID)
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
	case "getHealth":
		result = "ok"
	case "eth_getBlockByNumber":
		result = fmt.Sprintf("0x%x", atomic.LoadUint64(&chain.BlockNumber))
	case "eth_subscribe":
		if len(request.Params) < 2 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}
		subscriptionType, ok := request.Params[1].(string)
		if !ok {
			return createErrorResponse(-32602, "Invalid subscription type", nil, request.ID)
		}

		if subscriptionType != "newHeads" {
			return createErrorResponse(-32601, "Unsupported subscription type", nil, request.ID)
		}

		subID, err := subManager.Subscribe(chainName, conn, "newHeads")
		if err != nil {
			return createErrorResponse(-32603, err.Error(), nil, request.ID)
		}
		log.Printf("New EVM subscription created: ID=%d, Chain=%s, Type=%s", subID, chainName, subscriptionType)
		result = fmt.Sprintf("%d", subID) // Return subscription ID as string for EVM

	case "eth_unsubscribe":
		if len(request.Params) < 1 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}

		// Handle both string and number subscription IDs
		var subscriptionID uint64
		switch v := request.Params[0].(type) {
		case string:
			subscriptionID, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return createErrorResponse(-32602, "Invalid subscription ID", nil, request.ID)
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
