package solanaswapgo

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/pumpfun"
	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/pumpswap"
	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

var (
	PumpfunTradeEventDiscriminator  = [16]byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 219, 127, 211, 78, 230, 97, 238}
	PumpfunCreateEventDiscriminator = [16]byte{228, 69, 165, 46, 81, 203, 154, 29, 27, 114, 169, 77, 222, 235, 99, 118}
)

type PumpfunTradeEvent struct {
	Mint                 solana.PublicKey
	SolAmount            uint64
	TokenAmount          uint64
	IsBuy                bool
	User                 solana.PublicKey
	Timestamp            int64
	VirtualSolReserves   uint64
	VirtualTokenReserves uint64
}

type PumpfunCreateEvent struct {
	Name         string
	Symbol       string
	Uri          string
	Mint         solana.PublicKey
	BondingCurve solana.PublicKey
	User         solana.PublicKey
}

func (p *Parser) processPumpfunSwaps(instructionIndex int) []SwapData {
	// Pump.fun bonding curve V2 (buy_v2 / sell_v2 / *_v2) carries a non-SOL quote token
	// (e.g. USDC). The legacy TradeEvent path only reads sol_amount and yields nil amounts
	// for those, so handle V2 explicitly from instruction accounts + the full IDL event.
	outer := p.txInfo.Message.Instructions[instructionIndex]
	if len(outer.Data) >= 8 {
		disc := [8]byte(outer.Data[:8])
		switch disc {
		case pumpfun.Instruction_BuyV2, pumpfun.Instruction_SellV2,
			pumpfun.Instruction_BuyExactQuoteInV2:
			if tx := p.processPumpfunV2Swap(instructionIndex, outer); tx != nil {
				return []SwapData{{Type: PUMP_FUN, Tx: tx}}
			}
			return nil
		}
	}

	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			for _, innerInstruction := range innerInstructionSet.Instructions {
				if p.isPumpFunTradeEventInstruction(p.convertRPCToSolanaInstruction(innerInstruction)) {
					eventData, err := p.parsePumpfunTradeEventInstruction(p.convertRPCToSolanaInstruction(innerInstruction))
					if err != nil {
						p.Log.Errorf("error processing Pumpfun trade event: %s", err)
					}
					if eventData != nil {
						swaps = append(swaps, SwapData{Type: PUMP_FUN, Data: eventData})
					}
				}
			}
		}
	}
	return swaps
}

// processPumpfunV2Swap builds a TxInfo for a Pump.fun bonding-curve V2 swap.
// V2 account layout: [1]=base mint, [2]=quote mint, [10]=pool,
// [11]=pool base vault, [12]=pool quote vault.
// Amounts come from the full IDL TradeEvent (token_amount + quote_amount), which is
// correct regardless of whether the quote token is SOL or an SPL token such as USDC.
func (p *Parser) processPumpfunV2Swap(instructionIndex int, outer solana.CompiledInstruction) *TxInfo {
	if len(outer.Accounts) < 13 {
		p.Log.Errorf("pumpfun V2 swap: too few accounts %d", len(outer.Accounts))
		return nil
	}
	baseMint := p.allAccountKeys[outer.Accounts[1]]
	quoteMint := p.allAccountKeys[outer.Accounts[2]]
	pool := p.allAccountKeys[outer.Accounts[10]]
	baseVault := p.allAccountKeys[outer.Accounts[11]]
	quoteVault := p.allAccountKeys[outer.Accounts[12]]

	event := p.findPumpfunV2Event(instructionIndex)
	if event == nil {
		p.Log.Errorf("pumpfun V2 swap: no TradeEvent found, sig %v", p.txInfo.Signatures)
		return nil
	}

	tx := &TxInfo{
		Type:     TxTypeSwap,
		Amm:      p.allAccountKeys[outer.ProgramIDIndex],
		Owner:    *p.txInfo.Message.Signers().Last(),
		Pool:     pool,
		Protocol: string(PUMP_FUN),
		Index:    uint(instructionIndex * 256),
	}

	if event.IsBuy {
		// quote -> base: spend quote token, receive base token
		tx.InputMint = quoteMint
		tx.OutputMint = baseMint
		tx.PoolIn = quoteVault
		tx.PoolOut = baseVault
		tx.InputAmount = event.QuoteAmount
		tx.OutputAmount = event.TokenAmount
		tx.PoolInAmount = new(big.Int).SetUint64(event.RealQuoteReserves)
		tx.PoolOutAmount = new(big.Int).SetUint64(event.RealTokenReserves)
	} else {
		// base -> quote: spend base token, receive quote token
		tx.InputMint = baseMint
		tx.OutputMint = quoteMint
		tx.PoolIn = baseVault
		tx.PoolOut = quoteVault
		tx.InputAmount = event.TokenAmount
		tx.OutputAmount = event.QuoteAmount
		tx.PoolInAmount = new(big.Int).SetUint64(event.RealTokenReserves)
		tx.PoolOutAmount = new(big.Int).SetUint64(event.RealQuoteReserves)
	}
	tx.InputMintDecimals = p.pumpfunDecimals(tx.InputMint)
	tx.OutputMintDecimals = p.pumpfunDecimals(tx.OutputMint)
	return tx
}

