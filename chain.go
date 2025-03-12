package main

import (
	"log"
	"os"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
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
	Name            string `yaml:"name"`
	ChainID         string `yaml:"chain_id"`
	BlockNumber     uint64
	BlockInterval   time.Duration `yaml:"block_interval"`
	BlockIncrement  uint32        // 0 = normal, 1 = paused
	BlockInterrupt  uint32        // 0 = normal, 1 = interrupted
	ResponseTimeout time.Duration
}

type SolanaNode struct {
	SlotNumber      uint64
	SlotInterval    time.Duration `yaml:"slot_interval"`
	SlotIncrement   uint32        // 0 = normal, 1 = paused
	BlockInterrupt  uint32        // 0 = normal, 1 = interrupted
	ResponseTimeout time.Duration
	Version         string `yaml:"version"`
	FeatureSet      uint32 `yaml:"feature_set"`
}

type ChainConfig struct {
	EVMChains map[string]*EVMChain `yaml:"evm_chains"`
	Solana    *SolanaNode          `yaml:"solana"`
}

var (
	supportedChains map[string]*EVMChain
	solanaNode      *SolanaNode
)

func init() {
	// Load chain configurations from YAML file
	data, err := os.ReadFile("chains.yaml")
	if err != nil {
		log.Fatalf("Failed to read chains.yaml: %v", err)
	}

	var config ChainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse chains.yaml: %v", err)
	}

	// Initialize global variables
	supportedChains = config.EVMChains
	solanaNode = config.Solana

	// Initialize block numbers for each chain
	for _, chain := range supportedChains {
		chain.BlockNumber = 1
		chain.BlockIncrement = 0
	}
	// Initialize Solana slot number
	solanaNode.SlotNumber = 1
	solanaNode.SlotIncrement = 0
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
	subManager.BroadcastNewBlock("501", currentSlot-uint64(blocks))
}
