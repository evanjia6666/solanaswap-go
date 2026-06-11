package solanaswapgo

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

// TestRegression_NoSystemProgramPool verifies that ParseTransaction
// never returns swaps where the pool, inputMint, or outputMint is the
// zero-value public key (SystemProgram address). This guards against
// parsers that leave tx.Pool unset when setTxPoolInfo fails for an
// unknown AMM program ID.
func TestRegression_NoSystemProgramPool(t *testing.T) {
	data, err := os.ReadFile("/tmp/test_buggy_tx.json")
	if err != nil {
		t.Skipf("test data file not found: %v (run: curl ... > /tmp/test_buggy_tx.json)", err)
	}

	var rpcResp struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}
	require.NoError(t, json.Unmarshal(data, &rpcResp))

	var rawResult map[string]interface{}
	require.NoError(t, json.Unmarshal(rpcResp.Result, &rawResult))

	txJSON, _ := json.Marshal(rawResult["transaction"])
	metaJSON, _ := json.Marshal(rawResult["meta"])

	var tx solana.Transaction
	require.NoError(t, json.Unmarshal(txJSON, &tx))

	var meta rpc.TransactionMeta
	require.NoError(t, json.Unmarshal(metaJSON, &meta))

	parser, err := NewTransactionParser(&tx, &meta)
	require.NoError(t, err)

	legs, err := parser.ParseTransaction()
	require.NoError(t, err)

	systemProgram := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	for i, leg := range legs {
		if leg.Tx == nil {
			continue
		}
		require.False(t, leg.Tx.Pool.IsZero(),
			"leg %d pool is zero-value (SystemProgram) — parser left pool unset for unknown AMM", i)
		require.False(t, leg.Tx.Pool.Equals(systemProgram),
			"leg %d pool is SystemProgram — invalid pool address", i)
		require.False(t, leg.Tx.InputMint.IsZero(),
			"leg %d inputMint is zero-value — invalid mint", i)
		require.False(t, leg.Tx.OutputMint.IsZero(),
			"leg %d outputMint is zero-value — invalid mint", i)
	}
}
