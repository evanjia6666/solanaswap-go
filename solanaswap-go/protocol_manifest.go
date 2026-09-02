package solanaswapgo

import "github.com/gagliardetto/solana-go"

// manifestParser handles Manifest as a direct AMM and router inner target.
// Manifest uses Borsh-serialised swap parameters rather than Transfer-scanning,
// and has distinct inner-vs-router paths.
type manifestParser struct{}

func (manifestParser) Name() string                   { return string(MANIFEST) }
func (manifestParser) ProgramIDs() []solana.PublicKey { return []solana.PublicKey{MANIFEST_PROGRAM_ID} }
func (manifestParser) Kind() ParserKind               { return KindAMM }

func (manifestParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processManifestSwaps(outerIndex, false)
}

func (manifestParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return ctx.processManifestSwaps(outerIndex, true)
}

func init() { Register(manifestParser{}) }
