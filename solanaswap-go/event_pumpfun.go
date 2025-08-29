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

func (p *Parser) processPumpfunAMMSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			for i, innerInstruction := range innerInstructionSet.Instructions {
				switch {
				case p.isTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						swaps = append(swaps, SwapData{Type: PUMP_FUN, Data: transfer})
					}
				case p.isTransfer(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransfer(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						swaps = append(swaps, SwapData{Type: PUMP_FUN, Data: transfer})
					}
				}

				if p.isPumpFunAMMSwapEventInstruction(p.convertRPCToSolanaInstruction(innerInstruction)) {
					tx, err := p.parsePumpfunAMMSwapEvent(p.convertRPCToSolanaInstruction(innerInstruction), p.allAccountKeys[p.txInfo.Message.Instructions[instructionIndex].ProgramIDIndex], int64(instructionIndex), int64(i))
					if err != nil {
						p.Log.Errorf("error processing Pumpfun trade event: %s", err)
						continue
					}
					if len(swaps) > 0 {
						swaps[0].Tx = tx
					}
				}

			}
		}
	}
	if len(swaps) > 0 {
		p.setPumpFunSwapTxInfo(swaps[0].Tx, p.txInfo.Message.Instructions[instructionIndex])
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

func (p *Parser) setPumpFunSwapTxInfo(tx *TxInfo, instr solana.CompiledInstruction) error {
	if !p.allAccountKeys[instr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) {
		return fmt.Errorf("mismatched program id")
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
