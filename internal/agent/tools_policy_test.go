package agent

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/tx"
)

func TestLoadPolicy_ParsesEnv(t *testing.T) {
	t.Setenv("CLIFI_MAX_TX_ETH", "0.5")
	t.Setenv("CLIFI_ALLOW_TO", "0x1111111111111111111111111111111111111111, 0x2222222222222222222222222222222222222222")
	t.Setenv("CLIFI_DENY_TO", "0x3333333333333333333333333333333333333333")

	p := loadPolicy()

	require.NotNil(t, p.MaxPerTxWei)
	assert.Equal(t, common.HexToAddress("0x1111111111111111111111111111111111111111"), p.AllowTo[0])
	assert.Equal(t, common.HexToAddress("0x2222222222222222222222222222222222222222"), p.AllowTo[1])
	assert.Equal(t, common.HexToAddress("0x3333333333333333333333333333333333333333"), p.DenyTo[0])
}

func TestValidatePolicy(t *testing.T) {
	intent := tx.Intent{
		Chain:    "ethereum",
		From:     common.HexToAddress("0x0000000000000000000000000000000000000001"),
		To:       common.HexToAddress("0x1111111111111111111111111111111111111111"),
		ValueWei: common.Big1,
	}

	p := tx.Policy{
		MaxPerTxWei: common.Big1,
		AllowTo:     []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")},
	}
	assert.NoError(t, tx.Validate(intent, p))

	p.MaxPerTxWei = common.Big0
	assert.Error(t, tx.Validate(intent, p))
}

func TestDecimalToWei(t *testing.T) {
	v, err := decimalToWei("1.5", 6)
	require.NoError(t, err)
	assert.Equal(t, "1500000", v.String())

	_, err = decimalToWei("notnum", 18)
	assert.Error(t, err)
}
