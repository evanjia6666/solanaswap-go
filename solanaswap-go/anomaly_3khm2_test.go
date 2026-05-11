package solanaswapgo

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestAnomaly3KHM2(t *testing.T) {
	data, err := os.ReadFile("/tmp/anomaly_3khm_tx.json")
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

	var rawResult map[string]interface{}
	if err := json.Unmarshal(rpcResp.Result, &rawResult); err != nil {
		t.Fatal(err)
	}

	txJSON, _ := json.Marshal(rawResult["transaction"])
	metaJSON, _ := json.Marshal(rawResult["meta"])

	var tx solana.Transaction
	if err := json.Unmarshal(txJSON, &tx); err != nil {
		t.Fatal(err)
	}

	var meta rpc.TransactionMeta
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		t.Fatal(err)
	}

	// Print PostTokenBalances
	fmt.Println("=== PostTokenBalances ===")
	for _, tb := range meta.PostTokenBalances {
		mint := tb.Mint.String()
		if mint == "3KHMZhpthXuiCcgfTv7vVu9PpEz64KAEURFwi6Lopump" || mint == "So11111111111111111111111111111111111111112" {
			fmt.Printf("  Owner: %s\n", tb.Owner.String())
			fmt.Printf("  Mint: %s\n", mint)
			fmt.Printf("  Decimals: %d\n", tb.UiTokenAmount.Decimals)
			fmt.Printf("  Amount: %s\n", tb.UiTokenAmount.Amount)
			fmt.Println()
		}
	}

	parser, err := NewTransactionParser(&tx, &meta)
	if err != nil {
		t.Fatal(err)
	}

	// Print splDecimalsMap
	fmt.Println("=== splDecimalsMap ===")
	for mint, decimals := range parser.splDecimalsMap {
		if mint == "3KHMZhpthXuiCcgfTv7vVu9PpEz64KAEURFwi6Lopump" || mint == "So11111111111111111111111111111111111111112" {
			fmt.Printf("  %s: %d\n", mint, decimals)
		}
	}
}
