package solanaswapgo

import "github.com/gagliardetto/solana-go"

// raydiumParser handles the Raydium AMM family (V4, CPMM, CLMM, Launchlab) as a direct
// AMM and as a router inner target. It delegates to the existing processRaydSwaps, which
// owns the transfer-scan + init-liquidity + setTxPoolInfo logic (windowed per inner
// invocation, so every leg of a same-family route is emitted).
type raydiumParser struct{}

func (raydiumParser) Name() string { return string(RAYDIUM) }

func (raydiumParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{
		RAYDIUM_V4_PROGRAM_ID,
		RAYDIUM_CPMM_PROGRAM_ID,
		RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID,
		RAYDIUM_LAUNCHLAB_PROGRAM_ID,
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
