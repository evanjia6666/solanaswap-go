package solanaswapgo

import "github.com/gagliardetto/solana-go"

// zerofiParser handles ZeroFi as a direct AMM and router inner target.
type zerofiParser struct{}

func (zerofiParser) Name() string                   { return string(ZEROFI_SWAP) }
func (zerofiParser) ProgramIDs() []solana.PublicKey { return []solana.PublicKey{ZEROFI} }
func (zerofiParser) Kind() ParserKind               { return KindAMM }

func (zerofiParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processZerofiSwaps(outerIndex, false)
}

func (zerofiParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return ctx.processZerofiSwaps(outerIndex, true)
}

func init() { Register(zerofiParser{}) }
