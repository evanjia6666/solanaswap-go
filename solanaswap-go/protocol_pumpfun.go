package solanaswapgo

import "github.com/gagliardetto/solana-go"

// pumpfunBondingCurveParser handles the Pump.fun bonding-curve program (PUMP_FUN_PROGRAM_ID
// + its BSfD6 alias). V1 swaps emit TradeEvent; V2 swaps build TxInfo from accounts + the
// full IDL TradeEvent.
type pumpfunBondingCurveParser struct{}

func (pumpfunBondingCurveParser) Name() string { return PROTOCOL_PUMPFUN }
func (pumpfunBondingCurveParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{
		PUMP_FUN_PROGRAM_ID,
		solana.MustPublicKeyFromBase58("BSfD6SHZigAfDWSjzD5Q41jw8LmKwtmjskPH9XW1mrRW"),
	}
}
func (pumpfunBondingCurveParser) Kind() ParserKind { return KindAMM }
func (pumpfunBondingCurveParser) dedupKey() string { return PROTOCOL_PUMPFUN }
func (pumpfunBondingCurveParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processPumpfunSwaps(outerIndex)
}
func (pumpfunBondingCurveParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return ctx.processPumpfunSwaps(outerIndex)
}

// pumpfunAMMParser handles the PumpFun AMM program (pAMMBay…).
type pumpfunAMMParser struct{}

func (pumpfunAMMParser) Name() string { return PROTOCOL_PUMPFUN }
func (pumpfunAMMParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{PUMPFUN_AMM_PROGRAM_ID}
}
func (pumpfunAMMParser) Kind() ParserKind { return KindAMM }
func (pumpfunAMMParser) dedupKey() string { return PROTOCOL_PUMPFUN }

func (pumpfunAMMParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processPumpfunAMMSwaps(outerIndex, false)
}
func (pumpfunAMMParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return ctx.processPumpfunAMMSwaps(outerIndex, true)
}

func init() {
	Register(pumpfunBondingCurveParser{})
	Register(pumpfunAMMParser{})
}
