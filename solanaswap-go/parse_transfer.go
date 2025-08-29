package solanaswapgo

import (
	"encoding/binary"
	"strconv"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

type TransferInfo struct {
	Amount      uint64 `json:"amount"`
	Authority   string `json:"authority"`
	Destination string `json:"destination"`
	Source      string `json:"source"`
}

type TransferData struct {
	Info     TransferInfo `json:"info"`
	Type     string       `json:"type"`
	Mint     string       `json:"mint"`
	Decimals uint8        `json:"decimals"`
}

type TokenInfo struct {
	Mint     string
	Decimals uint8
}

type RaydiumInitLiquidity struct {
	Nonce          uint8
	OpenTime       uint64
	InitPcAmount   uint64
	InitCoinAmount uint64
}

func (p *Parser) processRaydSwaps(progId solana.PublicKey, instructionIndex int, instruction *solana.CompiledInstruction, isInner bool) []SwapData {
	if progId.Equals(RAYDIUM_V4_PROGRAM_ID) && instruction.Data[0] == 1 && !isInner { // init liquidity
		decoder := ag_binary.NewBorshDecoder(instruction.Data[1:])
		var data RaydiumInitLiquidity
		if err := decoder.Decode(&data); err != nil {
			return nil
		}
		if len(instruction.Accounts) < 11 {
			return nil
		}
		tx := &TxInfo{
			Type:               TxTypeAdd,
			Router:             progId,
			Amm:                progId,
			Owner:              *p.txInfo.Message.Signers().Last(),
			Protocol:           string(RAYDIUM),
			Index:              uint(instructionIndex * 256),
			InputMint:          p.allAccountKeys[instruction.Accounts[8]],
			InputMintDecimals:  p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[10]].String()].Decimals,
			InputAmount:        data.InitCoinAmount,
			OutputMint:         p.allAccountKeys[instruction.Accounts[9]],
			OutputMintDecimals: p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[11]].String()].Decimals,
			OutputAmount:       data.InitPcAmount,
		}
		if p.setTxPoolInfo(progId, tx, p.txInfo.Message.Instructions[instructionIndex]) != nil {
			return nil
		}
		return []SwapData{
			{
				Type: RAYDIUM,
				Tx:   tx,
			},
		}
	}

	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			var innerSwaps []SwapData
			for _, innerInstruction := range innerInstructionSet.Instructions {
				switch {
				case p.isTransfer(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransfer(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						swaps = append(swaps, SwapData{Type: RAYDIUM, Data: transfer})
					}
				case p.isTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						swaps = append(swaps, SwapData{Type: RAYDIUM, Data: transfer})
					}
				}
			}
			tx, err := p.parseTransferTxInfo(progId, instructionIndex, RAYDIUM, innerSwaps)
			if err == nil {
				swaps = append(swaps, SwapData{Type: RAYDIUM, Tx: tx})
			}
		}
	}
	return swaps
}
func (p *Parser) parseTransferTxInfo(progId solana.PublicKey, instructionIndex int, protocal SwapType, swaps []SwapData) (tx *TxInfo, err error) {
	tx = &TxInfo{
		Router:   progId,
		Amm:      progId,
		Owner:    *p.txInfo.Message.Signers().Last(),
		Protocol: string(protocal),
		Index:    uint(instructionIndex * 256),
	}
	for i, swap := range swaps {
		switch swap.Data.(type) {
		case *TransferData:
			transfer := swap.Data.(*TransferData)
			if i == 0 {
				tx.InputMint = solana.MustPublicKeyFromBase58(transfer.Mint)
				tx.InputMintDecimals = transfer.Decimals
				tx.InputAmount = transfer.Info.Amount
			} else {
				tx.OutputMint = solana.MustPublicKeyFromBase58(transfer.Mint)
				tx.OutputMintDecimals = transfer.Decimals
				tx.OutputAmount = transfer.Info.Amount
			}
		case *TransferCheck:
			transfer := swap.Data.(*TransferCheck)
			amount, _ := strconv.ParseFloat(transfer.Info.TokenAmount.Amount, 64)
			if i == 0 {
				tx.InputMint = solana.MustPublicKeyFromBase58(transfer.Info.Mint)
				tx.InputMintDecimals = transfer.Info.TokenAmount.Decimals
				tx.InputAmount = uint64(amount)
			} else {
				tx.OutputMint = solana.MustPublicKeyFromBase58(transfer.Info.Mint)
				tx.OutputMintDecimals = transfer.Info.TokenAmount.Decimals
				tx.OutputAmount = uint64(amount)
			}
		}
	}
	err = p.setTxPoolInfo(progId, tx, p.txInfo.Message.Instructions[instructionIndex])
	return
}

