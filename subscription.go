package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
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

// generateBlockHashForSubscription creates a deterministic hash based on block number and chain ID
func generateBlockHashForSubscription(blockNumber uint64, chainID string, seed string) string {
	// Create a unique input combining block number, chain ID, and seed
	input := fmt.Sprintf("%s-%d-%s", chainID, blockNumber, seed)
	hash := sha256.Sum256([]byte(input))
	return "0x" + hex.EncodeToString(hash[:])
}

// generateValidHexString generates a hex string that matches the pattern ^0x(0|[1-9a-f][0-9a-f]*)$
// This ensures no leading zeros except for "0x0"
func generateValidHexString(byteLength int) string {
	if byteLength == 0 {
		return "0x0"
	}
	// Generate random bytes
	bytes := make([]byte, byteLength)
	for i := range bytes {
		bytes[i] = byte(rand.Intn(256))
	}
	// Ensure first byte is non-zero to avoid leading zeros
	if bytes[0] == 0 {
		bytes[0] = byte(rand.Intn(255) + 1)
	}
	hexStr := hex.EncodeToString(bytes)
	// Remove leading zeros after the first non-zero digit
	hexStr = strings.TrimLeft(hexStr, "0")
	if hexStr == "" {
		hexStr = "0"
	}
	return "0x" + hexStr
}

// BlockNotification represents a new block notification
type BlockNotification struct {
	ParentHash       string        `json:"parentHash"`
	Number           string        `json:"number"`
	Hash             string        `json:"hash"`
	Timestamp        string        `json:"timestamp"`
	GasLimit         string        `json:"gasLimit"`
	GasUsed          string        `json:"gasUsed"`
	Miner            string        `json:"miner"`
	Difficulty       string        `json:"difficulty"`
	TotalDifficulty  string        `json:"totalDifficulty"`
	Size             string        `json:"size"`
	Nonce            string        `json:"nonce"`
	ExtraData        string        `json:"extraData"`
	BaseFeePerGas    string        `json:"baseFeePerGas"`
	Sha3Uncles       string        `json:"sha3Uncles"`
	LogsBloom        string        `json:"logsBloom"`
	TransactionsRoot string        `json:"transactionsRoot"`
	StateRoot        string        `json:"stateRoot"`
	ReceiptsRoot     string        `json:"receiptsRoot"`
	Uncles           []string      `json:"uncles"`
	Transactions     []interface{} `json:"transactions"`
}

// MarshalJSON implements custom JSON marshaling for BlockNotification
func (b BlockNotification) MarshalJSON() ([]byte, error) {
	// Create ordered fields
	fields := []struct {
		Key   string
		Value interface{}
	}{
		{"number", b.Number},
		{"hash", b.Hash},
		{"timestamp", b.Timestamp},
		{"gasLimit", b.GasLimit},
		{"gasUsed", b.GasUsed},
		{"miner", b.Miner},
		{"difficulty", b.Difficulty},
		{"totalDifficulty", b.TotalDifficulty},
		{"size", b.Size},
		{"nonce", b.Nonce},
		{"extraData", b.ExtraData},
		{"baseFeePerGas", b.BaseFeePerGas},
		{"sha3Uncles", b.Sha3Uncles},
		{"logsBloom", b.LogsBloom},
		{"transactionsRoot", b.TransactionsRoot},
		{"stateRoot", b.StateRoot},
		{"receiptsRoot", b.ReceiptsRoot},
		{"uncles", b.Uncles},
		{"transactions", b.Transactions},
	}

	// Randomly decide whether to put parentHash first or last
	port := os.Getenv("RPC_PORT")
	putFirst := port == "8545"

	// Create the final ordered slice
	var orderedFields []struct {
		Key   string
		Value interface{}
	}

	if putFirst {
		orderedFields = append([]struct {
			Key   string
			Value interface{}
		}{{"parentHash", b.ParentHash}}, fields...)
	} else {
		orderedFields = append(fields, struct {
			Key   string
			Value interface{}
		}{"parentHash", b.ParentHash})
	}

	// Create the JSON string manually to preserve order
	var result strings.Builder
	result.WriteString("{")
	for i, field := range orderedFields {
		if i > 0 {
			result.WriteString(",")
		}
		keyJSON, _ := json.Marshal(field.Key)
		valueJSON, _ := json.Marshal(field.Value)
		result.Write(keyJSON)
		result.WriteString(":")
		result.Write(valueJSON)
	}
	result.WriteString("}")

	return []byte(result.String()), nil
}

// Transaction represents a transaction in a block
type Transaction struct {
	Hash             string `json:"hash"`
	Nonce            string `json:"nonce"`
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	TransactionIndex string `json:"transactionIndex"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Input            string `json:"input"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
}

