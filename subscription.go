package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type Subscription struct {
	ID     string
	Type   string
	Conn   WSConn
	Method string
}

type SubscriptionManager struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	nextSubID     uint64
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
	}
}

func (sm *SubscriptionManager) Subscribe(subType string, conn WSConn, method string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := fmt.Sprintf("%d", atomic.AddUint64(&sm.nextSubID, 1))
	sm.subscriptions[id] = &Subscription{
		ID:     id,
		Type:   subType,
		Conn:   conn,
		Method: method,
	}

	return id, nil
}

func (sm *SubscriptionManager) Unsubscribe(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.subscriptions[id]; !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	delete(sm.subscriptions, id)
	return nil
}

func (sm *SubscriptionManager) DropAllConnections() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := len(sm.subscriptions)
	for _, sub := range sm.subscriptions {
		sub.Conn.Close()
	}
	sm.subscriptions = make(map[string]*Subscription)
	return count
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
					"subscription": sub.ID,
					"result": map[string]interface{}{
						"number": fmt.Sprintf("0x%x", blockNum),
					},
				},
			}
		case "solana":
			if sub.Method != "slotNotification" {
				continue
			}
			notification = JSONRPCNotification{
				JsonRPC: "2.0",
				Method:  "slotNotification",
				Params: map[string]interface{}{
					"subscription": sub.ID,
					"result":       blockNum,
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
