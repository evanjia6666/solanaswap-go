package solanaswapgo

import (
	"encoding/base64"
	"math/big"
	"strings"

	solana "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"

	orca_whirlpool "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/orca/orca_whirlpool"
	raydium_clmm "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/raydium/raydium_concentrated_liquidity"
)

// CLMMStateEvent is a normalized concentrated-liquidity state event used to
// maintain pool state locally (no RPC).
//
// Raydium-layout programs (Raydium CLMM and its forks Byreal/PancakeSwap)
// emit anchor events via `Program data:` log lines:
//   - SwapEvent carries the full post-swap pool state (sqrt price Q64.64,
//     active liquidity, current tick) — enough to keep Reserve fresh.
//   - LiquidityChangeEvent carries the position bounds and the liquidity
//     before/after — enough to apply tick deltas locally and re-pin the
//     active liquidity against drift.
//
// Orca Whirlpool emits Traded with the pre AND post sqrt prices (no
// tick/liquidity — the dex side derives the tick from the post price and
// keeps the stored liquidity until the next snapshot refresh), and
// LiquidityIncreased/LiquidityDecreased with the position bounds and the
// position's liquidity delta.
//
// Events are attributed to the emitting program by tracking the log's
// program call stack, so raydium-layout forks get their own program id.
type CLMMStateEvent struct {
	// Program is the on-chain AMM program id; Pool the pool account.
	Program string
	Pool    string

	// Kind: CLMMEventSwap or CLMMEventLiquidity.
	Kind string

	// Swap fields (post-swap state; Q64.64 sqrt price, raw liquidity).
	SqrtPriceX64 *big.Int
	// SqrtPriceX64Pre is the pre-swap sqrt price (Q64.64). Orca's Traded
	// carries it; raydium-layout SwapEvents leave it nil.
	SqrtPriceX64Pre *big.Int
	Liquidity       *big.Int
	Tick            int32

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

// ExtractCLMMStateEvents parses CLMM swap/liquidity events from a
// transaction's log lines, in emission order. Each event is attributed to
// the program that emitted it: "Program data:" lines belong to the
// innermost still-executing program (an anchor event is emitted after the
// instruction's own CPIs returned, so the last invoke alone is not enough —
// a call stack is tracked instead).
func ExtractCLMMStateEvents(meta *solrpc.TransactionMeta) []CLMMStateEvent {
	if meta == nil {
		return nil
	}
	var out []CLMMStateEvent
	var stack []solana.PublicKey
	current := func() solana.PublicKey {
		if len(stack) == 0 {
			return solana.PublicKey{}
		}
		return stack[len(stack)-1]
	}
	for _, line := range meta.LogMessages {
		if strings.HasPrefix(line, anchorEventDataPrefix) {
			currentProgram := current()
			raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, anchorEventDataPrefix))
			if err != nil || len(raw) < 8 {
				continue
			}
			if ev, ok := parseCLMMEvent(currentProgram, raw); ok {
				out = append(out, ev)
			}
			continue
		}
		// remaining "Program ..." lifecycle lines maintain the call stack
		if !strings.HasPrefix(line, "Program ") {
			continue
		}
		if idx := strings.Index(line, " invoke"); idx > len("Program ") {
			if id, err := solana.PublicKeyFromBase58(line[len("Program "):idx]); err == nil {
				stack = append(stack, id)
			}
			continue
		}
		if strings.HasSuffix(line, " success") || strings.HasSuffix(line, " failed") {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return out
}

// parseCLMMEvent decodes one anchor event payload for the emitting program.
func parseCLMMEvent(program solana.PublicKey, raw []byte) (CLMMStateEvent, bool) {
	var disc [8]byte
	copy(disc[:], raw[:8])
	switch disc {
	case raydium_clmm.Event_SwapEvent:
		if !isRaydiumLayoutProgram(program.String()) {
			return CLMMStateEvent{}, false
		}
		ev, err := raydium_clmm.ParseEvent_SwapEvent(raw)
		if err != nil {
			return CLMMStateEvent{}, false
		}
		return CLMMStateEvent{
			Program:      program.String(),
			Pool:         ev.PoolState.String(),
			Kind:         CLMMEventSwap,
			SqrtPriceX64: ev.SqrtPriceX64.BigInt(),
			Liquidity:    ev.Liquidity.BigInt(),
			Tick:         ev.Tick,
		}, true
	case raydium_clmm.Event_LiquidityChangeEvent:
		if !isRaydiumLayoutProgram(program.String()) {
			return CLMMStateEvent{}, false
		}
		ev, err := raydium_clmm.ParseEvent_LiquidityChangeEvent(raw)
		if err != nil {
			return CLMMStateEvent{}, false
		}
		delta := new(big.Int).Sub(ev.LiquidityAfter.BigInt(), ev.LiquidityBefore.BigInt())
		return CLMMStateEvent{
			Program:        program.String(),
			Pool:           ev.PoolState.String(),
			Kind:           CLMMEventLiquidity,
			Tick:           ev.Tick,
			TickLower:      ev.TickLower,
			TickUpper:      ev.TickUpper,
			DeltaLiquidity: delta,
		}, true
	case orca_whirlpool.Event_Traded:
		if !program.Equals(orca_whirlpool.ProgramID) {
			return CLMMStateEvent{}, false
		}
		ev, err := orca_whirlpool.ParseEvent_Traded(raw)
		if err != nil {
			return CLMMStateEvent{}, false
		}
		return CLMMStateEvent{
			Program:         program.String(),
			Pool:            ev.Whirlpool.String(),
			Kind:            CLMMEventSwap,
			SqrtPriceX64:    ev.PostSqrtPrice.BigInt(),
			SqrtPriceX64Pre: ev.PreSqrtPrice.BigInt(),
			// Traded carries no tick/liquidity: the dex side derives the
			// tick from the post price and keeps the stored liquidity
			// until the next snapshot refresh.
		}, true
	case orca_whirlpool.Event_LiquidityIncreased, orca_whirlpool.Event_LiquidityDecreased:
		if !program.Equals(orca_whirlpool.ProgramID) {
			return CLMMStateEvent{}, false
		}
		var pool string
		var tickLower, tickUpper int32
		var delta *big.Int
		if disc == orca_whirlpool.Event_LiquidityIncreased {
			ev, err := orca_whirlpool.ParseEvent_LiquidityIncreased(raw)
			if err != nil {
				return CLMMStateEvent{}, false
			}
			pool, tickLower, tickUpper = ev.Whirlpool.String(), ev.TickLowerIndex, ev.TickUpperIndex
			delta = ev.Liquidity.BigInt()
		} else {
			ev, err := orca_whirlpool.ParseEvent_LiquidityDecreased(raw)
			if err != nil {
				return CLMMStateEvent{}, false
			}
			pool, tickLower, tickUpper = ev.Whirlpool.String(), ev.TickLowerIndex, ev.TickUpperIndex
			delta = new(big.Int).Neg(ev.Liquidity.BigInt())
		}
		return CLMMStateEvent{
			Program:        program.String(),
			Pool:           pool,
			Kind:           CLMMEventLiquidity,
			TickLower:      tickLower,
			TickUpper:      tickUpper,
			DeltaLiquidity: delta,
		}, true
	}
	return CLMMStateEvent{}, false
}

// isRaydiumLayoutProgram reports whether the program id shares the raydium
// CLMM event/account layouts: Raydium itself plus its forks (Byreal,
// PancakeSwap CLMM), whose anchor discriminators are identical because the
// event names are.
func isRaydiumLayoutProgram(program string) bool {
	switch program {
	case raydium_clmm.ProgramID.String(),
		BYREAL_CLMM_PROGRAM_ID.String(),
		PANCAKE_SWAP_PROGRAM_ID.String():
		return true
	}
	return false
}