// findPumpfunV2Event decodes the full IDL TradeEvent emitted as a self-CPI inner
// instruction under the given outer instruction index.
func (p *Parser) findPumpfunV2Event(instructionIndex int) *pumpfun.TradeEvent {
	for _, set := range p.txMeta.InnerInstructions {
		if set.Index != uint16(instructionIndex) {
			continue
		}
		for _, inner := range set.Instructions {
			si := p.convertRPCToSolanaInstruction(inner)
			if !p.isPumpFunTradeEventInstruction(si) {
				continue
			}
			raw, err := base58.Decode(si.Data.String())
			if err != nil || len(raw) < 16 {
				continue
			}
			var ev pumpfun.TradeEvent
			if err := ag_binary.NewBorshDecoder(raw[16:]).Decode(&ev); err != nil {
				p.Log.Errorf("pumpfun V2 swap: decode TradeEvent: %s", err)
				continue
			}
			return &ev
		}
	}
	return nil
}

func (p *Parser) processPumpfunAMMSwaps(PInstructionIndex int, isInner bool) []SwapData {
	parentInstruction := p.txInfo.Message.Instructions[PInstructionIndex]
	pProgID := p.allAccountKeys[parentInstruction.ProgramIDIndex]

	if isInner {
		var innerSwaps []SwapData
		inners := p.getInnerInstructions(PInstructionIndex)
		for i, inner := range inners {
			switch {
			case p.isPumpFunAMMBuyDiscriminator(inner) || p.isPumpFunAMMBuyExactQuoteInDiscriminator(inner):
				// parse pool
				tx := p.processPumFumAMMBuySwaps(pProgID, inner)
				tx.Index = uint(PInstructionIndex*256) + uint(i)
				// parse event
				for x := i + 1; x < len(inners); x++ {
					if p.isPumpFunAMMSwapEventInstruction(inners[x]) {
						err := p.parsePumpfunAMMSwapEvent(tx, inners[x])
						if err != nil {
							p.Log.Errorf("error processing Pumpfun amm swap event: %s", err)
							return nil
						}
						if tx != nil {
							innerSwaps = append(innerSwaps, SwapData{Type: PUMP_FUN, Data: nil, Tx: tx})
						}
						break
					}
				}
				// insert event
			case p.isPumpFunAMMSellDiscriminator(inner) || p.isPumpFunAMMSellExactInDiscriminator(inner):
				tx := p.processPumpFunAMMSellSwaps(pProgID, inner)
				tx.Index = uint(PInstructionIndex*256) + uint(i)
				// parse event
				for x := i + 1; x < len(inners); x++ {
					if p.isPumpFunAMMSwapEventInstruction(inners[x]) {
						err := p.parsePumpfunAMMSwapEvent(tx, inners[x])
						if err != nil {
							p.Log.Errorf("error processing Pumpfun amm swap event: %s", err)
							return nil
						}
						if tx != nil {
							innerSwaps = append(innerSwaps, SwapData{Type: PUMP_FUN, Data: nil, Tx: tx})
						}
						break
					}
				}
			}

		}
		return innerSwaps
	}

	var swaps []SwapData
	var tx *TxInfo
	switch {
	case p.isPumpFunAMMBuyDiscriminator(parentInstruction) || p.isPumpFunAMMBuyExactQuoteInDiscriminator(parentInstruction):
		tx = p.processPumFumAMMBuySwaps(pProgID, parentInstruction)
	case p.isPumpFunAMMSellDiscriminator(parentInstruction) || p.isPumpFunAMMSellExactInDiscriminator(parentInstruction):
		tx = p.processPumpFunAMMSellSwaps(pProgID, parentInstruction)
	default:
		return nil
	}
	tx.Index = uint(PInstructionIndex * 256)

	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(PInstructionIndex) {

			for _, innerInstruction := range innerInstructionSet.Instructions {
				// switch {
				// case p.isTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction)):
				// 	transfer := p.processTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction))
				// 	if transfer != nil {
				// 		swaps = append(swaps, SwapData{Type: PUMP_FUN, Data: transfer})
				// 	}
				// case p.isTransfer(p.convertRPCToSolanaInstruction(innerInstruction)):
				// 	transfer := p.processTransfer(p.convertRPCToSolanaInstruction(innerInstruction))
				// 	if transfer != nil {
				// 		swaps = append(swaps, SwapData{Type: PUMP_FUN, Data: transfer})
				// 	}
				// }

				if p.isPumpFunAMMSwapEventInstruction(p.convertRPCToSolanaInstruction(innerInstruction)) {
					err := p.parsePumpfunAMMSwapEvent(tx, p.convertRPCToSolanaInstruction(innerInstruction))
					if err != nil {
						p.Log.Errorf("error processing Pumpfun trade event: %s", err)
						return nil
					}
					swaps = append(swaps, SwapData{Type: PUMP_FUN, Data: nil, Tx: tx})
				}
			}
		}
	}
	return swaps
}

