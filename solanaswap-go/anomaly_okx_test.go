package solanaswapgo

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestAnomalyOKX(t *testing.T) {
	data, err := os.ReadFile("/tmp/okx_abnormal_tx_wrapped.json")
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

	fmt.Printf("Transaction signatures: %v\n", tx.Signatures)

	parser, err := NewTransactionParser(&tx, &meta)
	if err != nil {
		t.Fatal(err)
	}

	transactionData, err := parser.ParseTransaction()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("\nParsed %d swap legs:\n", len(transactionData))
	for i, item := range transactionData {
		if item.Tx == nil {
			fmt.Printf("  Leg %d: nil Tx\n", i)
			continue
		}
		tx := item.Tx
		fmt.Printf("  Leg %d: Type=%s Protocol=%s\n", i, tx.Type, tx.Protocol)
		fmt.Printf("    InputMint:  %s (decimals=%d)\n", tx.InputMint.String(), tx.InputMintDecimals)
		fmt.Printf("    InputAmount: %d\n", tx.InputAmount)
		fmt.Printf("    OutputMint:  %s (decimals=%d)\n", tx.OutputMint.String(), tx.OutputMintDecimals)
		fmt.Printf("    OutputAmount: %d\n", tx.OutputAmount)
		fmt.Println()
	}

	swapInfo, txInfo, err := parser.ProcessSwapData(transactionData)
	if err != nil {
		fmt.Printf("ProcessSwapData error: %v\n", err)
		return
	}

	fmt.Println("=== ProcessSwapData Result ===")
	fmt.Printf("TokenInMint:  %s (decimals=%d)\n", swapInfo.TokenInMint.String(), swapInfo.TokenInDecimals)
	fmt.Printf("TokenInAmount: %d\n", swapInfo.TokenInAmount)
	fmt.Printf("TokenOutMint:  %s (decimals=%d)\n", swapInfo.TokenOutMint.String(), swapInfo.TokenOutDecimals)
	fmt.Printf("TokenOutAmount: %d\n", swapInfo.TokenOutAmount)

	if swapInfo.TokenInDecimals > 0 && swapInfo.TokenOutDecimals > 0 {
		inputDec := float64(swapInfo.TokenInAmount) / float64(pow10(swapInfo.TokenInDecimals))
		outputDec := float64(swapInfo.TokenOutAmount) / float64(pow10(swapInfo.TokenOutDecimals))
		if inputDec > 0 {
			price := outputDec / inputDec
			fmt.Printf("Final price: %.8f (in/out ratio)\n", price)
		}
	}

	if txInfo != nil {
		fmt.Printf("\nTxInfo InputMint:  %s\n", txInfo.InputMint.String())
		fmt.Printf("TxInfo InputAmount: %d\n", txInfo.InputAmount)
		fmt.Printf("TxInfo OutputMint:  %s\n", txInfo.OutputMint.String())
		fmt.Printf("TxInfo OutputAmount: %d\n", txInfo.OutputAmount)
	}
}
