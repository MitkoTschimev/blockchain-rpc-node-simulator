package main

import (
	"encoding/json"
	"log"
	"strconv"
	"sync/atomic"
)

func handleSolanaRequest(message []byte, conn WSConn) ([]byte, error) {
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return createErrorResponse(-32700, "Parse error", nil, nil)
	}

	// Only log non-health check messages
	if request.Method != "getHealth" {
		log.Printf("Incoming Solana message: %s", string(message))
	}

	// Validate JSON-RPC version
	if request.JsonRPC != "2.0" {
		return createErrorResponse(-32600, "Invalid Request", nil, request.ID)
	}

	var result interface{}
	var err error

	switch request.Method {
	case "getSlot":
		result = atomic.LoadUint64(&solanaNode.SlotNumber)
	case "getVersion":
		result = map[string]interface{}{
			"solana-core": solanaNode.Version,
			"feature-set": solanaNode.FeatureSet,
		}
	case "getHealth":
		result = "ok"
	case "slotSubscribe":
		subID, err := subManager.Subscribe("501", conn, "slotNotification")
		if err != nil {
			return createErrorResponse(-32603, err.Error(), nil, request.ID)
		}
		log.Printf("New Solana subscription created: ID=%d", subID)
		result = subID // Solana uses numeric IDs

	case "slotUnsubscribe":
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
