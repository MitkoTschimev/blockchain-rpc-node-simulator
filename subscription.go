package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Subscription struct {
	ID     uint64
	Type   string
	Conn   WSConn
	Method string
}

type SubscriptionManager struct {
	mu            sync.RWMutex
	subscriptions map[uint64]*Subscription
	nextSubID     uint64
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[uint64]*Subscription),
	}
}

func (sm *SubscriptionManager) Subscribe(subType string, conn WSConn, method string) (uint64, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := atomic.AddUint64(&sm.nextSubID, 1)
	sm.subscriptions[id] = &Subscription{
		ID:     id,
		Type:   subType,
		Conn:   conn,
		Method: method,
	}

	log.Printf("Created subscription: ID=%d, Type=%s, Method=%s", id, subType, method)
	return id, nil
}

func (sm *SubscriptionManager) Unsubscribe(id uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log.Printf("Looking for subscription with ID: %d", id)
	log.Printf("Current subscriptions: %d", len(sm.subscriptions))

	sub, exists := sm.subscriptions[id]
	if !exists {
		log.Printf("Subscription %d not found", id)
		return fmt.Errorf("subscription %d not found", id)
	}

	log.Printf("Found subscription: ID=%d, Type=%s, Method=%s", id, sub.Type, sub.Method)
	delete(sm.subscriptions, id)
	log.Printf("Subscription removed: ID=%d, Type=%s, Method=%s", id, sub.Type, sub.Method)
	return nil
}

// CleanupConnection removes all subscriptions associated with a specific connection
func (sm *SubscriptionManager) CleanupConnection(conn WSConn) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	for id, sub := range sm.subscriptions {
		if sub.Conn == conn {
			delete(sm.subscriptions, id)
			log.Printf("Subscription cleaned up on connection close: ID=%d, Type=%s, Method=%s", id, sub.Type, sub.Method)
			count++
		}
	}
	return count
}

func (sm *SubscriptionManager) DropAllConnections() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := len(sm.subscriptions)
	for id, sub := range sm.subscriptions {
		log.Printf("Subscription dropped: ID=%d, Type=%s, Method=%s", id, sub.Type, sub.Method)
		sub.Conn.Close()
	}
	sm.subscriptions = make(map[uint64]*Subscription)
	return count
}