func (sm *SubscriptionManager) BroadcastNewBlock(chain string, blockNumber uint64) {
	// First, get all relevant subscriptions under a read lock
	sm.mu.RLock()
	subs := make([]*Subscription, 0)
	for _, sub := range sm.subscriptions {
		if sub.Type == chain && (sub.Method == "newHeads" || sub.Method == "newHeadsWithTx" || sub.Method == "logs" || sub.Method == "slotNotification" || sub.Method == "rootNotification") {
			subs = append(subs, sub)
		}
	}
	sm.mu.RUnlock()

	// Sort subscriptions by ID to ensure deterministic order
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].ID < subs[j].ID
	})

	// Process each subscription outside the lock
	for _, sub := range subs {
		var notification interface{}
		switch chain {
		case "1", "10", "56", "100", "130", "137", "146", "250", "324", "8217", "8453", "42161", "43114", "59144":
			// Generate unique hashes for this block
			blockHash := generateBlockHashForSubscription(blockNumber, chain, "block")
			var parentHash string
			if blockNumber > 0 {
				parentHash = generateBlockHashForSubscription(blockNumber-1, chain, "block")
			} else {
				parentHash = "0x" + hex.EncodeToString(make([]byte, 32))
			}

			// Generate deterministic hashes for required fields
			sha3Uncles := generateBlockHashForSubscription(blockNumber, chain, "sha3Uncles")
			// logsBloom must be exactly 512 hex characters (256 bytes)
			logsBloom := generateBlockHashForSubscription(blockNumber, chain, "logsBloom")
			// Extend to 256 bytes by repeating the hash pattern
			logsBloomBytes := make([]byte, 256)
			hashBytes, _ := hex.DecodeString(logsBloom[2:]) // Remove "0x" prefix
			for i := 0; i < 256; i++ {
				logsBloomBytes[i] = hashBytes[i%32] // Repeat the 32-byte hash pattern
			}
			logsBloom = "0x" + hex.EncodeToString(logsBloomBytes)
			transactionsRoot := generateBlockHashForSubscription(blockNumber, chain, "transactionsRoot")
			stateRoot := generateBlockHashForSubscription(blockNumber, chain, "stateRoot")
			receiptsRoot := generateBlockHashForSubscription(blockNumber, chain, "receiptsRoot")

			// Create block notification
			block := BlockNotification{
				ParentHash:       parentHash,
				Number:           fmt.Sprintf("0x%x", blockNumber),
				Hash:             blockHash,
				Timestamp:        fmt.Sprintf("0x%x", time.Now().Unix()),
				GasLimit:         generateValidHexString(32),
				GasUsed:          generateValidHexString(32),
				Miner:            "0x" + hex.EncodeToString(make([]byte, 20)),
				Difficulty:       generateValidHexString(32),
				TotalDifficulty:  generateValidHexString(32),
				Size:             generateValidHexString(32),
				Nonce:            "0x" + hex.EncodeToString(make([]byte, 8)),
				ExtraData:        "0x" + hex.EncodeToString(make([]byte, 32)),
				BaseFeePerGas:    generateValidHexString(32),
				Sha3Uncles:       sha3Uncles,
				LogsBloom:        logsBloom,
				TransactionsRoot: transactionsRoot,
				StateRoot:        stateRoot,
				ReceiptsRoot:     receiptsRoot,
				Uncles:           []string{},
			}

			// Add transactions if subscription type is newHeadsWithTx
			if sub.Method == "newHeadsWithTx" {
				// Generate a random number of transactions (1-5)
				numTx := rand.Intn(5) + 1
				transactions := make([]Transaction, numTx)
				for i := 0; i < numTx; i++ {
					transactions[i] = Transaction{
						Hash:             "0x" + hex.EncodeToString(make([]byte, 32)),
						Nonce:            fmt.Sprintf("0x%x", rand.Uint64()),
						BlockHash:        blockHash, // Use the deterministic block hash
						BlockNumber:      fmt.Sprintf("0x%x", blockNumber),
						TransactionIndex: fmt.Sprintf("0x%x", i),
						From:             "0x" + hex.EncodeToString(make([]byte, 20)),
						To:               "0x" + hex.EncodeToString(make([]byte, 20)),
						Value:            "0x" + hex.EncodeToString(make([]byte, 32)),
						Gas:              "0x" + hex.EncodeToString(make([]byte, 32)),
						GasPrice:         "0x" + hex.EncodeToString(make([]byte, 32)),
						Input:            "0x" + hex.EncodeToString(make([]byte, 32)),
						V:                "0x" + hex.EncodeToString(make([]byte, 1)),
						R:                "0x" + hex.EncodeToString(make([]byte, 32)),
						S:                "0x" + hex.EncodeToString(make([]byte, 32)),
					}
				}
				block.Transactions = make([]interface{}, len(transactions))
				for i, tx := range transactions {
					block.Transactions[i] = tx
				}
			} else {
				block.Transactions = []interface{}{} // Empty array for regular newHeads
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

			// Handle different subscription types for Solana
			if sub.Method == "slotNotification" {
				// Regular slot notification - sent for every slot
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
			} else if sub.Method == "rootNotification" {
				// Root notification - only send for finalized (rooted) slots
				// Only broadcast if this slot is now the root (finalized)
				// This means we broadcast the root slot, not the current slot
				if root > 0 {
					notification = JSONRPCNotification{
						JsonRPC: "2.0",
						Method:  "rootNotification",
						Params: SubscriptionParams{
							Subscription: sub.ID,
							Result:       root, // Just the rooted slot number
						},
					}
				} else {
					// Skip if no root yet
					continue
				}
			}
		default:
			// Skip broadcasting for unknown chains
			log.Printf("Warning: Unknown chain ID %s in BroadcastNewBlock", chain)
			continue
		}

		// Skip if notification is nil (shouldn't happen with default case, but safety check)
		if notification == nil {
			continue
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

type SubscriptionParams struct {
	Subscription interface{} `json:"subscription"`
	Result       interface{} `json:"result"`
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
