package solanaswapgo

import "github.com/gagliardetto/solana-go"

// genericRouterParser handles trading bots and simple routers that only wrap inner AMM
// calls. Their ParseOuter delegates to processRouterSwaps (which now dispatches
// inner AMMs via the registry). This covers all the programs that were in the legacy
// routerPrograms slice plus the Raydium AMM Router.
type genericRouterParser struct{}

func (genericRouterParser) Name() string { return "GenericRouter" }
func (genericRouterParser) ProgramIDs() []solana.PublicKey {
	return []solana.PublicKey{
		RAYDIUM_AMM_ROUTER_PROGRAM_ID,
		BANANA_GUN_PROGRAM_ID,
		MINTECH_PROGRAM_ID,
		BLOOM_PROGRAM_ID,
		NOVA_PROGRAM_ID,
		MAESTRO_PROGRAM_ID,
		JUPITER_DCA_PROGRAM_ID,
		THREE_Q_ROUTER_PROGRAM_ID,
		BITGET_SWAP_PROGRAM_ID,
		BITGET_DEX_AGGREGATOR_PROGRAM_ID,
		BINANCE_WALLET_PROGRAM_ID,
		AXIOM_TRADE_PROGRAM_ID,
		ARBITRAGE_BOT_3S1R_PROGRAM_ID,
		ARBITRAGE_BOT_B7QNN_PROGRAM_ID,
		solana.MustPublicKeyFromBase58("AP51WLiiqTdbZfgyRMs35PsZpdmLuPDdHYmrB23pEtMU"),
		// Jupiter zap (zapvX9M3…) and the rexhfZLR… router wrap inner AMM
		// legs like the bots above; seen wrapping Byreal routes on mainnet.
		solana.MustPublicKeyFromBase58("zapvX9M3uf5pvy4wRPAbQgdQsM1xmuiFnkfHKPvwMiz"),
		solana.MustPublicKeyFromBase58("rexhfZLRRxRkPkw9izswgMFRPDb9U58jeinH7wqUVuw"),
	}
}
func (genericRouterParser) Kind() ParserKind { return KindRouter }

func (genericRouterParser) ParseOuter(ctx *ParseContext, outerIndex int) []SwapData {
	return ctx.processRouterSwaps(outerIndex)
}

func (genericRouterParser) ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData {
	return nil
}

func init() { Register(genericRouterParser{}) }