func (p *Parser) parsePumpfunTradeEventInstruction(instruction solana.CompiledInstruction) (*PumpfunTradeEvent, error) {
	decodedBytes, err := base58.Decode(instruction.Data.String())
	if err != nil {
		return nil, fmt.Errorf("error decoding instruction data: %s", err)
	}
	decoder := ag_binary.NewBorshDecoder(decodedBytes[16:])

	return handlePumpfunTradeEvent(decoder)
}

func handlePumpfunTradeEvent(decoder *ag_binary.Decoder) (*PumpfunTradeEvent, error) {
	var trade PumpfunTradeEvent
	if err := decoder.Decode(&trade); err != nil {
		return nil, fmt.Errorf("error unmarshaling TradeEvent: %s", err)
	}

	return &trade, nil
}

func (p *Parser) setPumpFunSwapTxInfo(tx *TxInfo, instructIndex int) error {
	var instr *solana.CompiledInstruction
	parentInstr := p.txInfo.Message.Instructions[instructIndex]
	if p.allAccountKeys[parentInstr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) {
		instr = &parentInstr
	} else {
		for _, innerInstructionSet := range p.txMeta.InnerInstructions {
			if innerInstructionSet.Index == uint16(instructIndex) {
				for _, innerInstruction := range innerInstructionSet.Instructions {
					if p.allAccountKeys[innerInstruction.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) {
						inP := p.convertRPCToSolanaInstruction(innerInstruction)
						instr = &inP
						break
					}
				}
			}
		}
	}
	if instr == nil {
		return fmt.Errorf("no match instruction found")
	}

	if len(instr.Data) < 8 {
		return fmt.Errorf("instruction data too short")
	}

	poolIndex := 0
	baseMintIndex := 3
	quoteMintIndex := 4 // wsol
	basePoolIndex := 7  // base
	quotePoolIndex := 8 // quote

	tx.Amm = p.allAccountKeys[instr.ProgramIDIndex]
	tx.Pool = p.allAccountKeys[instr.Accounts[poolIndex]]

	switch {
	case bytes.Equal(instr.Data[:8], pumpswap.Instruction_Buy[:]),
		bytes.Equal(instr.Data[:8], pumpswap.Instruction_BuyExactQuoteIn[:]):
		// BUY: sell sol buy other
		tx.InputMint = p.allAccountKeys[instr.Accounts[quoteMintIndex]]
		tx.OutputMint = p.allAccountKeys[instr.Accounts[baseMintIndex]]
		tx.PoolIn = p.allAccountKeys[instr.Accounts[quotePoolIndex]]
		tx.PoolOut = p.allAccountKeys[instr.Accounts[basePoolIndex]]

	case bytes.Equal(instr.Data[:8], pumpswap.Instruction_Sell[:]),
		bytes.Equal(instr.Data[:8], PumpFunAMMSellExactInDiscriminator[:]):
		// SELL: sell other buy sol
		tx.InputMint = p.allAccountKeys[instr.Accounts[baseMintIndex]]
		tx.OutputMint = p.allAccountKeys[instr.Accounts[quoteMintIndex]]
		tx.PoolIn = p.allAccountKeys[instr.Accounts[basePoolIndex]]
		tx.PoolOut = p.allAccountKeys[instr.Accounts[quotePoolIndex]]
	}
	tx.InputMintDecimals = p.pumpfunDecimals(tx.InputMint)
	tx.OutputMintDecimals = p.pumpfunDecimals(tx.OutputMint)
	tx.Protocol = PROTOCOL_PUMPFUN

	return nil
}
