package solanaswapgo

import (
	"encoding/base64"
	"math/big"
	"strings"
	"testing"

	"github.com/gagliardetto/binary"
	solana "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"

	orca_whirlpool "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/orca/orca_whirlpool"
	raydium_clmm "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/raydium/raydium_concentrated_liquidity"
)

func b64Line(disc [8]byte, payload []byte) string {
	raw := append(append([]byte{}, disc[:]...), payload...)
	return "Program data: " + base64.StdEncoding.EncodeToString(raw)
}

func invokeLine(program string) string {
	return "Program " + program + " invoke [2]"
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
		"Program 11111111111111111111111111111111 invoke [1]",
		"Program log: noise",
		invokeLine(raydium_clmm.ProgramID.String()),
		b64Line(raydium_clmm.Event_SwapEvent, body),
		"Program " + raydium_clmm.ProgramID.String() + " success",
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
	if got.SqrtPriceX64Pre != nil {
		t.Fatalf("raydium swap events carry no pre price: %+v", got)
	}
}

// TestExtractCLMMStateEventsForkAttribution: raydium-layout forks emit the
// identical discriminators; the event must be attributed to the invoking
// program, not hardcoded to raydium.
func TestExtractCLMMStateEventsForkAttribution(t *testing.T) {
	ev := raydium_clmm.SwapEvent{
		PoolState:    solana.MustPublicKeyFromBase58("DyzGYEhdgSn5EEUt4XXviavZ7v7SV2YGrYV8HX3Aw5XT"),
		SqrtPriceX64: bin.Uint128{Lo: 5846607886497075687},
		Liquidity:    bin.Uint128{Lo: 468687208968},
		Tick:         -22982,
	}
	body, _ := ev.Marshal()
	logs := []string{
		invokeLine(BYREAL_CLMM_PROGRAM_ID.String()),
		b64Line(raydium_clmm.Event_SwapEvent, body),
	}
	events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: logs})
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if events[0].Program != BYREAL_CLMM_PROGRAM_ID.String() {
		t.Fatalf("fork event must be attributed to the fork program, got %s", events[0].Program)
	}
}

// TestExtractCLMMStateEventsOrcaTraded covers the whirlpool Traded event:
// pre+post sqrt prices, no tick/liquidity.
func TestExtractCLMMStateEventsOrcaTraded(t *testing.T) {
	ev := orca_whirlpool.Traded{
		Whirlpool:     solana.MustPublicKeyFromBase58("6R4r93V5fcMzc13CL2enEepDSYcr4Qx3ptZBDwudTXCo"),
		AToB:          true,
		PreSqrtPrice:  bin.Uint128{Lo: 1111111111111111111},
		PostSqrtPrice: bin.Uint128{Lo: 2222222222222222222},
		InputAmount:   400000000,
		OutputAmount:  3990976628,
	}
	body, err := ev.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	logs := []string{
		"Program JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4 invoke [1]",
		invokeLine(orca_whirlpool.ProgramID.String()),
		"Program log: Instruction: SwapV2",
		b64Line(orca_whirlpool.Event_Traded, body),
		"Program " + orca_whirlpool.ProgramID.String() + " success",
	}
	events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: logs})
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	got := events[0]
	if got.Kind != CLMMEventSwap || got.Program != orca_whirlpool.ProgramID.String() || got.Pool != ev.Whirlpool.String() {
		t.Fatalf("kind/program/pool mismatch: %+v", got)
	}
	if got.SqrtPriceX64.Uint64() != 2222222222222222222 || got.SqrtPriceX64Pre.Uint64() != 1111111111111111111 {
		t.Fatalf("pre/post price mismatch: %+v", got)
	}
	if got.Liquidity != nil || got.Tick != 0 {
		t.Fatalf("traded carries no tick/liquidity: %+v", got)
	}
}

// TestExtractCLMMStateEventsOrcaLiquidity covers both whirlpool liquidity
// events with their signed deltas.
func TestExtractCLMMStateEventsOrcaLiquidity(t *testing.T) {
	inc := orca_whirlpool.LiquidityIncreased{
		Whirlpool:      solana.MustPublicKeyFromBase58("6R4r93V5fcMzc13CL2enEepDSYcr4Qx3ptZBDwudTXCo"),
		TickLowerIndex: -100,
		TickUpperIndex: 100,
		Liquidity:      bin.Uint128{Lo: 777},
	}
	dec := orca_whirlpool.LiquidityDecreased{
		Whirlpool:      inc.Whirlpool,
		TickLowerIndex: -100,
		TickUpperIndex: 100,
		Liquidity:      bin.Uint128{Lo: 300},
	}
	incBody, _ := inc.Marshal()
	decBody, _ := dec.Marshal()
	logs := []string{
		invokeLine(orca_whirlpool.ProgramID.String()),
		b64Line(orca_whirlpool.Event_LiquidityIncreased, incBody),
		b64Line(orca_whirlpool.Event_LiquidityDecreased, decBody),
	}
	events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: logs})
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].DeltaLiquidity.Cmp(big.NewInt(777)) != 0 {
		t.Fatalf("increase delta mismatch: %v", events[0].DeltaLiquidity)
	}
	if events[1].DeltaLiquidity.Cmp(big.NewInt(-300)) != 0 {
		t.Fatalf("decrease delta mismatch: %v", events[1].DeltaLiquidity)
	}
	for _, e := range events {
		if e.TickLower != -100 || e.TickUpper != 100 || e.Program != orca_whirlpool.ProgramID.String() {
			t.Fatalf("bounds/program mismatch: %+v", e)
		}
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
		invokeLine(raydium_clmm.ProgramID.String()),
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
		invokeLine(raydium_clmm.ProgramID.String()),
		b64Line(raydium_clmm.Event_LiquidityChangeEvent, body),
	}})
	if len(events) != 1 || events[0].DeltaLiquidity.Cmp(big.NewInt(-300)) != 0 {
		t.Fatalf("decrease delta mismatch: %+v", events)
	}
}

