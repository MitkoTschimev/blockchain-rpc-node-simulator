package main

import (
	"math/rand"
)

// ErrorConfig defines a configurable error that can be simulated
type ErrorConfig struct {
	Code        int      `json:"code" yaml:"code"`
	Message     string   `json:"message" yaml:"message"`
	Data        string   `json:"data,omitempty" yaml:"data,omitempty"`
	Probability float64  `json:"probability" yaml:"probability"`               // 0.0 to 1.0
	Methods     []string `json:"methods,omitempty" yaml:"methods,omitempty"`   // If empty, applies to all methods
	DelayMs     int      `json:"delay_ms,omitempty" yaml:"delay_ms,omitempty"` // Delay in milliseconds before returning error (0 = no delay)
}

// PredefinedErrors contains common Ethereum JSON-RPC errors
var PredefinedErrors = map[string]ErrorConfig{
	// Standard JSON-RPC 2.0 errors
	"parse_error": {
		Code:    -32700,
		Message: "Parse error",
		Data:    "Invalid JSON was received by the server",
	},
	"invalid_request": {
		Code:    -32600,
		Message: "Invalid Request",
		Data:    "The JSON sent is not a valid Request object",
	},
	"method_not_found": {
		Code:    -32601,
		Message: "Method not found",
		Data:    "The method does not exist / is not available",
	},
	"invalid_params": {
		Code:    -32602,
		Message: "Invalid params",
		Data:    "Invalid method parameter(s)",
	},
	"internal_error": {
		Code:    -32603,
		Message: "Internal error",
		Data:    "Internal JSON-RPC error",
	},

	// Ethereum-specific errors (-32000 range)
	"header_not_found": {
		Code:    -32000,
		Message: "header not found",
		Data:    "The requested block header was not found",
	},
	"block_not_found": {
		Code:    -32000,
		Message: "block not found",
		Data:    "The requested block was not found",
	},
	"nonce_too_low": {
		Code:    -32000,
		Message: "nonce too low",
		Data:    "Transaction nonce is too low",
		Methods: []string{"eth_sendTransaction", "eth_sendRawTransaction"},
	},
	"stack_limit_reached": {
		Code:    -32000,
		Message: "stack limit reached 1024",
		Data:    "EVM stack limit exceeded",
		Methods: []string{"eth_call", "eth_estimateGas"},
	},
	"execution_timeout": {
		Code:    -32000,
		Message: "execution timeout",
		Data:    "Transaction execution timed out",
		Methods: []string{"eth_call", "eth_estimateGas"},
	},
	"filter_not_found": {
		Code:    -32000,
		Message: "filter not found",
		Data:    "The filter does not exist or has expired",
		Methods: []string{"eth_getFilterChanges", "eth_getFilterLogs", "eth_uninstallFilter"},
	},
	"invalid_block_range": {
		Code:    -32000,
		Message: "invalid block range params",
		Data:    "The block range parameters are invalid (toBlock > current block or fromBlock > toBlock)",
		Methods: []string{"eth_getLogs"},
	},
	"resource_not_found": {
		Code:    -32001,
		Message: "resource not found",
		Data:    "Requested resource was not found",
	},
	"resource_unavailable": {
		Code:    -32002,
		Message: "resource unavailable",
		Data:    "Requested resource is temporarily unavailable",
	},
	"transaction_rejected": {
		Code:    -32003,
		Message: "transaction rejected",
		Data:    "Transaction was rejected",
		Methods: []string{"eth_sendTransaction", "eth_sendRawTransaction"},
	},
	"method_not_supported": {
		Code:    -32004,
		Message: "method not supported",
		Data:    "Method is not implemented",
	},
	"limit_exceeded": {
		Code:    -32005,
		Message: "limit exceeded",
		Data:    "Request exceeds defined limit",
	},
	"gas_limit_too_low": {
		Code:    -32010,
		Message: "gas limit too low",
		Data:    "Transaction gas limit is too low",
		Methods: []string{"eth_call", "eth_estimateGas", "eth_sendTransaction", "eth_sendRawTransaction"},
	},
	"vm_execution_error": {
		Code:    -32015,
		Message: "VM execution error",
		Data:    "Error occurred during contract execution",
		Methods: []string{"eth_call", "eth_estimateGas"},
	},
	"execution_reverted": {
		Code:    3,
		Message: "execution reverted",
		Data:    "Transaction execution was reverted",
		Methods: []string{"eth_call", "eth_estimateGas", "eth_sendTransaction", "eth_sendRawTransaction"},
	},
}

// ShouldSimulateError checks if an error should be simulated for the given method
// Returns the error config to use, or nil if no error should be simulated
func ShouldSimulateError(errorConfigs []ErrorConfig, method string) *ErrorConfig {
	if len(errorConfigs) == 0 {
		return nil
	}

	// Filter applicable errors for this method
	var applicableErrors []ErrorConfig
	for _, errConfig := range errorConfigs {
		// Check if error applies to this method
		if len(errConfig.Methods) == 0 {
			// No method filter, applies to all
			applicableErrors = append(applicableErrors, errConfig)
		} else {
			// Check if method is in the list
			for _, m := range errConfig.Methods {
				if m == method {
					applicableErrors = append(applicableErrors, errConfig)
					break
				}
			}
		}
	}

	if len(applicableErrors) == 0 {
		return nil
	}

	// Calculate total probability
	totalProb := 0.0
	for _, errConfig := range applicableErrors {
		totalProb += errConfig.Probability
	}

	// If total probability is 0, don't simulate any error
	if totalProb == 0 {
		return nil
	}

	// Roll the dice to see if any error should occur
	roll := rand.Float64()

	// If the roll is higher than total probability, no error
	if roll > totalProb {
		return nil
	}

	// Select which error to return based on weighted probability
	cumulative := 0.0
	for i := range applicableErrors {
		cumulative += applicableErrors[i].Probability
		if roll <= cumulative {
			return &applicableErrors[i]
		}
	}

	// Fallback: if we didn't match (shouldn't happen with proper probabilities),
	// return the last error if totalProb >= roll
	if len(applicableErrors) > 0 && totalProb >= roll {
		return &applicableErrors[len(applicableErrors)-1]
	}

	return nil
}
