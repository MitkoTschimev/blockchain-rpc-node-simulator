package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startTestServer starts the server for testing and returns a cleanup function
func startTestServer(t *testing.T) (string, func()) {
	// Initialize the subscription manager and connection tracker
	subManager = NewSubscriptionManager()
	connTracker = NewConnectionTracker()

	// Create a new ServeMux for the test server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/chain/", handleChainWebSocket)

	// Create a listener first to get the actual port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Get the actual address
	serverAddr := listener.Addr().String()

	// Create the server
	server := &http.Server{
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Return the cleanup function
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shut down server: %v", err)
		}
	}

	return serverAddr, cleanup
}

func TestE2EChainConnections(t *testing.T) {
	// Save and restore original global state
	originalSupportedChains := supportedChains
	originalSolanaNode := solanaNode
	defer func() {
		supportedChains = originalSupportedChains
		solanaNode = originalSolanaNode
	}()

	// Initialize chain configurations for testing
	supportedChains = map[string]*EVMChain{
		"ethereum": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"optimism": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"binance": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"gnosis": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"polygon": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"fantom": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"zksync": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"kaia": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"base": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"arbitrum": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"avalanche": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
		"linea": {
			BlockNumber:    0,
			BlockInterval:  100 * time.Millisecond,
			BlockIncrement: 0,
		},
	}

	solanaNode = &SolanaNode{
		SlotNumber:    0,
		SlotInterval:  100 * time.Millisecond,
		SlotIncrement: 0,
	}

	// Start the test server
	serverAddr, cleanup := startTestServer(t)
	defer cleanup()

	// Initialize block incrementers for each chain
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
				t.Errorf("Could not find chain ID for %s", chainName)
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
					subManager.BroadcastNewBlock(chainId, newBlock)
				}
			}
		}(chainName, chain)
	}

	// Initialize Solana slot incrementer
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

	// Test cases for different chains
	chains := []struct {
		id       string
		isEVM    bool
		subType  string
		method   string
		validate func(notification JSONRPCNotification) error
	}{
		{
			id:      "1",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "10",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "56",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "100",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "137",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "250",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "324",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "8217",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "8453",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "42161",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "43114",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "59144",
			isEVM:   true,
			subType: "newHeads",
			method:  "eth_subscription",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["number"].(string); !ok {
					return fmt.Errorf("missing or invalid block number")
				}
				return nil
			},
		},
		{
			id:      "501",
			isEVM:   false,
			subType: "501",
			method:  "slotNotification",
			validate: func(n JSONRPCNotification) error {
				params, ok := n.Params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid params format")
				}
				result, ok := params["result"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid result format")
				}
				if _, ok := result["slot"].(float64); !ok {
					return fmt.Errorf("missing or invalid slot number")
				}
				return nil
			},
		},
	}

	var wg sync.WaitGroup
	results := make(chan error, len(chains))

	for _, chain := range chains {
		wg.Add(1)
		go func(c struct {
			id       string
			isEVM    bool
			subType  string
			method   string
			validate func(notification JSONRPCNotification) error
		}) {
			defer wg.Done()

			// Create a real WebSocket connection
			wsURL := fmt.Sprintf("ws://%s/ws/chain/%s", serverAddr, c.id)
			wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				results <- fmt.Errorf("chain %s: failed to connect: %v", chainIdToName[c.id], err)
				return
			}
			defer wsConn.Close()

			// Create subscription
			if c.isEVM {
				// Create EVM subscription
				request := JSONRPCRequest{
					JsonRPC: "2.0",
					Method:  "eth_subscribe",
					Params:  []interface{}{c.subType},
					ID:      1,
				}
				if err := wsConn.WriteJSON(request); err != nil {
					results <- fmt.Errorf("chain %s: failed to send subscription request: %v", chainIdToName[c.id], err)
					return
				}
			} else {
				// Create Solana subscription
				request := JSONRPCRequest{
					JsonRPC: "2.0",
					Method:  "slotSubscribe",
					Params:  []interface{}{},
					ID:      1,
				}
				if err := wsConn.WriteJSON(request); err != nil {
					results <- fmt.Errorf("chain %s: failed to send subscription request: %v", chainIdToName[c.id], err)
					return
				}
			}

			// Read subscription response
			var subResp JSONRPCResponse
			if err := wsConn.ReadJSON(&subResp); err != nil {
				results <- fmt.Errorf("chain %s: failed to read subscription response: %v", chainIdToName[c.id], err)
				return
			}

			// Wait for notification
			var notification JSONRPCNotification
			maxRetries := 10
			success := false
			for i := 0; i < maxRetries; i++ {
				if err := wsConn.ReadJSON(&notification); err != nil {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				success = true
				break
			}

			if !success {
				results <- fmt.Errorf("chain %s: failed to receive notification after %d retries", chainIdToName[c.id], maxRetries)
				return
			}

			// Validate notification
			if notification.Method != c.method {
				results <- fmt.Errorf("chain %s: expected method %s, got %s", chainIdToName[c.id], c.method, notification.Method)
				return
			}

			if validateErr := c.validate(notification); validateErr != nil {
				results <- fmt.Errorf("chain %s: validation failed: %v", chainIdToName[c.id], validateErr)
				return
			}

			results <- nil
		}(chain)
	}

	// Wait for all tests to complete
	wg.Wait()
	close(results)

	// Check results
	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}
}
