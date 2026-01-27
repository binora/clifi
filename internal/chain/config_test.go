package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultChains(t *testing.T) {
	chains := DefaultChains()

	t.Run("returns all expected chains", func(t *testing.T) {
		expectedChains := []string{
			"ethereum",
			"base",
			"arbitrum",
			"optimism",
			"polygon",
			"sepolia",
			"base-sepolia",
		}

		assert.Len(t, chains, len(expectedChains))
		for _, name := range expectedChains {
			_, ok := chains[name]
			assert.True(t, ok, "missing chain: %s", name)
		}
	})

	t.Run("ethereum config is correct", func(t *testing.T) {
		eth := chains["ethereum"]
		require.NotNil(t, eth)

		assert.Equal(t, "Ethereum Mainnet", eth.Name)
		assert.Equal(t, int64(1), eth.ChainID.Int64())
		assert.Equal(t, int64(1), eth.ChainIDInt)
		assert.NotEmpty(t, eth.RPCURLs)
		assert.Equal(t, "https://etherscan.io", eth.ExplorerURL)
		assert.Equal(t, "ETH", eth.NativeCurrency)
		assert.False(t, eth.IsTestnet)
	})

	t.Run("base config is correct", func(t *testing.T) {
		base := chains["base"]
		require.NotNil(t, base)

		assert.Equal(t, "Base", base.Name)
		assert.Equal(t, int64(8453), base.ChainID.Int64())
		assert.Equal(t, "ETH", base.NativeCurrency)
		assert.False(t, base.IsTestnet)
	})

	t.Run("arbitrum config is correct", func(t *testing.T) {
		arb := chains["arbitrum"]
		require.NotNil(t, arb)

		assert.Equal(t, "Arbitrum One", arb.Name)
		assert.Equal(t, int64(42161), arb.ChainID.Int64())
		assert.Equal(t, "ETH", arb.NativeCurrency)
		assert.False(t, arb.IsTestnet)
	})

	t.Run("optimism config is correct", func(t *testing.T) {
		op := chains["optimism"]
		require.NotNil(t, op)

		assert.Equal(t, "Optimism", op.Name)
		assert.Equal(t, int64(10), op.ChainID.Int64())
		assert.Equal(t, "ETH", op.NativeCurrency)
		assert.False(t, op.IsTestnet)
	})

	t.Run("polygon config is correct", func(t *testing.T) {
		poly := chains["polygon"]
		require.NotNil(t, poly)

		assert.Equal(t, "Polygon", poly.Name)
		assert.Equal(t, int64(137), poly.ChainID.Int64())
		assert.Equal(t, "MATIC", poly.NativeCurrency)
		assert.False(t, poly.IsTestnet)
	})

	t.Run("sepolia testnet config is correct", func(t *testing.T) {
		sepolia := chains["sepolia"]
		require.NotNil(t, sepolia)

		assert.Equal(t, "Sepolia Testnet", sepolia.Name)
		assert.Equal(t, int64(11155111), sepolia.ChainID.Int64())
		assert.Equal(t, "ETH", sepolia.NativeCurrency)
		assert.True(t, sepolia.IsTestnet)
	})

	t.Run("base-sepolia testnet config is correct", func(t *testing.T) {
		baseSepolia := chains["base-sepolia"]
		require.NotNil(t, baseSepolia)

		assert.Equal(t, "Base Sepolia Testnet", baseSepolia.Name)
		assert.Equal(t, int64(84532), baseSepolia.ChainID.Int64())
		assert.Equal(t, "ETH", baseSepolia.NativeCurrency)
		assert.True(t, baseSepolia.IsTestnet)
	})

	t.Run("all chains have RPC URLs", func(t *testing.T) {
		for name, config := range chains {
			assert.NotEmpty(t, config.RPCURLs, "chain %s has no RPC URLs", name)
		}
	})

	t.Run("all chains have explorer URL", func(t *testing.T) {
		for name, config := range chains {
			assert.NotEmpty(t, config.ExplorerURL, "chain %s has no explorer URL", name)
		}
	})

	t.Run("chainID matches chainIDInt", func(t *testing.T) {
		for name, config := range chains {
			assert.Equal(t, config.ChainIDInt, config.ChainID.Int64(),
				"chain %s: ChainID and ChainIDInt mismatch", name)
		}
	})
}
