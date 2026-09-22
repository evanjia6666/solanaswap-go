package solanaswapgo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	meteora_dlmm "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/meteora/meteora_dlmm"
	solana "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
)

// DLMM state events via anchor emit_cpi (self-CPI inner instructions).
//
// The deployed DLMM program emits its events through anchor's emit_cpi: an
// inner instruction invoking the program itself, data = self-CPI
// discriminator + event discriminator + payload (no "Program data:" log
// line, so the log-based forEachAnchorEvent never sees them). The same
// mechanism carries pumpswap's and jupiter's events, so the self-CPI
// discriminator is shared. Newer swaps emit BOTH the legacy Swap event and
// the extended Swap2Evt; the extractor keeps Swap2Evt (superset: it adds
// amount_left and the fee-direction flags) and dedupes per pool.
//
// Layouts: the generated package's event discriminators still match the
// deployed program (verified on mainnet — swap2 instruction disc
// 414b3f4c…, Swap payload 129B, Swap2Evt payload 147B), so the generated
// parsers decode the legacy events; Swap2Evt postdates the generated
// package and is hand-decoded from the published IDL field order. The
// restriction to the LBUZ… program id is deliberate: the second registered
// id (King7…) has not been live-verified.

// DLMM event kinds.
const (
	DLMMEventSwap      = "swap"
	DLMMEventLiquidity = "liquidity"
	DLMMEventCreate    = "create"
	DLMMEventFee       = "fee"
)

// DLMMStateEvent is the normalized DLMM event consumed by the dex-side
// state maintenance (zero-RPC active-bin updates, dirty marking, pool
// registration at creation).
type DLMMStateEvent struct {
	Program string
	Pool    string

	Kind string

	// Swap fields.
	StartBinId int32
	EndBinId   int32
	AmountIn   uint64
	AmountOut  uint64
	SwapForY   bool
	FeeBps     *big.Int // u128 dynamic-fee snapshot carried by the event

	// Create fields.
	BinStep uint16
	TokenX  string
	TokenY  string
}

// dlmmEventSwap2Evt is the Swap2Evt discriminator from the published IDL;
// the generated package predates it.
var dlmmEventSwap2Evt = [8]byte{46, 116, 82, 215, 148, 27, 84, 77}

// ExtractDLMMEvents parses DLMM self-CPI events from a transaction's inner
// instructions, in emission order. allKeys is the combined account key list
// (message static keys + loaded writable + readonly) that inner
// instructions index into.
func ExtractDLMMEvents(meta *solrpc.TransactionMeta, allKeys []solana.PublicKey) []DLMMStateEvent {
	var out []DLMMStateEvent
	var supersededSwap []int // legacy Swap events superseded by Swap2Evt on the same pool
	forEachSelfCPIEvent(meta, allKeys, func(program solana.PublicKey, disc [8]byte, raw []byte) {
		if !program.Equals(METEORA_PROGRAM_ID) {
			return
		}
		switch disc {
		case meteora_dlmm.Event_Swap:
			ev, err := meteora_dlmm.ParseEvent_Swap(raw)
			if err != nil {
				return
			}
			out = append(out, DLMMStateEvent{
				Program:    program.String(),
				Pool:       ev.LbPair.String(),
				Kind:       DLMMEventSwap,
				StartBinId: ev.StartBinId,
				EndBinId:   ev.EndBinId,
				AmountIn:   ev.AmountIn,
				AmountOut:  ev.AmountOut,
				SwapForY:   ev.SwapForY,
				FeeBps:     ev.FeeBps.BigInt(),
			})
		case dlmmEventSwap2Evt:
			ev, err := parseDLMMSwap2Evt(program, raw)
			if err != nil {
				return
			}
			// a newer build emits Swap and Swap2Evt for one swap: the legacy
			// copy of the same pool is redundant
			for i := range out {
				if out[i].Kind == DLMMEventSwap && out[i].Pool == ev.Pool {
					supersededSwap = append(supersededSwap, i)
				}
			}
			out = append(out, ev)
		case meteora_dlmm.Event_AddLiquidity, meteora_dlmm.Event_RemoveLiquidity,
			meteora_dlmm.Event_FeeParameterUpdate, meteora_dlmm.Event_DynamicFeeParameterUpdate:
			// dirty/fee events only need the pool: lb_pair is the first
			// field after the discriminator in all four layouts
			if len(raw) < 8+32 {
				return
			}
			kind := DLMMEventLiquidity
			if disc == meteora_dlmm.Event_FeeParameterUpdate || disc == meteora_dlmm.Event_DynamicFeeParameterUpdate {
				kind = DLMMEventFee
			}
			out = append(out, DLMMStateEvent{
				Program: program.String(),
				Pool:    solana.PublicKeyFromBytes(raw[8:40]).String(),
				Kind:    kind,
			})
		case meteora_dlmm.Event_LbPairCreate:
			ev, err := meteora_dlmm.ParseEvent_LbPairCreate(raw)
			if err != nil {
				return
			}
			out = append(out, DLMMStateEvent{
				Program: program.String(),
				Pool:    ev.LbPair.String(),
				Kind:    DLMMEventCreate,
				BinStep: ev.BinStep,
				TokenX:  ev.TokenX.String(),
				TokenY:  ev.TokenY.String(),
			})
		}
	})
	if len(supersededSwap) == 0 {
		return out
	}
	drop := make(map[int]struct{}, len(supersededSwap))
	for _, i := range supersededSwap {
		drop[i] = struct{}{}
	}
	filtered := make([]DLMMStateEvent, 0, len(out))
	for i, ev := range out {
		if _, skip := drop[i]; !skip {
			filtered = append(filtered, ev)
		}
	}
	return filtered
}

