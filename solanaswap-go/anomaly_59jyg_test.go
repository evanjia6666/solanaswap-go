package solanaswapgo

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func TestAnomaly59JYG_OrcaOKXPoolSystemProgram(t *testing.T) {
	data, err := os.ReadFile("/tmp/tx_59jyg_orca_okx.json")
	if err != nil {
		t.Fatal(err)
	}

	var rpcResp struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		t.Fatal(err)
	}

	var txResult rpc.GetTransactionResult
	if err := json.Unmarshal(rpcResp.Result, &txResult); err != nil {
		t.Fatal(err)
	}

	parser, err := NewParser(&txResult)
	require.NoError(t, err)

	legs, err := parser.ParseTransaction()
	require.NoError(t, err)

	fmt.Printf("\nParsed %d swap legs:\n", len(legs))
	for i, item := range legs {
		if item.Tx == nil {
			fmt.Printf("  Leg %d: nil Tx\n", i)
			continue
		}
		tx := item.Tx
		fmt.Printf("  Leg %d: Type=%s Protocol=%s\n", i, tx.Type, tx.Protocol)
		fmt.Printf("    Pool:      %s\n", tx.Pool)
		fmt.Printf("    PoolIn:    %s\n", tx.PoolIn)
		fmt.Printf("    PoolOut:   %s\n", tx.PoolOut)
		fmt.Printf("    InputMint:  %s (decimals=%d)\n", tx.InputMint.String(), tx.InputMintDecimals)
		fmt.Printf("    InputAmount: %d\n", tx.InputAmount)
		fmt.Printf("    OutputMint:  %s (decimals=%d)\n", tx.OutputMint.String(), tx.OutputMintDecimals)
		fmt.Printf("    OutputAmount: %d\n", tx.OutputAmount)
		fmt.Println()
	}

	require.GreaterOrEqual(t, len(legs), 3, "expected at least 3 swap legs")

	systemProgram := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	leg0 := legs[0].Tx
	require.NotNil(t, leg0)
	require.NotEqual(t, systemProgram, leg0.Pool, "Leg 0 Pool should not be System Program")
	require.NotEqual(t, systemProgram, leg0.PoolIn, "Leg 0 PoolIn should not be System Program")
	require.NotEqual(t, systemProgram, leg0.PoolOut, "Leg 0 PoolOut should not be System Program")

	expectedPool := solana.MustPublicKeyFromBase58("HrR513nHofvYWVrc8AzXuaSpfvnneCXPQccBrnFiAnmz")
	expectedPoolIn := solana.MustPublicKeyFromBase58("A9LDqD7vu4Q2jB5rzbjiNZP9S1FaSyuP3DScpzykLs2K")
	expectedPoolOut := solana.MustPublicKeyFromBase58("3BLfypgszPmQgKTra3tSnL5czfPW6pK7xiQbCmHio7ub")

	require.Equal(t, expectedPool, leg0.Pool, "Leg 0 Pool mismatch")
	require.Equal(t, expectedPoolIn, leg0.PoolIn, "Leg 0 PoolIn mismatch")
	require.Equal(t, expectedPoolOut, leg0.PoolOut, "Leg 0 PoolOut mismatch")

	require.Equal(t, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", leg0.InputMint.String(), "Leg 0 InputMint mismatch")
	require.Equal(t, "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", leg0.OutputMint.String(), "Leg 0 OutputMint mismatch")
}

