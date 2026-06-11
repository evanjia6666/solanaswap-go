package solanaswapgo

import "github.com/gagliardetto/solana-go"

// humidifiParser handles HumidiFi as a direct AMM and router inner target.
type humidifiParser struct{}

func (humidifiParser) Name() string                   { return string(HUMIDIDI) }
func (humidifiParser) ProgramIDs() []solana.PublicKey { return []solana.PublicKey{HUMIDIDI_PROGRAM_ID} }
func (humidifiParser) Kind() ParserKind               { return KindAMM }

func (humidifiParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	ix := ctx.txInfo.Message.Instructions[outerIndex]
	return ctx.processHumidifiSwaps(outerIndex, 0, &ix)
}

func (humidifiParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return ctx.processHumidifiSwaps(outerIndex, innerIndex, &inner)
}

func init() { Register(humidifiParser{}) }
