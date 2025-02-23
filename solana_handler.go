package main

import (
	"encoding/json"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

func handleSolanaRequest(message []byte, conn *websocket.Conn) ([]byte, error) {
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
	case "getSlot":
		result = atomic.LoadUint64(&currentBlock)
	case "getVersion":
		result = map[string]interface{}{
			"solana-core": "1.14.10",
			"feature-set": 1234567,
		}
	case "slotSubscribe":
		subID, err := subManager.Subscribe("solana", conn, "slotNotification")
		if err != nil {
			return createErrorResponse(-32603, err.Error(), nil, request.ID)
		}
		result = subID

	case "slotUnsubscribe":
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
