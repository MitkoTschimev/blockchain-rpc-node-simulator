package main

import (
	"fmt"
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
	Name                 string        `yaml:"name"`
	ChainID              string        `yaml:"chain_id"`
	BlockNumber          uint64        `yaml:"block_number"`          // Latest block number
	SafeBlockNumber      uint64        `yaml:"safe_block_number"`      // Safe block (typically latest - 32 slots)
	FinalizedBlockNumber uint64        `yaml:"finalized_block_number"` // Finalized block (typically latest - 64 slots)
	BlockIncrement       uint32        `yaml:"block_increment"`
	BlockInterrupt       uint32        `yaml:"block_interrupt"`
	BlockInterval        time.Duration `yaml:"block_interval"`
	ResponseTimeout      time.Duration
	Latency              time.Duration `yaml:"latency"`
	ErrorProbability     float64       `yaml:"error_probability"` // Deprecated: use ErrorConfigs instead
	ErrorConfigs         []ErrorConfig `yaml:"error_configs" json:"error_configs"` // Configurable error simulation
	LogsPerBlock         int           `yaml:"logs_per_block"`    // Number of log events to generate per block
	LogIndex             uint64        // Incremental counter for log events
}

type SolanaNode struct {
	SlotNumber      uint64
	SlotInterval    time.Duration `yaml:"slot_interval"`
	SlotIncrement   uint32        // 0 = normal, 1 = paused
	BlockInterrupt  uint32        // 0 = normal, 1 = interrupted
	ResponseTimeout time.Duration
	Version         string        `yaml:"version"`
	FeatureSet      uint32        `yaml:"feature_set"`
	Latency         time.Duration `yaml:"latency"`
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
		// Set default logs per block if not configured
		if chain.LogsPerBlock == 0 {
			chain.LogsPerBlock = 5
		}
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

// SaveChainConfig saves the chain configuration to a YAML file
func SaveChainConfig(filename string, config *ChainConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// LoadChainConfig loads the chain configuration from a YAML file
func LoadChainConfig(filename string) (*ChainConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config ChainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}
