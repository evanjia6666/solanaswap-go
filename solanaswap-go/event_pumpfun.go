package solanaswapgo

import (
	"bytes"
	"fmt"

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

func (p *Parser) processPumpfunAMMSwaps(PInstructionIndex int, isInner bool) []SwapData {
	parentInstruction := p.txInfo.Message.Instructions[PInstructionIndex]
	pProgID := p.allAccountKeys[parentInstruction.ProgramIDIndex]

	if isInner {
		var innerSwaps []SwapData
		inners := p.getInnerInstructions(PInstructionIndex)
		for i, inner := range inners {
			switch {
			case p.isPumpFunAMMBuyDiscriminator(inner):
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
			case p.isPumpFunAMMSellDiscriminator(inner):
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
	case p.isPumpFunAMMBuyDiscriminator(parentInstruction):
		tx = p.processPumFumAMMBuySwaps(pProgID, parentInstruction)
	case p.isPumpFunAMMSellDiscriminator(parentInstruction):
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
	case bytes.Equal(instr.Data[:8], PumpFunAMMBuyDiscriminator[:]):
		// BUY: sell sol buy other
		tx.InputMint = p.allAccountKeys[instr.Accounts[quoteMintIndex]]
		tx.OutputMint = p.allAccountKeys[instr.Accounts[baseMintIndex]]
		tx.PoolIn = p.allAccountKeys[instr.Accounts[quotePoolIndex]]
		tx.PoolOut = p.allAccountKeys[instr.Accounts[basePoolIndex]]

	case bytes.Equal(instr.Data[:8], PumpFunAMMSellDiscriminator[:]):
		// SELL: sell other buy sol
		tx.InputMint = p.allAccountKeys[instr.Accounts[baseMintIndex]]
		tx.OutputMint = p.allAccountKeys[instr.Accounts[quoteMintIndex]]
		tx.PoolIn = p.allAccountKeys[instr.Accounts[basePoolIndex]]
		tx.PoolOut = p.allAccountKeys[instr.Accounts[quotePoolIndex]]
	}
	tx.InputMintDecimals = p.splDecimalsMap[tx.InputMint.String()]
	tx.OutputMintDecimals = p.splDecimalsMap[tx.OutputMint.String()]
	tx.Protocol = PROTOCOL_PUMPFUN

	return nil
}
