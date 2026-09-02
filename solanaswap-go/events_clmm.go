package solanaswapgo

import (
	"encoding/base64"
	"math/big"
	"strings"

	solrpc "github.com/gagliardetto/solana-go/rpc"

	raydium_clmm "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/raydium/raydium_concentrated_liquidity"
)

// CLMMStateEvent is a normalized concentrated-liquidity state event used to
// maintain pool state locally (no RPC).
//
// Raydium CLMM emits anchor events via `Program data:` log lines:
//   - SwapEvent carries the full post-swap pool state (sqrt price Q64.64,
//     active liquidity, current tick) — enough to keep Reserve fresh.
//   - LiquidityChangeEvent carries the position bounds and the liquidity
//     before/after — enough to apply tick deltas locally and re-pin the
//     active liquidity against drift.
//
// Orca Whirlpool (the older whirLb program) emits no events and its
// liquidity instructions do not carry tick bounds (they live in the
// position account), so Orca state stays on the RPC refresh path and is
// not represented here.
type CLMMStateEvent struct {
	// Program is the on-chain AMM program id; Pool the pool account.
	Program string
	Pool    string

	// Kind: CLMMEventSwap or CLMMEventLiquidity.
	Kind string

	// Swap fields (post-swap state; Q64.64 sqrt price, raw liquidity).
	SqrtPriceX64 *big.Int
	Liquidity    *big.Int
	Tick         int32

	// Liquidity fields.
	TickLower      int32
	TickUpper      int32
	DeltaLiquidity *big.Int // signed: + increase, - decrease
}

// CLMM event kinds.
const (
	CLMMEventSwap      = "swap"
	CLMMEventLiquidity = "liquidity"
)

// anchorEventDataPrefix is the log marker anchor emits for event data.
const anchorEventDataPrefix = "Program data: "

// ExtractCLMMStateEvents parses Raydium CLMM swap/liquidity events from a
// transaction's log lines, in emission order.
func ExtractCLMMStateEvents(meta *solrpc.TransactionMeta) []CLMMStateEvent {
	if meta == nil {
		return nil
	}
	var out []CLMMStateEvent
	for _, line := range meta.LogMessages {
		if !strings.HasPrefix(line, anchorEventDataPrefix) {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, anchorEventDataPrefix))
		if err != nil || len(raw) < 8 {
			continue
		}
		var disc [8]byte
		copy(disc[:], raw[:8])
		switch disc {
		case raydium_clmm.Event_SwapEvent:
			ev, err := raydium_clmm.ParseEvent_SwapEvent(raw)
			if err != nil {
				continue
			}
			out = append(out, CLMMStateEvent{
				Program:      raydium_clmm.ProgramID.String(),
				Pool:         ev.PoolState.String(),
				Kind:         CLMMEventSwap,
				SqrtPriceX64: ev.SqrtPriceX64.BigInt(),
				Liquidity:    ev.Liquidity.BigInt(),
				Tick:         ev.Tick,
			})
		case raydium_clmm.Event_LiquidityChangeEvent:
			ev, err := raydium_clmm.ParseEvent_LiquidityChangeEvent(raw)
			if err != nil {
				continue
			}
			delta := new(big.Int).Sub(ev.LiquidityAfter.BigInt(), ev.LiquidityBefore.BigInt())
			out = append(out, CLMMStateEvent{
				Program:        raydium_clmm.ProgramID.String(),
				Pool:           ev.PoolState.String(),
				Kind:           CLMMEventLiquidity,
				Tick:           ev.Tick,
				TickLower:      ev.TickLower,
				TickUpper:      ev.TickUpper,
				DeltaLiquidity: delta,
			})
		}
	}
	return out
}