func (p *Parser) processOrcaSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			var innerSwaps []SwapData
			for _, innerInstruction := range innerInstructionSet.Instructions {
				if p.isTransfer(p.convertRPCToSolanaInstruction(innerInstruction)) {
					transfer := p.processTransfer(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						swaps = append(swaps, SwapData{Type: ORCA, Data: transfer})
					}
				}
			}
			tx, err := p.parseTransferTxInfo(ORCA_PROGRAM_ID, instructionIndex, ORCA, innerSwaps)
			if err == nil {
				swaps = append(swaps, SwapData{Type: ORCA, Tx: tx})
			}
		}
	}
	return swaps
}

func (p *Parser) processTransfer(instr solana.CompiledInstruction) *TransferData {
	amount := binary.LittleEndian.Uint64(instr.Data[1:9])

	transferData := &TransferData{
		Info: TransferInfo{
			Amount:      amount,
			Source:      p.allAccountKeys[instr.Accounts[0]].String(),
			Destination: p.allAccountKeys[instr.Accounts[1]].String(),
			Authority:   p.allAccountKeys[instr.Accounts[2]].String(),
		},
		Type:     "transfer",
		Mint:     p.splTokenInfoMap[p.allAccountKeys[instr.Accounts[1]].String()].Mint,
		Decimals: p.splTokenInfoMap[p.allAccountKeys[instr.Accounts[1]].String()].Decimals,
	}

	if transferData.Mint == "" {
		transferData.Mint = "Unknown"
	}

	return transferData
}

func (p *Parser) extractSPLTokenInfo() error {
	splTokenAddresses := make(map[string]TokenInfo)

	for _, accountInfo := range p.txMeta.PostTokenBalances {
		if !accountInfo.Mint.IsZero() {
			accountKey := p.allAccountKeys[accountInfo.AccountIndex].String()
			splTokenAddresses[accountKey] = TokenInfo{
				Mint:     accountInfo.Mint.String(),
				Decimals: accountInfo.UiTokenAmount.Decimals,
			}
		}
	}

	processInstruction := func(instr solana.CompiledInstruction) {
		if !p.allAccountKeys[instr.ProgramIDIndex].Equals(solana.TokenProgramID) {
			return
		}

		if len(instr.Data) == 0 || (instr.Data[0] != 3 && instr.Data[0] != 12) {
			return
		}

		if len(instr.Accounts) < 3 {
			return
		}

		source := p.allAccountKeys[instr.Accounts[0]].String()
		destination := p.allAccountKeys[instr.Accounts[1]].String()

		if _, exists := splTokenAddresses[source]; !exists {
			splTokenAddresses[source] = TokenInfo{Mint: "", Decimals: 0}
		}
		if _, exists := splTokenAddresses[destination]; !exists {
			splTokenAddresses[destination] = TokenInfo{Mint: "", Decimals: 0}
		}
	}

	for _, instr := range p.txInfo.Message.Instructions {
		processInstruction(instr)
	}
	for _, innerSet := range p.txMeta.InnerInstructions {
		for _, instr := range innerSet.Instructions {
			processInstruction(p.convertRPCToSolanaInstruction(instr))
		}
	}

	for account, info := range splTokenAddresses {
		if info.Mint == "" {
			splTokenAddresses[account] = TokenInfo{
				Mint:     NATIVE_SOL_MINT_PROGRAM_ID.String(),
				Decimals: 9, // Native SOL has 9 decimal places
			}
		}
	}

	p.splTokenInfoMap = splTokenAddresses

	return nil
}
