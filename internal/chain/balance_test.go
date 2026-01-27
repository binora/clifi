package chain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBalance(t *testing.T) {
	t.Run("nil balance returns zero", func(t *testing.T) {
		result := FormatBalance(nil, 18)
		assert.Equal(t, "0", result)
	})

	t.Run("zero balance", func(t *testing.T) {
		result := FormatBalance(big.NewInt(0), 18)
		assert.Equal(t, "0.000000", result)
	})

	t.Run("1 ETH (18 decimals)", func(t *testing.T) {
		// 1 ETH = 1e18 wei
		oneEth := new(big.Int)
		oneEth.SetString("1000000000000000000", 10)

		result := FormatBalance(oneEth, 18)
		assert.Equal(t, "1.000000", result)
	})

	t.Run("0.5 ETH", func(t *testing.T) {
		halfEth := new(big.Int)
		halfEth.SetString("500000000000000000", 10)

		result := FormatBalance(halfEth, 18)
		assert.Equal(t, "0.500000", result)
	})

	t.Run("large balance (1000 ETH)", func(t *testing.T) {
		largeBalance := new(big.Int)
		largeBalance.SetString("1000000000000000000000", 10) // 1000 ETH

		result := FormatBalance(largeBalance, 18)
		assert.Equal(t, "1000.000000", result)
	})

	t.Run("very small balance", func(t *testing.T) {
		// 1 wei
		result := FormatBalance(big.NewInt(1), 18)
		assert.Equal(t, "0.000000", result) // Too small to display with 6 decimals
	})

	t.Run("6 decimals (USDC)", func(t *testing.T) {
		// 100 USDC = 100 * 1e6
		hundredUsdc := big.NewInt(100000000)

		result := FormatBalance(hundredUsdc, 6)
		assert.Equal(t, "100.000000", result)
	})

	t.Run("2 decimals", func(t *testing.T) {
		// 100 with 2 decimals
		result := FormatBalance(big.NewInt(10000), 2)
		assert.Equal(t, "100.00", result)
	})

	t.Run("0 decimals", func(t *testing.T) {
		result := FormatBalance(big.NewInt(12345), 0)
		assert.Equal(t, "12345", result)
	})

	t.Run("precision capped at 6 for high decimal tokens", func(t *testing.T) {
		// For tokens with >6 decimals, precision should be capped at 6
		balance := new(big.Int)
		balance.SetString("1234567890123456789", 10) // ~1.23 ETH

		result := FormatBalance(balance, 18)
		// Should show 6 decimal places
		assert.Contains(t, result, ".")
		parts := []byte(result)
		decimalPos := -1
		for i, c := range parts {
			if c == '.' {
				decimalPos = i
				break
			}
		}
		if decimalPos >= 0 {
			decimals := len(parts) - decimalPos - 1
			assert.LessOrEqual(t, decimals, 6)
		}
	})
}

func TestDecodeString(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		result := decodeString([]byte{})
		assert.Equal(t, "", result)
	})

	t.Run("short data (non-ABI format)", func(t *testing.T) {
		// Some tokens return raw strings without ABI encoding
		data := []byte("USDC\x00\x00\x00\x00")
		result := decodeString(data)
		assert.Equal(t, "USDC", result)
	})

	t.Run("ABI-encoded string", func(t *testing.T) {
		// Standard ABI encoding: offset (32) + length (32) + data
		// This encodes "TEST"
		data := make([]byte, 96)
		// Offset (32) - points to byte 32
		data[31] = 32
		// Length (4)
		data[63] = 4
		// Data "TEST"
		copy(data[64:], []byte("TEST"))

		result := decodeString(data)
		assert.Equal(t, "TEST", result)
	})

	t.Run("null terminated string", func(t *testing.T) {
		data := []byte("WETH\x00\x00\x00\x00\x00\x00\x00\x00")
		result := decodeString(data)
		assert.Equal(t, "WETH", result)
	})

	t.Run("handles length overflow", func(t *testing.T) {
		// Create data where length field is larger than remaining data
		data := make([]byte, 64)
		// Offset
		data[31] = 32
		// Length (255 - way more than available)
		data[63] = 255

		result := decodeString(data)
		assert.Equal(t, "", result) // Should return empty for invalid data
	})

	t.Run("handles zero length", func(t *testing.T) {
		data := make([]byte, 64)
		// Offset
		data[31] = 32
		// Length (0)
		data[63] = 0

		result := decodeString(data)
		assert.Equal(t, "", result)
	})
}

func TestTokenBalance_Structure(t *testing.T) {
	t.Run("can create TokenBalance", func(t *testing.T) {
		tb := TokenBalance{
			TokenAddress: "0x1234567890123456789012345678901234567890",
			Symbol:       "TEST",
			Name:         "Test Token",
			Balance:      big.NewInt(1000000),
			Decimals:     18,
		}

		assert.Equal(t, "TEST", tb.Symbol)
		assert.Equal(t, "Test Token", tb.Name)
		assert.Equal(t, uint8(18), tb.Decimals)
	})
}

func TestNativeBalance_Structure(t *testing.T) {
	t.Run("can create NativeBalance", func(t *testing.T) {
		nb := NativeBalance{
			Chain:    "ethereum",
			Symbol:   "ETH",
			Balance:  big.NewInt(1000000000000000000),
			Decimals: 18,
		}

		assert.Equal(t, "ethereum", nb.Chain)
		assert.Equal(t, "ETH", nb.Symbol)
		assert.Equal(t, uint8(18), nb.Decimals)
	})
}

func TestPortfolio_Structure(t *testing.T) {
	t.Run("can create Portfolio", func(t *testing.T) {
		p := Portfolio{
			Address:        "0xabcd",
			NativeBalances: make(map[string]*NativeBalance),
			TokenBalances:  make(map[string][]*TokenBalance),
		}

		p.NativeBalances["ethereum"] = &NativeBalance{
			Chain:   "ethereum",
			Symbol:  "ETH",
			Balance: big.NewInt(1000),
		}

		assert.Equal(t, "0xabcd", p.Address)
		assert.Len(t, p.NativeBalances, 1)
	})
}
