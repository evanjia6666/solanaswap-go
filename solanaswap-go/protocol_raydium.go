package solanaswapgo

import "github.com/gagliardetto/solana-go"

// raydiumParser handles the Raydium AMM family (V4, CPMM, CLMM, Launchlab)
// plus Byreal — a full CLMM fork sharing the raydium layout (pool@2,
// vault@5/6, same anchor events) — as direct AMMs and as router inner
// targets. It delegates to processRaydSwaps, which owns the transfer-scan +
// init-liquidity + setTxPoolInfo logic (windowed per inner invocation, so
// every leg of a same-family route is emitted; setTxPoolInfo classifies by
// program id, so Byreal legs keep their own protocol label).
type raydiumParser struct{}

func (raydiumParser) Name() string { return string(RAYDIUM) }

func (raydiumParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{
		RAYDIUM_V4_PROGRAM_ID,
		RAYDIUM_CPMM_PROGRAM_ID,
		RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID,
		RAYDIUM_LAUNCHLAB_PROGRAM_ID,
		BYREAL_CLMM_PROGRAM_ID,
	}
}

func (raydiumParser) Kind() ParserKind { return KindAMM }

func (raydiumParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	ix := ctx.txInfo.Message.Instructions[outerIndex]
	progID := ctx.allAccountKeys[ix.ProgramIDIndex]
	return ctx.processRaydSwaps(progID, outerIndex, 0, &ix, false)
}

func (raydiumParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	progID := ctx.allAccountKeys[inner.ProgramIDIndex]
	return ctx.processRaydSwaps(progID, outerIndex, innerIndex, &inner, true)
}

func init() { Register(raydiumParser{}) }