// TestAnomaly5wwM_DFlowOrcaSwap verifies Whirlpools swap (11 accounts) layout
// via DFlow aggregator (processJupiterSwaps path).
func TestAnomaly5wwM_DFlowOrcaSwap(t *testing.T) {
	data, err := os.ReadFile("/tmp/tx_5wwM_orca_swap.json")
	if err != nil {
		t.Fatal(err)
	}

	var rpcResp struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &rpcResp); err != nil {
		t.Fatal(err)
	}

	var txResult rpc.GetTransactionResult
	if err := json.Unmarshal(rpcResp.Result, &txResult); err != nil {
		t.Fatal(err)
	}

	parser, err := NewParser(&txResult)
	require.NoError(t, err)

	legs, err := parser.ParseTransaction()
	require.NoError(t, err)

	fmt.Printf("\nParsed %d swap legs:\n", len(legs))
	for i, item := range legs {
		if item.Tx == nil {
			fmt.Printf("  Leg %d: nil Tx\n", i)
			continue
		}
		tx := item.Tx
		fmt.Printf("  Leg %d: Type=%s Protocol=%s\n", i, tx.Type, tx.Protocol)
		fmt.Printf("    Pool:      %s\n", tx.Pool)
		fmt.Printf("    PoolIn:    %s\n", tx.PoolIn)
		fmt.Printf("    PoolOut:   %s\n", tx.PoolOut)
		fmt.Printf("    InputMint:  %s (decimals=%d)\n", tx.InputMint.String(), tx.InputMintDecimals)
		fmt.Printf("    InputAmount: %d\n", tx.InputAmount)
		fmt.Printf("    OutputMint:  %s (decimals=%d)\n", tx.OutputMint.String(), tx.OutputMintDecimals)
		fmt.Printf("    OutputAmount: %d\n", tx.OutputAmount)
		fmt.Println()
	}

	require.GreaterOrEqual(t, len(legs), 3, "expected at least 3 swap legs")

	systemProgram := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	// Leg 0: Whirlpools swap (11 accounts) — SOL → wWIF
	leg0 := legs[0].Tx
	require.NotNil(t, leg0)
	require.NotEqual(t, systemProgram, leg0.Pool, "Leg 0 Pool should not be System Program")
	require.NotEqual(t, systemProgram, leg0.PoolIn, "Leg 0 PoolIn should not be System Program")
	require.NotEqual(t, systemProgram, leg0.PoolOut, "Leg 0 PoolOut should not be System Program")
	require.Equal(t, "D6NdKrKNQPmRZCCnG1GqXtF7MMoHB7qR6GU5TkG59Qz1", leg0.Pool.String(), "Leg 0 Pool mismatch")
	require.Equal(t, "76LDmQCyrfqnQ6AUoX3C2HHY67WnG3ED9RQH8UkU5rrk", leg0.PoolIn.String(), "Leg 0 PoolIn mismatch")
	require.Equal(t, "VgwuBBGNCVyT3fnEvRPFhMcYsDN9BH66F6nMQoX9azw", leg0.PoolOut.String(), "Leg 0 PoolOut mismatch")
	require.Equal(t, "So11111111111111111111111111111111111111112", leg0.InputMint.String(), "Leg 0 InputMint mismatch")
	require.Equal(t, "EKpQGSJtjMFqKZ9KQanSqYXRcF8fBopzLHYxdM65zcjm", leg0.OutputMint.String(), "Leg 0 OutputMint mismatch")

	// Leg 1: Whirlpools swap (11 accounts) — wWIF → BONK
	leg1 := legs[1].Tx
	require.NotNil(t, leg1)
	require.NotEqual(t, systemProgram, leg1.Pool, "Leg 1 Pool should not be System Program")
	require.NotEqual(t, systemProgram, leg1.PoolIn, "Leg 1 PoolIn should not be System Program")
	require.NotEqual(t, systemProgram, leg1.PoolOut, "Leg 1 PoolOut should not be System Program")
	require.Equal(t, "3gpyRVcdbH6Jy1AR5Xz5hypNGyGWECLXM5eaA79z4GdQ", leg1.Pool.String(), "Leg 1 Pool mismatch")
	require.Equal(t, "EJcuGkiZ3EbDCikPuWperRDHS4afLPkgz1LF4DpJrLrd", leg1.PoolIn.String(), "Leg 1 PoolIn mismatch")
	require.Equal(t, "5v42ejcpYRE7UcCnC8hifyaa7qiwqKSYepYJtLMHeVhH", leg1.PoolOut.String(), "Leg 1 PoolOut mismatch")
	require.Equal(t, "EKpQGSJtjMFqKZ9KQanSqYXRcF8fBopzLHYxdM65zcjm", leg1.InputMint.String(), "Leg 1 InputMint mismatch")
	require.Equal(t, "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263", leg1.OutputMint.String(), "Leg 1 OutputMint mismatch")

	// Leg 2: ZeroFi — BONK → USDC
	leg2 := legs[2].Tx
	require.NotNil(t, leg2)
	require.NotEqual(t, systemProgram, leg2.Pool, "Leg 2 Pool should not be System Program")
	require.NotEqual(t, systemProgram, leg2.PoolIn, "Leg 2 PoolIn should not be System Program")
	require.NotEqual(t, systemProgram, leg2.PoolOut, "Leg 2 PoolOut should not be System Program")
	require.Equal(t, "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263", leg2.InputMint.String(), "Leg 2 InputMint mismatch")
	require.Equal(t, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", leg2.OutputMint.String(), "Leg 2 OutputMint mismatch")
}
