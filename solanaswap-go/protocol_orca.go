package solanaswapgo

import "github.com/gagliardetto/solana-go"

// orcaParser handles Orca Whirlpool as a direct AMM and as a router inner target.
// It delegates to processOrcaSwaps, which scans inner transfers and resolves pool info
// via setTxPoolInfo (Swap vs SwapV2 account layout).
type orcaParser struct{}

func (orcaParser) Name() string { return string(ORCA) }

func (orcaParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{ORCA_PROGRAM_ID}
}

func (orcaParser) Kind() ParserKind { return KindAMM }

func (orcaParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processOrcaSwaps(outerIndex, nil)
}

func (orcaParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return ctx.processOrcaSwaps(outerIndex, &inner)
}

func init() { Register(orcaParser{}) }
