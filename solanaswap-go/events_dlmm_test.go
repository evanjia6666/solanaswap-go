package solanaswapgo

// Live golden: an OKX-routed DLMM swap on the pump/USDC pool. The deployed
// program emits the legacy Swap and the extended Swap2Evt via anchor
// emit_cpi (self-CPI inner instructions); the extractor must keep exactly
// the Swap2Evt copy with the on-chain amounts.

import (
	"context"
	"math/big"
	"os"
	"testing"

	solana "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

const dlmmGoldenSig = "Jjc1WHzQmVZh4wxzwXnCCMXBei6uBmVJbA5vqLEWDAZ9ieT3PKoxZtcSs9PawuKCBJNw88uqdYioJb6RobvL6Fd"

func TestExtractDLMMEventsLiveGolden(t *testing.T) {
	rpcURL := os.Getenv("SOLANA_RPC_URL")
	if rpcURL == "" {
		t.Skip("SOLANA_RPC_URL not set")
	}
	client := solrpc.New(rpcURL)
	sig := solana.MustSignatureFromBase58(dlmmGoldenSig)
	maxV := uint64(0)
	txr, err := client.GetTransaction(context.TODO(), sig, &solrpc.GetTransactionOpts{
		MaxSupportedTransactionVersion: &maxV,
	})
	if err != nil || txr == nil {
		t.Skipf("tx unavailable: %v", err)
	}
	parser, err := NewParser(txr)
	require.NoError(t, err)

	events := ExtractDLMMEvents(parser.txMeta, parser.allAccountKeys)
	require.NotEmpty(t, events)

	const pool = "Cgk5DWJc59TcWTQ1iaJ8Hn4bsVjKXJF2fCjPAZtMUSkV"
	var swaps, other int
	for _, ev := range events {
		require.Equal(t, METEORA_PROGRAM_ID.String(), ev.Program)
		switch ev.Kind {
		case DLMMEventSwap:
			swaps++
			require.Equal(t, pool, ev.Pool)
			// vault fund flow: pump vault -6,663,768 / USDC vault +297,549 —
			// the user bought 6,663,768 pump with 297,549 USDC
			require.Equal(t, uint64(297549), ev.AmountIn)
			require.Equal(t, uint64(6663768), ev.AmountOut)
			// golden values from the live sample: a single-bin swap (start==end,
			// bin_step keeps small trades inside the active bin), Y in / X out
			// (token_y=USDC, token_x=pump), dynamic-fee snapshot fee_bps in
			// 1e8 units (~2.03%)
			require.Equal(t, int32(-1557), ev.StartBinId)
			require.Equal(t, int32(-1557), ev.EndBinId)
			require.False(t, ev.SwapForY)
			require.NotNil(t, ev.FeeBps)
			require.Equal(t, 0, big.NewInt(2025610).Cmp(ev.FeeBps))
		default:
			other++
		}
	}
	require.Equal(t, 1, swaps, "legacy Swap must be superseded by Swap2Evt exactly once")
	_ = other
}

func TestParseDLMMSwap2EvtShort(t *testing.T) {
	_, err := parseDLMMSwap2Evt(METEORA_PROGRAM_ID, make([]byte, 100))
	require.Error(t, err)
}
