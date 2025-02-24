package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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
	currentBlock         uint64 = 1 // Start from block 1
	blockIncrementPaused uint32 = 0 // 0 = running, 1 = paused
	subManager                  = NewSubscriptionManager()
)

func main() {
	// Start block number incrementer
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if atomic.LoadUint32(&blockIncrementPaused) == 0 {
				newBlock := atomic.AddUint64(&currentBlock, 1)
				log.Printf("Block number increased to: %d", newBlock)
				subManager.BroadcastNewBlock(newBlock)
			}
		}
	}()

	// Create a new ServeMux for better route handling
	mux := http.NewServeMux()

	// WebSocket endpoints
	mux.HandleFunc("/ws/evm", handleEVMWebSocket)
	mux.HandleFunc("/ws/solana", handleSolanaWebSocket)

	// HTTP endpoints
	mux.HandleFunc("/evm", handleEVMHTTP)
	mux.HandleFunc("/solana", handleSolanaHTTP)

	// Control endpoints
	handleControlEndpoints(mux)

	port := ":8545"
	log.Printf("Starting RPC simulator on port %s", port)
	log.Printf("EVM endpoint: ws://localhost%s/ws/evm", port)
	log.Printf("Solana endpoint: ws://localhost%s/ws/solana", port)
	log.Printf("Control endpoints:")
	log.Printf("  POST /control/connections/drop - Drop all connections (optional: block_duration_seconds)")
	log.Printf("  POST /control/block/set - Set block number")
	log.Printf("  POST /control/block/pause - Pause block increment")
	log.Printf("  POST /control/block/resume - Resume block increment")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// wsConnWrapper wraps a *websocket.Conn to implement WSConn
type wsConnWrapper struct {
	*websocket.Conn
	writeMu sync.Mutex // Protects writes to the connection
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

func handleEVMWebSocket(w http.ResponseWriter, r *http.Request) {
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
		Conn: wsConn,
	}
	defer func() {
		count := subManager.CleanupConnection(conn)
		log.Printf("Cleaned up %d subscriptions for disconnected EVM client", count)
		conn.Close()
	}()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("EVM client disconnected unexpectedly: %v", err)
			}
			break
		}

		response, err := handleEVMRequest(message, conn)
		if err != nil {
			log.Println("Handler error:", err)
			break
		}

		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func handleSolanaWebSocket(w http.ResponseWriter, r *http.Request) {
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
		Conn: wsConn,
	}
	defer func() {
		count := subManager.CleanupConnection(conn)
		log.Printf("Cleaned up %d subscriptions for disconnected Solana client", count)
		conn.Close()
	}()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Solana client disconnected unexpectedly: %v", err)
			}
			break
		}

		response, err := handleSolanaRequest(message, conn)
		if err != nil {
			log.Println("Handler error:", err)
			break
		}

		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func handleEVMHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		log.Printf("Incoming EVM HTTP message: %s", string(message))
	}

	// Create a mock connection for the request
	mockConn := NewMockWSConn()
	response, err := handleEVMRequest(message, mockConn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func handleSolanaHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		log.Printf("Incoming Solana HTTP message: %s", string(message))
	}

	// Create a mock connection for the request
	mockConn := NewMockWSConn()
	response, err := handleSolanaRequest(message, mockConn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
