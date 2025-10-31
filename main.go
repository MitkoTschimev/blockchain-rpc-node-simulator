package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for testing
		},
	}
	subManager  = NewSubscriptionManager()
	connTracker = NewConnectionTracker()

	// chainIdToName maps chainIds to their corresponding chain names
	chainIdToName = map[string]string{
		"1":     "ethereum",  // Ethereum Mainnet
		"10":    "optimism",  // Optimism
		"56":    "binance",   // Binance Smart Chain
		"100":   "gnosis",    // Gnosis Chain
		"137":   "polygon",   // Polygon
		"250":   "fantom",    // Fantom
		"324":   "zksync",    // zkSync Era
		"8217":  "kaia",      // kaia
		"8453":  "base",      // Base
		"42161": "arbitrum",  // Arbitrum One
		"43114": "avalanche", // Avalanche
		"59144": "linea",     // Linea
		"501":   "solana",    // Solana
	}
)

func main() {
	// Start block number incrementer for each chain
	for chainName, chain := range supportedChains {
		go func(chainName string, c *EVMChain) {
			// Find chain ID for this chain
			var chainId string
			for id, name := range chainIdToName {
				if name == chainName {
					chainId = id
					break
				}
			}
			if chainId == "" {
				log.Printf("Warning: Could not find chain ID for %s", chainName)
				return
			}

			for {
				time.Sleep(c.BlockInterval)
				// Check if blocks are interrupted
				if atomic.LoadUint32(&c.BlockInterrupt) == 1 {
					continue
				}
				// Check if blocks are paused
				if atomic.LoadUint32(&c.BlockIncrement) == 0 {
					newBlock := atomic.AddUint64(&c.BlockNumber, 1)

					// Update safe block (latest - 32)
					if newBlock > 32 {
						atomic.StoreUint64(&c.SafeBlockNumber, newBlock-32)
					} else {
						atomic.StoreUint64(&c.SafeBlockNumber, 0)
					}

					// Update finalized block (latest - 64)
					if newBlock > 64 {
						atomic.StoreUint64(&c.FinalizedBlockNumber, newBlock-64)
					} else {
						atomic.StoreUint64(&c.FinalizedBlockNumber, 0)
					}

					subManager.BroadcastNewBlock(chainId, newBlock)

					// Generate and broadcast log events per block, spread across the block interval
					// In a real implementation, you would generate logs based on actual contract events
					go func(blockNum uint64, interval time.Duration, logsPerBlock int) {
						if logsPerBlock <= 0 {
							return
						}
						logInterval := interval / time.Duration(logsPerBlock)
						for i := 0; i < logsPerBlock; i++ {
							if i > 0 {
								time.Sleep(logInterval)
							}
							logIndex := atomic.AddUint64(&c.LogIndex, 1) - 1
							logEvent := LogEvent{
								Address:     "0x" + hex.EncodeToString(make([]byte, 20)),
								Topics:      []string{"0x" + hex.EncodeToString(make([]byte, 32))},
								Data:        "0x" + hex.EncodeToString(make([]byte, 32)),
								BlockNumber: blockNum,
								TxHash:      "0x" + hex.EncodeToString(make([]byte, 32)),
								TxIndex:     uint64(i),
								BlockHash:   "0x" + hex.EncodeToString(make([]byte, 32)),
								LogIndex:    logIndex,
								Removed:     false,
							}
							subManager.BroadcastNewLog(chainId, logEvent)
						}
					}(newBlock, c.BlockInterval, c.LogsPerBlock)
				}
			}
		}(chainName, chain)
	}

	// Start Solana slot incrementer
	go func() {
		for {
			time.Sleep(solanaNode.SlotInterval)
			// Check if slots are interrupted
			if atomic.LoadUint32(&solanaNode.BlockInterrupt) == 1 {
				continue
			}
			// Check if slots are paused
			if atomic.LoadUint32(&solanaNode.SlotIncrement) == 0 {
				newSlot := atomic.AddUint64(&solanaNode.SlotNumber, 1)
				subManager.BroadcastNewBlock("501", newSlot)
			}
		}
	}()

	// Create a new ServeMux for better route handling
	mux := http.NewServeMux()

	// Serve static files for the web UI
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/", fs)

	// Unified WebSocket and HTTP endpoints
	mux.HandleFunc("/ws/chain/", handleChainWebSocket)
	mux.HandleFunc("/chain/", handleChainHTTP)

	// SSE endpoints
	mux.HandleFunc("/sse/connections", handleConnectionsSSE)

	// Control endpoints
	handleControlEndpoints(mux)

	// Get port from environment variable or use default
	port := os.Getenv("RPC_PORT")
	if port == "" {
		port = "8545"
	}
	port = ":" + port

	log.Printf("Starting RPC simulator on port %s", port)
	log.Printf("Web UI: http://localhost%s", port)
	log.Printf("Chain endpoints:")
	for chainId, chainName := range chainIdToName {
		log.Printf("  %s: ws://localhost%s/ws/chain/%s", chainName, port, chainId)
	}
	log.Printf("Solana endpoint: ws://localhost%s/ws/chain/501", port)
	log.Printf("Control endpoints:")
	log.Printf("  POST /control/connections/drop - Drop all connections (optional: block_duration_seconds)")
	log.Printf("  POST /control/block/set - Set block number")
	log.Printf("  POST /control/block/pause - Pause block increment")
	log.Printf("  POST /control/block/resume - Resume block increment")
	log.Printf("  POST /control/block/interval - Set block interval")
	log.Printf("  POST /control/block/interrupt - Interrupt block emissions")
	log.Printf("  POST /control/timeout/set - Set response timeout")
	log.Printf("  POST /control/timeout/clear - Clear response timeout")
	log.Printf("  POST /control/chain/reorg - Trigger chain reorganization")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// wsConnWrapper wraps a *websocket.Conn to implement WSConn
type wsConnWrapper struct {
	*websocket.Conn
	writeMu sync.Mutex // Protects writes to the connection
	chainId string     // Store the chainId for this connection
}

func (w *wsConnWrapper) WriteMessage(messageType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.Conn.WriteMessage(messageType, data)
}

func (w *wsConnWrapper) Close() error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.Conn.Close()
}

