package solanaswapgo

import "github.com/gagliardetto/solana-go"

// --- Moonshot ---

type moonshotParser struct{}

func (moonshotParser) Name() string                   { return string(MOONSHOT) }
func (moonshotParser) ProgramIDs() []solana.PublicKey { return []solana.PublicKey{MOONSHOT_PROGRAM_ID} }
func (moonshotParser) Kind() ParserKind               { return KindAggregator }

func (moonshotParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processMoonshotSwaps()
}

func (moonshotParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return nil
}

// --- OKX Labs 1 ---

type okxLabs1Parser struct{}

func (okxLabs1Parser) Name() string { return "OKX Labs 1" }
func (okxLabs1Parser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{OKX_LABS_1_PROGRAM_ID}
}
func (okxLabs1Parser) Kind() ParserKind { return KindAggregator }

func (okxLabs1Parser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processOKXSwaps(outerIndex)
}

func (okxLabs1Parser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return nil
}

// --- OKX Labs 2 ---

type okxLabs2Parser struct{}

func (okxLabs2Parser) Name() string { return "OKX Labs 2" }
func (okxLabs2Parser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{OKX_LABS_2_PROGRAM_ID}
}
func (okxLabs2Parser) Kind() ParserKind { return KindAggregator }

func (okxLabs2Parser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processOKXLabs2SwapEvents(outerIndex)
}

func (okxLabs2Parser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return nil
}

// --- Jupiter + DFlow (share the same event parser) ---

type jupiterParser struct{}

func (jupiterParser) Name() string { return string(JUPITER) }
func (jupiterParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{JUPITER_PROGRAM_ID, DFLOW_AGGREGATOR_V4}
}
func (jupiterParser) Kind() ParserKind { return KindAggregator }

func (jupiterParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	if swaps := ctx.processJupiterSwaps(outerIndex); len(swaps) > 0 {
		return swaps
	}
	// Fallback: RouteV2 or other newer Jupiter instructions that don't emit
	// JupiterRouteEvent — scan inner instructions like a normal router.
	return ctx.processRouterSwaps(outerIndex)
}

func (jupiterParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return nil
}

func init() {
	Register(moonshotParser{})
	Register(okxLabs1Parser{})
	Register(okxLabs2Parser{})
	Register(jupiterParser{})
}
