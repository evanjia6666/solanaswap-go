package solanaswapgo

import (
	"encoding/base64"
	"math/big"
	"strings"
	"testing"

	"github.com/gagliardetto/binary"
	solana "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"

	raydium_clmm "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/raydium/raydium_concentrated_liquidity"
)

func b64Line(disc [8]byte, payload []byte) string {
	raw := append(append([]byte{}, disc[:]...), payload...)
	return "Program data: " + base64.StdEncoding.EncodeToString(raw)
}

func TestExtractCLMMStateEventsSwap(t *testing.T) {
	ev := raydium_clmm.SwapEvent{
		PoolState:    solana.MustPublicKeyFromBase58("8sLbNZoA1cfnvMJLPfp98ZLAnFSYCFApfJKMbiXNLwxj"),
		SqrtPriceX64: bin.Uint128{Lo: 5452460710584908793},
		Liquidity:    bin.Uint128{Lo: 1332042112266},
		Tick:         -24378,
	}
	body, err := ev.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	logs := []string{
		"Program 11111111111111111111111111111111 invoke [success]",
		b64Line(raydium_clmm.Event_SwapEvent, body),
		"Program 11111111111111111111111111111111 success",
	}
	events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: logs})
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	got := events[0]
	if got.Kind != CLMMEventSwap || got.Pool != ev.PoolState.String() {
		t.Fatalf("kind/pool mismatch: %+v", got)
	}
	if got.Tick != -24378 || got.SqrtPriceX64.Uint64() != 5452460710584908793 || got.Liquidity.Uint64() != 1332042112266 {
		t.Fatalf("state mismatch: %+v", got)
	}
	if got.Program != raydium_clmm.ProgramID.String() {
		t.Fatalf("program mismatch: %s", got.Program)
	}
}

func TestExtractCLMMStateEventsLiquidity(t *testing.T) {
	ev := raydium_clmm.LiquidityChangeEvent{
		PoolState:       solana.MustPublicKeyFromBase58("49iMatQtoyabsYAQc8GafVq6aeBFVDxSRH44oiatyyw6"),
		Tick:            -22641,
		TickLower:       -24420,
		TickUpper:       -22680,
		LiquidityBefore: bin.Uint128{Lo: 1000},
		LiquidityAfter:  bin.Uint128{Lo: 1777},
	}
	body, err := ev.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	logs := []string{
		"Program log: noise",
		b64Line(raydium_clmm.Event_LiquidityChangeEvent, body),
		"Program data: !!!not-base64!!!", // malformed line must be skipped
	}
	events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: logs})
	if len(events) != 1 {
		t.Fatalf("want 1 event (malformed line skipped), got %d", len(events))
	}
	got := events[0]
	if got.Kind != CLMMEventLiquidity {
		t.Fatalf("kind mismatch: %+v", got)
	}
	if got.TickLower != -24420 || got.TickUpper != -22680 || got.Tick != -22641 {
		t.Fatalf("bounds mismatch: %+v", got)
	}
	if got.DeltaLiquidity == nil || got.DeltaLiquidity.Cmp(big.NewInt(777)) != 0 {
		t.Fatalf("delta mismatch: %v", got.DeltaLiquidity)
	}
}

func TestExtractCLMMStateEventsDecrease(t *testing.T) {
	ev := raydium_clmm.LiquidityChangeEvent{
		PoolState:       solana.MustPublicKeyFromBase58("49iMatQtoyabsYAQc8GafVq6aeBFVDxSRH44oiatyyw6"),
		TickLower:       -100,
		TickUpper:       100,
		LiquidityBefore: bin.Uint128{Lo: 500},
		LiquidityAfter:  bin.Uint128{Lo: 200},
	}
	body, _ := ev.Marshal()
	events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: []string{
		b64Line(raydium_clmm.Event_LiquidityChangeEvent, body),
	}})
	if len(events) != 1 || events[0].DeltaLiquidity.Cmp(big.NewInt(-300)) != 0 {
		t.Fatalf("decrease delta mismatch: %+v", events)
	}
}

func TestExtractCLMMStateEventsIgnoresForeignEvents(t *testing.T) {
	// a jupiter-style event discriminator must not be picked up
	line := "Program data: " + base64.StdEncoding.EncodeToString(make([]byte, 40))
	if strings.Contains(line, "x") {
		t.Fatal("unreachable")
	}
	if events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: []string{line, "Program data: short"}}); len(events) != 0 {
		t.Fatalf("foreign/short events must be ignored, got %d", len(events))
	}
	if ExtractCLMMStateEvents(nil) != nil {
		t.Fatal("nil meta must return nil")
	}
}
