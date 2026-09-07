package solanaswapgo

import "github.com/gagliardetto/solana-go"

// ParseContext is the shared dependency surface handed to each protocol parser.
// Within this package it is the Parser itself (which already carries txMeta, txInfo,
// allAccountKeys, the SPL maps, postBalance and Log, plus every shared helper such as
// getInnerInstructions / isTransfer / processTransfer / pumpfunDecimals). The alias
// gives protocol parsers a name that expresses intent without a wrapper layer.
type ParseContext = Parser

// ParserKind classifies how a program participates in a transaction.
type ParserKind int

const (
	// KindAMM is a liquidity venue invoked directly or as a router inner instruction.
	KindAMM ParserKind = iota
	// KindRouter wraps AMM calls via inner instructions (trading bots, simple routers).
	KindRouter
	// KindAggregator emits its own swap events (Jupiter, DFlow, OKX).
	KindAggregator
)

// ProtocolParser is the unit of protocol knowledge: how to identify a program, how to
// parse it as an outer instruction, and (for AMMs) how to parse it when a router
// delegates to it as an inner instruction. Adding a protocol means implementing this
// interface and calling Register in an init() — no dispatch switch edits.
type ProtocolParser interface {
	// Name is the human-readable protocol label written into TxInfo.Protocol.
	Name() string
	// ProgramIDs lists every on-chain program this parser handles (a family may share one).
	ProgramIDs() []solana.PublicKey
	// Kind classifies the program for dispatch ordering.
	Kind() ParserKind
	// ParseOuter handles the program when it is an outer (top-level) instruction.
	ParseOuter(ctx *ParseContext, outerIndex int) []SwapData
	// ParseInner handles the program when a router delegates to it as an inner
	// instruction. AMMs implement it; pure routers/aggregators may return nil.
	ParseInner(ctx *ParseContext, outerIndex, innerIndex int, inner solana.CompiledInstruction) []SwapData
}

// aggregateParser is an optional capability for AMM parsers whose ParseInner
// aggregates the whole inner-instruction set instead of windowing to the
// invoked leg (pumpfun curve/AMM). The router leg dispatcher runs such parsers
// at most once per program per router instruction to avoid emitting duplicate
// overlapping aggregates. Windowed parsers (raydium, meteora, ...) run for
// every matching inner invocation so multi-leg routes of the same family are
// fully recovered — one dedup key per family used to silently drop e.g. the
// CLMM leg of a Raydium-router CPMM→CLMM route.
type aggregateParser interface {
	aggregatesInner() bool
}

var (
	registry    = map[solana.PublicKey]ProtocolParser{}
	knownAMMSet = map[solana.PublicKey]bool{}
)

// Register adds a protocol parser to the registry. Called from each protocol file's
// init(). Idempotent per program ID (last registration wins).
func Register(p ProtocolParser) {
	for _, id := range p.ProgramIDs() {
		registry[id] = p
		if p.Kind() == KindAMM {
			knownAMMSet[id] = true
		}
	}
}

// lookupParser returns the registered parser for a program ID, if any.
func lookupParser(id solana.PublicKey) (ProtocolParser, bool) {
	p, ok := registry[id]
	return p, ok
}

// layoutRegistry holds pool layouts for simple ACCOUNT_INDEX-only programs that have no
// dedicated parser: they are reached only when a router's transfer scanner calls
// setTxPoolInfo, so they need a layout lookup but not a ParseOuter/ParseInner. Keeping
// them out of `registry`/`knownAMMSet` preserves isKnownAMM membership exactly.
var layoutRegistry = map[solana.PublicKey]poolLayout{}

// RegisterLayout adds a pool layout for a simple program. Called from init().
func RegisterLayout(id solana.PublicKey, l poolLayout) {
	layoutRegistry[id] = l
}

// lookupLayout returns the registered layout for a program ID, if any.
func lookupLayout(id solana.PublicKey) (poolLayout, bool) {
	l, ok := layoutRegistry[id]
	return l, ok
}
