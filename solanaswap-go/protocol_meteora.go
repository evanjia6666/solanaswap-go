package solanaswapgo

import "github.com/gagliardetto/solana-go"

// meteoraParser handles the Meteora family (DLMM, Pools, Dynamic Bonding Curve, DAMM V2),
// which shares processMeteoraSwaps. It delegates to that function, which scans inner
// transfers and resolves pool info via setTxPoolInfo. Byreal (a Raydium-layout CLMM
// fork) is handled by raydiumParser.
//
// As an outer instruction, some Meteora programs act as routers (e.g. the King7ki… DLMM
// router): when the direct parse yields nothing, the legacy dispatch fell back to
// processRouterSwaps, so ParseOuter preserves that fallback.
type meteoraParser struct{}

func (meteoraParser) Name() string { return string(METEORA) }

func (meteoraParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{
		METEORA_PROGRAM_ID,
		METEORA_POOLS_PROGRAM_ID,
		METEORA_DLMM_PROGRAM_ID,
		Meteora_Dynamic_Bonding_Curve_Program,
		METEORA_DAMM_V2,
	}
}

func (meteoraParser) Kind() ParserKind { return KindAMM }

func (meteoraParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	ix := ctx.txInfo.Message.Instructions[outerIndex]
	progID := ctx.allAccountKeys[ix.ProgramIDIndex]
	if swaps := ctx.processMeteoraSwaps(progID, outerIndex, 0, false); len(swaps) > 0 {
		return swaps
	}
	// Fallback: some Meteora programs act as routers (e.g. King7ki... DLMM router).
	return ctx.processRouterSwaps(outerIndex)
}

func (meteoraParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	progID := ctx.allAccountKeys[inner.ProgramIDIndex]
	return ctx.processMeteoraSwaps(progID, outerIndex, innerIndex, true)
}

func init() { Register(meteoraParser{}) }