func (sm *SubscriptionManager) BroadcastNewBlock(chain string, blockNumber uint64) {
	// First, get all relevant subscriptions under a read lock
	sm.mu.RLock()
	subs := make([]*Subscription, 0)
	for _, sub := range sm.subscriptions {
		if sub.Type == chain {
			subs = append(subs, sub)
		}
	}
	sm.mu.RUnlock()

	// Process each subscription outside the lock
	for _, sub := range subs {
		var notification interface{}
		switch chain {
		case "1", "10", "56", "100", "137", "250", "324", "8217", "8453", "42161", "43114", "59144":
			// Create block notification
			block := map[string]interface{}{
				"number":          fmt.Sprintf("0x%x", blockNumber),
				"hash":            "0x" + hex.EncodeToString(make([]byte, 32)),
				"parentHash":      "0x" + hex.EncodeToString(make([]byte, 32)),
				"timestamp":       fmt.Sprintf("0x%x", time.Now().Unix()),
				"gasLimit":        "0x" + hex.EncodeToString(make([]byte, 32)),
				"gasUsed":         "0x" + hex.EncodeToString(make([]byte, 32)),
				"miner":           "0x" + hex.EncodeToString(make([]byte, 20)),
				"difficulty":      "0x" + hex.EncodeToString(make([]byte, 32)),
				"totalDifficulty": "0x" + hex.EncodeToString(make([]byte, 32)),
				"size":            "0x" + hex.EncodeToString(make([]byte, 32)),
				"nonce":           "0x" + hex.EncodeToString(make([]byte, 8)),
				"extraData":       "0x" + hex.EncodeToString(make([]byte, 32)),
				"baseFeePerGas":   "0x" + hex.EncodeToString(make([]byte, 32)),
				"uncles":          []string{},
			}

			// Add transactions if subscription type is newHeadsWithTx
			if sub.Method == "newHeadsWithTx" {
				// Generate a random number of transactions (1-5)
				numTx := rand.Intn(5) + 1
				transactions := make([]map[string]interface{}, numTx)
				for i := 0; i < numTx; i++ {
					transactions[i] = map[string]interface{}{
						"hash":             "0x" + hex.EncodeToString(make([]byte, 32)),
						"nonce":            fmt.Sprintf("0x%x", rand.Uint64()),
						"blockHash":        "0x" + hex.EncodeToString(make([]byte, 32)),
						"blockNumber":      fmt.Sprintf("0x%x", blockNumber),
						"transactionIndex": fmt.Sprintf("0x%x", i),
						"from":             "0x" + hex.EncodeToString(make([]byte, 20)),
						"to":               "0x" + hex.EncodeToString(make([]byte, 20)),
						"value":            "0x" + hex.EncodeToString(make([]byte, 32)),
						"gas":              "0x" + hex.EncodeToString(make([]byte, 32)),
						"gasPrice":         "0x" + hex.EncodeToString(make([]byte, 32)),
						"input":            "0x" + hex.EncodeToString(make([]byte, 32)),
						"v":                "0x" + hex.EncodeToString(make([]byte, 1)),
						"r":                "0x" + hex.EncodeToString(make([]byte, 32)),
						"s":                "0x" + hex.EncodeToString(make([]byte, 32)),
					}
				}
				block["transactions"] = transactions
			} else {
				block["transactions"] = []interface{}{} // Empty array for regular newHeads
			}

			notification = JSONRPCNotification{
				JsonRPC: "2.0",
				Method:  "eth_subscription",
				Params: SubscriptionParams{
					Subscription: fmt.Sprintf("0x%x", sub.ID),
					Result:       block,
				},
			}

		case "501":
			// Calculate root as a few blocks behind the current slot
			root := uint64(0)
			if blockNumber > 3 {
				root = blockNumber - 3
			}
			notification = JSONRPCNotification{
				JsonRPC: "2.0",
				Method:  "slotNotification",
				Params: SubscriptionParams{
					Subscription: sub.ID, // Solana uses numeric IDs
					Result: map[string]interface{}{
						"parent": blockNumber - 1,
						"root":   root,
						"slot":   blockNumber,
					},
				},
			}
		}

		data, err := json.Marshal(notification)
		if err != nil {
			continue
		}

		if sub.Method == "newHeadsWithTx" {
			log.Printf("Broadcasting block notification")
		}
		if err := sub.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// If we can't write to the connection, remove the subscription
			sm.Unsubscribe(sub.ID)
		}
	}
}

type SubscriptionParams struct {
	Subscription interface{}            `json:"subscription"`
	Result       map[string]interface{} `json:"result"`
}

// LogEvent represents an EVM log event
type LogEvent struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber uint64   `json:"blockNumber"`
	TxHash      string   `json:"transactionHash"`
	TxIndex     uint64   `json:"transactionIndex"`
	BlockHash   string   `json:"blockHash"`
	LogIndex    uint64   `json:"logIndex"`
	Removed     bool     `json:"removed"`
}

// BroadcastNewLog broadcasts a new log event to all subscribers
func (sm *SubscriptionManager) BroadcastNewLog(chainId string, logEvent LogEvent) {
	// First, get all relevant subscriptions under a read lock
	sm.mu.RLock()
	subs := make([]*Subscription, 0)
	for _, sub := range sm.subscriptions {
		if sub.Type == chainId && sub.Method == "logs" {
			subs = append(subs, sub)
		}
	}
	sm.mu.RUnlock()

	// Process each subscription outside the lock
	for _, sub := range subs {
		// Create the notification
		notification := JSONRPCNotification{
			JsonRPC: "2.0",
			Method:  "eth_subscription",
			Params: struct {
				Subscription string   `json:"subscription"`
				Result       LogEvent `json:"result"`
			}{
				Subscription: fmt.Sprintf("0x%x", sub.ID),
				Result:       logEvent,
			},
		}

		// Marshal the notification
		message, err := json.Marshal(notification)
		if err != nil {
			log.Printf("Error marshaling log notification: %v", err)
			continue
		}

		if err := sub.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error sending log notification: %v", err)
			// If we can't write to the connection, remove the subscription
			sm.Unsubscribe(sub.ID)
		}
	}
}

// getSubscriptionID returns the subscription ID for a given chain and type
func (sm *SubscriptionManager) getSubscriptionID(chainId, subType string) uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, sub := range sm.subscriptions {
		if sub.Type == chainId && sub.Method == subType {
			return sub.ID
		}
	}
	return 0
}
