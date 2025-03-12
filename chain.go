package main

import (
	"log"
	"sync/atomic"
	"time"
)

// Chain interface defines methods that both EVM and Solana chains must implement
type Chain interface {
	SetTimeout(duration time.Duration)
	ClearTimeout()
	InterruptBlocks()
	ResumeBlocks()
	TriggerReorg(blocks int)
}

type EVMChain struct {
	Name            string
	ChainID         string
	BlockNumber     uint64
	BlockInterval   time.Duration
	BlockIncrement  uint32 // 0 = normal, 1 = paused
	BlockInterrupt  uint32 // 0 = normal, 1 = interrupted
	ResponseTimeout time.Duration
}

type SolanaNode struct {
	SlotNumber      uint64
	SlotInterval    time.Duration
	SlotIncrement   uint32 // 0 = normal, 1 = paused
	BlockInterrupt  uint32 // 0 = normal, 1 = interrupted
	ResponseTimeout time.Duration
	Version         string
	FeatureSet      uint32
}

var supportedChains = map[string]*EVMChain{
	"ethereum": {
		Name:          "ethereum",
		ChainID:       "0x1", // 1 Mainnet
		BlockInterval: 12 * time.Second,
	},
	"optimism": {
		Name:          "optimism",
		ChainID:       "0xa", // 10
		BlockInterval: 2 * time.Second,
	},
	"binance": {
		Name:          "binance",
		ChainID:       "0x38", // 56
		BlockInterval: 3 * time.Second,
	},
	"gnosis": {
		Name:          "gnosis",
		ChainID:       "0x64", // 100
		BlockInterval: 5 * time.Second,
	},
	"polygon": {
		Name:          "polygon",
		ChainID:       "0x89", // 137
		BlockInterval: 2 * time.Second,
	},
	"fantom": {
		Name:          "fantom",
		ChainID:       "0xfa", // 250
		BlockInterval: 1 * time.Second,
	},
	"zksync": {
		Name:          "zksync",
		ChainID:       "0x144", // 324
		BlockInterval: 1 * time.Second,
	},
	"klaytn": {
		Name:          "klaytn",
		ChainID:       "0x2019", // 8217
		BlockInterval: 1 * time.Second,
	},
	"base": {
		Name:          "base",
		ChainID:       "0x2105", // 8453
		BlockInterval: 2 * time.Second,
	},
	"arbitrum": {
		Name:          "arbitrum",
		ChainID:       "0xa4b1", // 42161
		BlockInterval: 250 * time.Millisecond,
	},
	"avalanche": {
		Name:          "avalanche",
		ChainID:       "0xa86a", // 43114
		BlockInterval: 2 * time.Second,
	},
	"linea": {
		Name:          "linea",
		ChainID:       "0xe708", // 59144
		BlockInterval: 12 * time.Second,
	},
}

var solanaNode = &SolanaNode{
	SlotInterval: 400 * time.Millisecond,
	Version:      "1.14.10",
	FeatureSet:   1,
}

// EVMChain methods
func (c *EVMChain) SetTimeout(duration time.Duration) {
	c.ResponseTimeout = duration
}

func (c *EVMChain) ClearTimeout() {
	c.ResponseTimeout = 0
}

func (c *EVMChain) InterruptBlocks() {
	atomic.StoreUint32(&c.BlockInterrupt, 1)
	log.Printf("Block emissions interrupted for chain %s", c.Name)
}

func (c *EVMChain) ResumeBlocks() {
	atomic.StoreUint32(&c.BlockInterrupt, 0)
	log.Printf("Block emissions resumed for chain %s", c.Name)
}

func (c *EVMChain) TriggerReorg(blocks int) {
	currentBlock := atomic.LoadUint64(&c.BlockNumber)
	if currentBlock < uint64(blocks) {
		return
	}

	// Revert blocks
	atomic.StoreUint64(&c.BlockNumber, currentBlock-uint64(blocks))

	// Broadcast the reorg through the subscription manager
	subManager.BroadcastNewBlock(c.ChainID, currentBlock-uint64(blocks))
}

// SolanaNode methods
func (n *SolanaNode) SetTimeout(duration time.Duration) {
	n.ResponseTimeout = duration
}

func (n *SolanaNode) ClearTimeout() {
	n.ResponseTimeout = 0
}

func (n *SolanaNode) InterruptBlocks() {
	atomic.StoreUint32(&n.BlockInterrupt, 1)
	log.Printf("Slot emissions interrupted for Solana")
}

func (n *SolanaNode) ResumeBlocks() {
	atomic.StoreUint32(&n.BlockInterrupt, 0)
	log.Printf("Slot emissions resumed for Solana")
}

func (n *SolanaNode) TriggerReorg(blocks int) {
	currentSlot := atomic.LoadUint64(&n.SlotNumber)
	if currentSlot < uint64(blocks) {
		return
	}

	// Revert slots
	atomic.StoreUint64(&n.SlotNumber, currentSlot-uint64(blocks))

	// Broadcast the reorg through the subscription manager
	subManager.BroadcastNewBlock("solana", currentSlot-uint64(blocks))
}
