package solanaswapgo

import "github.com/gagliardetto/solana-go"

// genericLayouts lists the simple ACCOUNT_INDEX-only programs that have no dedicated
// parser. They are reached only when a router's transfer scanner calls setTxPoolInfo,
// so they need a pool layout but no ParseOuter/ParseInner, and they are intentionally
// NOT added to knownAMMSet (isKnownAMM membership must stay unchanged).
//
// Every entry here is a verbatim port of a case from the legacy setTxPoolInfo switch.
// Programs whose account layout varies with account count (StableWeighted, stabble
// Stable Swap) are NOT here — they remain in the legacy switch.
var genericLayouts = []struct {
	id     string
	layout poolLayout
}{
	{"SoLFiHG9TfgtdUXUjWAxi3LtvYuFyDLVhBWxdMZxyCe", poolLayout{poolIdx: 1, poolInIdx: 2, poolOutIdx: 3, protocol: "SolFi", discriminatorLen: 1, whitelist: [][]byte{{7}}}},
	{"2wT8Yq49kHgDzXuPxZSaeLaH1qbmGXtEyPy64bL7aD3c", poolLayout{poolIdx: 1, poolInIdx: 5, poolOutIdx: 6, protocol: "Lifinity Swap V2"}},
	{"9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Orca Token Swap V2", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"DEXYosS6oEGvk8uCDayvwEZz4qEyDJRf9nFgYCaqPMTm", poolLayout{poolIdx: 2, poolInIdx: 3, poolOutIdx: 4, protocol: "1Dex"}},
	{"H8W3ctz92svYg6mkn1UtGfu2aQr2fnUFHM1RhScEtQDt", poolLayout{poolIdx: 2, poolInIdx: 4, poolOutIdx: 6, protocol: "Cropper"}},
	{"HyaB3W9q6XdA5xwpU4XnSZV94htfmbmqJXZcEbRaJutt", poolLayout{poolIdx: 1, poolInIdx: 5, poolOutIdx: 6, protocol: "Invariant"}},
	{"SSwpkEEcbUqx4vtoEByFjSkhKdCT862DNVb52nZg1UZ", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Saber Stable Swap", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"SSwapUtytfBdBn1b9NUGG6foMVPtcWgpRU32HToDUZr", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Saros", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"FLUXubRmkEi2q6K3Y9kBPg9248ggaZVsoSFhtJHSrm1X", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Fluxbeam", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"Gswppe6ERWKpUTXvRPfXdzHhiCyJvLadVvXGfdpBqcE1", poolLayout{poolIdx: 1, poolInIdx: 4, poolOutIdx: 5, protocol: "Guac"}},
	{"BSwp6bEBihVLdqJRKGgzjcGLHkcTuzmSo1TQkHepzH8p", poolLayout{poolIdx: 1, poolInIdx: 4, poolOutIdx: 5, protocol: "BonkSwap"}},
	{"DSwpgjMvXhtGn6BsbqmacdBZyfLj6jSWf3HJpdJtmg6N", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "DexlabSwap", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"CURVGoZn8zycx6FXwwevgBTB2gVvdbGTEpvMJDbgs2t4", poolLayout{poolIdx: 0, poolInIdx: 3, poolOutIdx: 4, protocol: "Aldrin"}},
	{"AMM55ShdkoGRB5jVYPjWziwk8m5MpwyDgsMWHaMSQWH6", poolLayout{poolIdx: 0, poolInIdx: 3, poolOutIdx: 4, protocol: "Aldrin"}},
	{"DjVE6JNiYqPL2QXyCUUh8rNjHrbz9hXHNYt99MQ59qw1", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Orca Token Swap", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Swap Program", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"5jnapfrAN47UYkLkEf7HnprPPBCQLvkYWGZDeKkaP5hv", poolLayout{poolIdx: 4, poolInIdx: 7, poolOutIdx: 8, protocol: "DaoFun"}},
	{"CLMM9tUoggJu2wagPkkqs9eFG4BWhVBZWkP1qv3Sp7tR", poolLayout{poolIdx: 1, poolInIdx: 6, poolOutIdx: 7, protocol: "Crema Finance Program"}},
	{"Dooar9JkhdZ7J3LHN3A7YCuoGRUggXhQaG4kijfLGU2j", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "StepN DOOAR Swap", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"treaf4wWBBty3fHdyBpo35Mz84M8k3heKXmjmi9vFt5", poolLayout{poolIdx: 0, poolInIdx: 3, poolOutIdx: 4, protocol: "Helium Treasury Management"}},
	{"PSwapMdSai8tjrEXcxFeQth87xC4rRsa4VA5mhGhXkP", poolLayout{poolIdx: 0, poolInIdx: 4, poolOutIdx: 5, protocol: "Penguin Finance", discriminatorLen: 1, whitelist: [][]byte{{1}}}},
	{"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY", poolLayout{poolIdx: 2, poolInIdx: 6, poolOutIdx: 7, protocol: "Phoenix", discriminatorLen: 1, whitelist: [][]byte{{0}}}},
	{"HpNfyc2Saw7RKkQd8nEL4khUcuPhQ7WwY1B2qjx8jxFq", poolLayout{poolIdx: 2, poolInIdx: 5, poolOutIdx: 6, protocol: "PancakeSwap"}},
	{"TessVdML9pBGgG9yGks7o4HewRaXVAMuoVj4x83GLQH", poolLayout{poolIdx: 1, poolInIdx: 4, poolOutIdx: 5, protocol: "Tessera V", discriminatorLen: 1, whitelist: [][]byte{{16}}}},
}

func init() {
	for _, e := range genericLayouts {
		RegisterLayout(solana.MustPublicKeyFromBase58(e.id), e.layout)
	}
}