// GetMessages implements WSConn for compatibility with tests
// This is not used in production, only needed to satisfy the interface
func (w *wsConnWrapper) GetMessages() [][]byte {
	return nil
}

// ClearMessages implements WSConn for compatibility with tests
// This is not used in production, only needed to satisfy the interface
func (w *wsConnWrapper) ClearMessages() {
	// No-op in production
}

func handleConnectionsSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for client disconnection
	clientGone := r.Context().Done()
	ticker := time.NewTicker(time.Second) // Update every second
	defer ticker.Stop()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection counts
	counts := connTracker.GetConnections()
	data, err := json.Marshal(counts)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Keep sending updates
	for {
		select {
		case <-clientGone:
			return // Client disconnected
		case <-ticker.C:
			counts := connTracker.GetConnections()
			data, err := json.Marshal(counts)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleChainWebSocket handles WebSocket connections for all chains
func handleChainWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract chainId from URL path
	chainId := r.URL.Path[len("/ws/chain/"):]
	chainName, exists := chainIdToName[chainId]
	if !exists {
		http.Error(w, "Invalid chain ID", http.StatusBadRequest)
		return
	}

	log.Printf("Client connected to chain %s (chainId: %s)", chainName, chainId)
	if IsBlocked() {
		http.Error(w, "Server is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	conn := &wsConnWrapper{
		Conn:    wsConn,
		chainId: chainId,
	}

	// Track the connection
	connTracker.AddConnection(chainId)
	defer func() {
		connTracker.RemoveConnection(chainId)
		count := subManager.CleanupConnection(conn)
		log.Printf("Cleaned up %d subscriptions for disconnected client (chain: %s)", count, chainName)
		conn.Close()
	}()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client disconnected unexpectedly from chain %s: %v", chainName, err)
			}
			break
		}

		var response []byte
		if chainId == "501" { // Solana
			response, err = handleSolanaRequest(message, conn)
		} else { // EVM chains
			response, err = handleEVMRequest(message, conn, chainId)
		}

		if err != nil {
			log.Printf("Handler error for chain %s: %v", chainName, err)
			break
		}

		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Printf("Write error for chain %s: %v", chainName, err)
			break
		}
	}
}

// handleChainHTTP handles HTTP requests for all chains
func handleChainHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract chainId from URL path
	chainId := r.URL.Path[len("/chain/"):]
	chainName, exists := chainIdToName[chainId]
	if !exists {
		http.Error(w, "Invalid chain ID", http.StatusBadRequest)
		return
	}

	var message []byte
	var err error
	message, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Only log non-health check messages
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err == nil && request.Method != "getHealth" {
		log.Printf("Incoming HTTP message for chain %s: %s", chainName, string(message))
	}

	// Create a mock connection for the request
	mockConn := NewMockWSConn()

	var response []byte
	if chainId == "501" { // Solana
		response, err = handleSolanaRequest(message, mockConn)
	} else { // EVM chains
		response, err = handleEVMRequest(message, mockConn, chainId)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
