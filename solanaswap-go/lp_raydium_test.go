package solanaswapgo

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
)

// loadLPTxFixture loads a mainnet getTransaction fixture captured for the
// liquidity-leg regression suite.
func loadLPTxFixture(t *testing.T, name string) *Parser {
	t.Helper()
	raw, err := os.ReadFile(name)
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
	return parser
}

// wantLPLeg asserts items contains exactly one leg with the given type /
// amm / pool and returns it.
func wantLPLeg(t *testing.T, items []SwapData, typ, amm, pool string) *TxInfo {
	t.Helper()
	if len(items) != 1 || items[0].Tx == nil {
		t.Fatalf("want exactly 1 leg, got %d", len(items))
	}
	tx := items[0].Tx
	if tx.Type != typ {
		t.Fatalf("type = %s, want %s", tx.Type, typ)
	}
	if tx.Amm.String() != amm {
		t.Fatalf("amm = %s, want %s", tx.Amm, amm)
	}
	if tx.Pool.String() != pool {
		t.Fatalf("pool = %s, want %s", tx.Pool, pool)
	}
	return tx
}

// TestRaydiumCPMMWithdrawUnderWrapperGolden replays mainnet CPMM withdraws
// that only exist as CPI inner instructions under wrapper programs with no
// registered parser (the CPMM lock program LockrWmn6…, the Jupiter trigger
// router BLiaZWNQ…). Before the sweep pass these produced zero legs: the
// dispatch was driven by top-level programs only, so the withdraw leg — and
// its fundmap/remove classification and reserve refresh — was silently lost.
// Expected values are extracted from the raw instruction bytes (pool@2,
// vaults@6/7 per the CPMM remove layout) and the vault→user transferChecked
// amounts, not from parser output.
func TestRaydiumCPMMWithdrawUnderWrapperGolden(t *testing.T) {
	cases := []struct {
		name      string
		fixture   string
		pool      string
		vaultA    string
		vaultB    string
		amountIn  uint64
		amountOut uint64
	}{
		{
			name:      "jupiter-trigger-wrapper",
			fixture:   "testdata_lp_cpmm_withdraw_bliaz.json",
			pool:      "82BHvVSeHgcGNtiH8fGTaYRsKbbjF53beAZp8LCATXsc",
			vaultA:    "55xGcZnYkAx89QEfgTG3HWHkrdw9QhsVFTC4igtesCzY",
			vaultB:    "DFmCteqnLs5jBYgzeKgQxcwGZXQDmKxG1umtTm6ymYUg",
			amountIn:  3227651095,
			amountOut: 82225458793,
		},
		{
			name:      "lock-program-top-level",
			fixture:   "testdata_lp_cpmm_withdraw_lockr_top.json",
			pool:      "CheJUxVQrmpeb4UFWHZ72afQPTKM7AEb8giyzNR7cPvH",
			vaultA:    "ECidSvZMiRyM6EqEneTtzRC1Asc4pTrohzwSHkEyjQru",
			vaultB:    "4NXE27YdcqkgaMKbw8eugpYNJmiyC9tC6YTPQmbmZ4cY",
			amountIn:  66893701,
			amountOut: 40653952955,
		},
		{
			name:      "jupiter-trigger-wrapper-minimal",
			fixture:   "testdata_lp_cpmm_withdraw_bliaz_min.json",
			pool:      "82BHvVSeHgcGNtiH8fGTaYRsKbbjF53beAZp8LCATXsc",
			vaultA:    "55xGcZnYkAx89QEfgTG3HWHkrdw9QhsVFTC4igtesCzY",
			vaultB:    "DFmCteqnLs5jBYgzeKgQxcwGZXQDmKxG1umtTm6ymYUg",
			amountIn:  7091044033,
			amountOut: 173101966273,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := loadLPTxFixture(t, tc.fixture)
			items, err := parser.ParseTransaction()
			if err != nil {
				t.Fatal(err)
			}
			tx := wantLPLeg(t, items, TxTypeRemove,
				"CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C", tc.pool)
			if tx.InputAmount != tc.amountIn || tx.OutputAmount != tc.amountOut {
				t.Fatalf("amounts = %d / %d, want %d / %d",
					tx.InputAmount, tx.OutputAmount, tc.amountIn, tc.amountOut)
			}
			// The layout may orient PoolIn/PoolOut to match the transfer
			// order; the vault pair must be exactly the instruction's two
			// vault accounts.
			if !(tx.PoolIn.String() == tc.vaultA && tx.PoolOut.String() == tc.vaultB) &&
				!(tx.PoolIn.String() == tc.vaultB && tx.PoolOut.String() == tc.vaultA) {
				t.Fatalf("vaults = %s / %s, want %s / %s",
					tx.PoolIn, tx.PoolOut, tc.vaultA, tc.vaultB)
			}
		})
	}
}

// TestRaydiumCPMMDirectLPGolden pins the already-working direct (top-level)
// CPMM deposit/withdraw path so the sweep pass cannot regress it.
func TestRaydiumCPMMDirectLPGolden(t *testing.T) {
	t.Run("direct-withdraw", func(t *testing.T) {
		parser := loadLPTxFixture(t, "testdata_lp_cpmm_withdraw_direct.json")
		items, err := parser.ParseTransaction()
		if err != nil {
			t.Fatal(err)
		}
		tx := wantLPLeg(t, items, TxTypeRemove,
			"CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C",
			"6ZQFf18dqiiCczZ9jbQga6CP4T8SFpjeAq7Cj4JToEdw")
		if tx.InputMint.String() != "So11111111111111111111111111111111111111112" ||
			tx.OutputMint.String() != "73bYJWJsxvxAmZTZYm1SEyxYCiUnE5qr7Qy6UZzgPgFJ" {
			t.Fatalf("mints = %s / %s", tx.InputMint, tx.OutputMint)
		}
		if tx.InputAmount != 2602717655 || tx.OutputAmount != 899076645575682 {
			t.Fatalf("amounts = %d / %d, want 2602717655 / 899076645575682",
				tx.InputAmount, tx.OutputAmount)
		}
		if tx.PoolIn.String() != "LzuFCrsWsGV4xyupn3sh7gAQLJxrNDGUSdryZhb2CWE" ||
			tx.PoolOut.String() != "BCDcECLfQ39dhxP1STPFPdzvw796guUwoiH8b4r9wFRB" {
			t.Fatalf("vaults = %s / %s", tx.PoolIn, tx.PoolOut)
		}
	})
	t.Run("direct-deposit", func(t *testing.T) {
		parser := loadLPTxFixture(t, "testdata_lp_cpmm_deposit_direct.json")
		items, err := parser.ParseTransaction()
		if err != nil {
			t.Fatal(err)
		}
		tx := wantLPLeg(t, items, TxTypeAdd,
			"CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C",
			"7vW6cmvM2YYHzoLTx7qJqACzj3X2Rq236b83YHpqCbyD")
		if tx.InputMint.String() != "So11111111111111111111111111111111111111112" ||
			tx.OutputMint.String() != "Dw9xf7EmMH5dD7rdqkFbzjJtcAWk4KXLBAXUkSRcLSLi" {
			t.Fatalf("mints = %s / %s", tx.InputMint, tx.OutputMint)
		}
		if tx.InputAmount != 4111414 || tx.OutputAmount != 974999999991809 {
			t.Fatalf("amounts = %d / %d, want 4111414 / 974999999991809",
				tx.InputAmount, tx.OutputAmount)
		}
	})
}