// ExtractDLMMEventsFromTransaction is the convenience form of
// ExtractDLMMEvents for callers holding the decoded transaction: it builds
// the combined account key list (static + loaded writable + readonly) inner
// instructions index into.
func ExtractDLMMEventsFromTransaction(tx *solana.Transaction, meta *solrpc.TransactionMeta) []DLMMStateEvent {
	if tx == nil || meta == nil {
		return nil
	}
	allKeys := append(append([]solana.PublicKey{}, tx.Message.AccountKeys...), meta.LoadedAddresses.Writable...)
	allKeys = append(allKeys, meta.LoadedAddresses.ReadOnly...)
	return ExtractDLMMEvents(meta, allKeys)
}

// forEachSelfCPIEvent walks a transaction's inner instructions handing every
// anchor emit_cpi payload to fn together with the emitting program and the
// event data (event discriminator + payload). A self-CPI event instruction
// invokes the program itself and carries exactly one account (the event
// authority), so the inner instruction's program IS the emitter — no
// call-stack attribution needed.
func forEachSelfCPIEvent(meta *solrpc.TransactionMeta, allKeys []solana.PublicKey, fn func(program solana.PublicKey, disc [8]byte, raw []byte)) {
	if meta == nil {
		return
	}
	for _, set := range meta.InnerInstructions {
		for _, ix := range set.Instructions {
			if int(ix.ProgramIDIndex) >= len(allKeys) {
				continue
			}
			if len(ix.Data) < 16 {
				continue
			}
			if !bytes.Equal(ix.Data[:8], AnchorSelfCPIDiscriminator[:]) {
				continue
			}
			var disc [8]byte
			copy(disc[:], ix.Data[8:16])
			fn(allKeys[ix.ProgramIDIndex], disc, ix.Data[8:])
		}
	}
}

// parseDLMMSwap2Evt decodes the Swap2Evt event data (discriminator prefix
// included). Field order from the published IDL:
// lb_pair, from, start_bin_id, end_bin_id, swap_for_y, fee_bps(u128),
// amount_in, amount_left, amount_out, mm_fee, protocol_fee,
// limit_order_fee, host_fee, fees_on_input, fees_on_token_x — 147 payload
// bytes, matching the mainnet sample.
func parseDLMMSwap2Evt(program solana.PublicKey, raw []byte) (DLMMStateEvent, error) {
	if len(raw) < 8+147 {
		return DLMMStateEvent{}, fmt.Errorf("dlmm Swap2Evt: short payload %d", len(raw))
	}
	p := raw[8:]
	getU64 := func(off int) uint64 { return binary.LittleEndian.Uint64(p[off : off+8]) }
	// u128 little-endian: reverse for big.Int
	feeBytes := make([]byte, 16)
	for i := 0; i < 16; i++ {
		feeBytes[15-i] = p[73+i]
	}
	return DLMMStateEvent{
		Program:    program.String(),
		Pool:       solana.PublicKeyFromBytes(p[0:32]).String(),
		Kind:       DLMMEventSwap,
		StartBinId: int32(binary.LittleEndian.Uint32(p[64:68])),
		EndBinId:   int32(binary.LittleEndian.Uint32(p[68:72])),
		SwapForY:   p[72] == 1,
		FeeBps:     new(big.Int).SetBytes(feeBytes),
		AmountIn:   getU64(89),
		AmountOut:  getU64(105),
	}, nil
}
