package main

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

func handleEVMRequest(message []byte, conn *websocket.Conn) ([]byte, error) {
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return createErrorResponse(-32700, "Parse error", nil, nil)
	}

	// Validate JSON-RPC version
	if request.JsonRPC != "2.0" {
		return createErrorResponse(-32600, "Invalid Request", nil, request.ID)
	}

	var result interface{}
	var err error

	switch request.Method {
	case "eth_chainId":
		result = "0x1" // Mainnet
	case "eth_blockNumber":
		result = fmt.Sprintf("0x%x", atomic.LoadUint64(&currentBlock))
	case "eth_getBalance":
		result = "0x1234567890"
	case "eth_subscribe":
		if len(request.Params) < 1 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}
		subscriptionType, ok := request.Params[0].(string)
		if !ok {
			return createErrorResponse(-32602, "Invalid subscription type", nil, request.ID)
		}

		if subscriptionType != "newHeads" {
			return createErrorResponse(-32601, "Unsupported subscription type", nil, request.ID)
		}

		subID, err := subManager.Subscribe("evm", conn, "newHeads")
		if err != nil {
			return createErrorResponse(-32603, err.Error(), nil, request.ID)
		}
		result = subID

	case "eth_unsubscribe":
		if len(request.Params) < 1 {
			return createErrorResponse(-32602, "Invalid params", nil, request.ID)
		}
		subscriptionID, ok := request.Params[0].(string)
		if !ok {
			return createErrorResponse(-32602, "Invalid subscription ID", nil, request.ID)
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
