package solanaswapgo

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
)

// TestMeteoraLPLegGolden replays the mainnet DAMM v2 add_liquidity
// transaction reported as missing liquidity coverage (outer add_liquidity +
// one self-CPI event, no swap): the parser must now produce one add leg
// with the pool, both vaults, both mints and the deposited amounts.
func TestMeteoraLPLegGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata_lp_dammv2_add.json")
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	var res struct {
		Result *rpc.GetTransactionResult `json:"result"`
	}
	if err := json.Unmarshal(raw, &res); err != nil || res.Result == nil {
		t.Skipf("fixture undecodable: %v", err)
	}

	parser, err := NewParser(res.Result)
	if err != nil {
		t.Fatal(err)
	}
	items, err := parser.ParseTransaction()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Tx == nil {
		t.Fatalf("want exactly 1 leg, got %d", len(items))
	}
	tx := items[0].Tx
	if tx.Type != TxTypeAdd {
		t.Fatalf("type = %s, want %s", tx.Type, TxTypeAdd)
	}
	if tx.Pool.String() != "C2U19LDRdMyV4PXro9shK4JNLmbmjeUmFFqtgtgoq7A3" {
		t.Fatalf("pool = %s", tx.Pool)
	}
	if tx.Amm.String() != "cpamdpZCGKUy5JxQXB4dcpGPiikHawvSWAd6mEn1sGG" {
		t.Fatalf("amm = %s", tx.Amm)
	}
	if tx.PoolIn.String() != "223SMzmyaPXMHzRnLNs88ohC4JFkQUEDxzzcXdratYtX" ||
		tx.PoolOut.String() != "34iM67wyaaXKpcTHkfS73DC2x7DSTYzg5ZQwTpEPKSuH" {
		t.Fatalf("vaults = %s / %s", tx.PoolIn, tx.PoolOut)
	}
	if tx.InputMint.String() != "AbKaeNaS8FeyykruJ7sAT9pYyeAHSE4bni9RxirrZxMT" ||
		tx.OutputMint.String() != "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263" {
		t.Fatalf("mints = %s / %s", tx.InputMint, tx.OutputMint)
	}
	if tx.InputAmount != 3379280 || tx.OutputAmount != 3956 {
		t.Fatalf("amounts = %d / %d, want 3379280 / 3956", tx.InputAmount, tx.OutputAmount)
	}
	if tx.PoolInAmount == nil || tx.PoolInAmount.Uint64() != 291559094680170064 {
		t.Fatalf("poolIn snapshot = %v", tx.PoolInAmount)
	}
}