func TestExtractCLMMStateEventsIgnoresForeignEvents(t *testing.T) {
	// a jupiter-style event discriminator must not be picked up, and events
	// without a known invoking program must not be attributed blindly
	line := "Program data: " + base64.StdEncoding.EncodeToString(make([]byte, 40))
	if strings.Contains(line, "x") {
		t.Fatal("unreachable")
	}
	if events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: []string{line, "Program data: short"}}); len(events) != 0 {
		t.Fatalf("foreign/short events must be ignored, got %d", len(events))
	}
	// raydium discriminator emitted from an unknown program: not attributed
	body, _ := raydium_clmm.SwapEvent{
		PoolState:    solana.MustPublicKeyFromBase58("8sLbNZoA1cfnvMJLPfp98ZLAnFSYCFApfJKMbiXNLwxj"),
		SqrtPriceX64: bin.Uint128{Lo: 1},
		Liquidity:    bin.Uint128{Lo: 1},
	}.Marshal()
	if events := ExtractCLMMStateEvents(&solrpc.TransactionMeta{LogMessages: []string{
		b64Line(raydium_clmm.Event_SwapEvent, body),
	}}); len(events) != 0 {
		t.Fatalf("events without invoke context must be ignored, got %d", len(events))
	}
	if ExtractCLMMStateEvents(nil) != nil {
		t.Fatal("nil meta must return nil")
	}
}

// TestExtractCLMMCreateEvents covers the PoolCreatedEvent extraction: full
// pool identity (mints, vaults, spacing, initial price) attributed to the
// invoking program — raydium plus both forks share the layout.
func TestExtractCLMMCreateEvents(t *testing.T) {
	ev := raydium_clmm.PoolCreatedEvent{
		TokenMint0:   solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
		TokenMint1:   solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"),
		TickSpacing:  10,
		PoolState:    solana.MustPublicKeyFromBase58("49iMatQtoyabsYAQc8GafVq6aeBFVDxSRH44oiatyyw6"),
		SqrtPriceX64: bin.Uint128{Lo: 5452460710584908793},
		Tick:         -24378,
		TokenVault0:  solana.MustPublicKeyFromBase58("8sLbNZoA1cfnvMJLPfp98ZLAnFSYCFApfJKMbiXNLwxj"),
		TokenVault1:  solana.MustPublicKeyFromBase58("7fqohXEWP41Rjwr7Nbo4UQFeDZxJQEmZVup81iFhfMgy"),
	}
	body, err := ev.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	logs := []string{
		"Program 11111111111111111111111111111111 invoke [1]",
		invokeLine(raydium_clmm.ProgramID.String()),
		"Program log: Instruction: CreatePool",
		b64Line(raydium_clmm.Event_PoolCreatedEvent, body),
		"Program " + raydium_clmm.ProgramID.String() + " success",
	}
	events := ExtractCLMMCreateEvents(&solrpc.TransactionMeta{LogMessages: logs})
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	got := events[0]
	if got.Pool != ev.PoolState.String() || got.Program != raydium_clmm.ProgramID.String() {
		t.Fatalf("pool/program mismatch: %+v", got)
	}
	if got.Mint0 != ev.TokenMint0.String() || got.Mint1 != ev.TokenMint1.String() {
		t.Fatalf("mints mismatch: %+v", got)
	}
	if got.Vault0 != ev.TokenVault0.String() || got.Vault1 != ev.TokenVault1.String() {
		t.Fatalf("vaults mismatch: %+v", got)
	}
	if got.TickSpacing != 10 || got.Tick != -24378 || got.SqrtPriceX64.Uint64() != 5452460710584908793 {
		t.Fatalf("spacing/tick/price mismatch: %+v", got)
	}

	// fork attribution: same discriminator emitted by the byreal program
	forkLogs := []string{
		invokeLine(BYREAL_CLMM_PROGRAM_ID.String()),
		b64Line(raydium_clmm.Event_PoolCreatedEvent, body),
	}
	events = ExtractCLMMCreateEvents(&solrpc.TransactionMeta{LogMessages: forkLogs})
	if len(events) != 1 || events[0].Program != BYREAL_CLMM_PROGRAM_ID.String() {
		t.Fatalf("fork creation must be attributed to the fork program: %+v", events)
	}
}
