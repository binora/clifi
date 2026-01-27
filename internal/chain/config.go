package chain

import "math/big"

// ChainConfig holds configuration for an EVM chain.
// Invariant: ChainID and ChainIDInt must always represent the same value.
// ChainIDInt exists for YAML serialization (big.Int doesn't serialize cleanly).
// ChainID is used at runtime for RPC calls and transaction signing.
type ChainConfig struct {
	Name           string   `yaml:"name"`
	ChainID        *big.Int `yaml:"-"`        // Runtime use (signing, RPC validation)
	ChainIDInt     int64    `yaml:"chain_id"` // YAML serialization
	RPCURLs        []string `yaml:"rpc_urls"`
	ExplorerURL    string   `yaml:"explorer_url"`
	NativeCurrency string   `yaml:"native_currency"`
	IsTestnet      bool     `yaml:"is_testnet"`
}

// DefaultChains returns the default chain configurations
func DefaultChains() map[string]*ChainConfig {
	return map[string]*ChainConfig{
		"ethereum": {
			Name:           "Ethereum Mainnet",
			ChainID:        big.NewInt(1),
			ChainIDInt:     1,
			RPCURLs:        []string{"https://eth.llamarpc.com", "https://rpc.ankr.com/eth"},
			ExplorerURL:    "https://etherscan.io",
			NativeCurrency: "ETH",
			IsTestnet:      false,
		},
		"base": {
			Name:           "Base",
			ChainID:        big.NewInt(8453),
			ChainIDInt:     8453,
			RPCURLs:        []string{"https://mainnet.base.org", "https://base.llamarpc.com"},
			ExplorerURL:    "https://basescan.org",
			NativeCurrency: "ETH",
			IsTestnet:      false,
		},
		"arbitrum": {
			Name:           "Arbitrum One",
			ChainID:        big.NewInt(42161),
			ChainIDInt:     42161,
			RPCURLs:        []string{"https://arb1.arbitrum.io/rpc", "https://arbitrum.llamarpc.com"},
			ExplorerURL:    "https://arbiscan.io",
			NativeCurrency: "ETH",
			IsTestnet:      false,
		},
		"optimism": {
			Name:           "Optimism",
			ChainID:        big.NewInt(10),
			ChainIDInt:     10,
			RPCURLs:        []string{"https://mainnet.optimism.io", "https://optimism.llamarpc.com"},
			ExplorerURL:    "https://optimistic.etherscan.io",
			NativeCurrency: "ETH",
			IsTestnet:      false,
		},
		"polygon": {
			Name:           "Polygon",
			ChainID:        big.NewInt(137),
			ChainIDInt:     137,
			RPCURLs:        []string{"https://polygon-rpc.com", "https://polygon.llamarpc.com"},
			ExplorerURL:    "https://polygonscan.com",
			NativeCurrency: "MATIC",
			IsTestnet:      false,
		},
		"sepolia": {
			Name:           "Sepolia Testnet",
			ChainID:        big.NewInt(11155111),
			ChainIDInt:     11155111,
			RPCURLs:        []string{"https://rpc.sepolia.org", "https://sepolia.drpc.org"},
			ExplorerURL:    "https://sepolia.etherscan.io",
			NativeCurrency: "ETH",
			IsTestnet:      true,
		},
		"base-sepolia": {
			Name:           "Base Sepolia Testnet",
			ChainID:        big.NewInt(84532),
			ChainIDInt:     84532,
			RPCURLs:        []string{"https://sepolia.base.org"},
			ExplorerURL:    "https://sepolia.basescan.org",
			NativeCurrency: "ETH",
			IsTestnet:      true,
		},
	}
}
