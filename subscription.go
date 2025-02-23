package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

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

	return id, nil
}

func (sm *SubscriptionManager) Unsubscribe(id uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription %d not found", id)
	}

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

// calculateSolanaEpochRoot calculates the root slot for the current epoch
// For simplicity, we'll use a fixed epoch size of 32 slots
func calculateSolanaEpochRoot(slot uint64) uint64 {
	epochSize := uint64(32)
	return (slot / epochSize) * epochSize
}

func (sm *SubscriptionManager) BroadcastNewBlock(blockNum uint64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, sub := range sm.subscriptions {
		var notification interface{}

		switch sub.Type {
		case "evm":
			if sub.Method != "newHeads" {
				continue
			}
			notification = JSONRPCNotification{
				JsonRPC: "2.0",
				Method:  "eth_subscription",
				Params: map[string]interface{}{
					"subscription": fmt.Sprintf("%d", sub.ID),
					"result": map[string]interface{}{
						"number": fmt.Sprintf("0x%x", blockNum),
					},
				},
			}
		case "solana":
			if sub.Method != "slotNotification" {
				continue
			}
			root := calculateSolanaEpochRoot(blockNum)
			notification = JSONRPCNotification{
				JsonRPC: "2.0",
				Method:  "slotNotification",
				Params: map[string]interface{}{
					"subscription": sub.ID,
					"result": map[string]interface{}{
						"parent": blockNum - 1,
						"root":   root,
						"slot":   blockNum,
					},
				},
			}
		}

		data, err := json.Marshal(notification)
		if err != nil {
			continue
		}

		if err := sub.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// If we can't write to the connection, remove the subscription
			sm.Unsubscribe(sub.ID)
		}
	}
}
