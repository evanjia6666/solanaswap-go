package solanaswapgo

import (
	"encoding/binary"
	"encoding/hex"
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

func (p *Parser) processRaydSwaps(router solana.PublicKey, instructionIndex int, innerIdx int, instruction *solana.CompiledInstruction, isInner bool) []SwapData {
	if router.Equals(RAYDIUM_V4_PROGRAM_ID) && instruction.Data[0] == 1 && !isInner { // init liquidity
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
			Router:             router,
			Amm:                router,
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
		if p.setTxPoolInfo(router, tx, p.txInfo.Message.Instructions[instructionIndex]) != nil {
			return nil
		}
		return []SwapData{
			{
				Type: RAYDIUM,
				Tx:   tx,
			},
		}
	}

	if router.Equals(RAYDIUM_AMM_ROUTER_PROGRAM_ID) && isInner {
		innerInstructions := p.getInnerInstructions(instructionIndex)
		var swaps []SwapData
		for i, innerInstruction := range innerInstructions {
			pID := p.allAccountKeys[innerInstruction.ProgramIDIndex]
			if pID.Equals(RAYDIUM_CPMM_PROGRAM_ID) || pID.Equals(RAYDIUM_V4_PROGRAM_ID) || pID.Equals(RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID) {
				tx := &TxInfo{}
				tx.Router = router
				tx.Amm = pID
				tx.Owner = *p.txInfo.Message.Signers().Last()
				tx.Index = uint(instructionIndex*256) + uint(i)

				innerSwaps := []SwapData{}
				for _, inner := range innerInstructions[i:] {
					switch {
					case p.isTransfer(inner):
						transfer := p.processTransfer(inner)
						if transfer != nil {
							innerSwaps = append(innerSwaps, SwapData{Type: RAYDIUM, Data: transfer})
						}
					case p.isTransferCheck(inner):
						transfer := p.processTransferCheck(inner)
						if transfer != nil {
							innerSwaps = append(innerSwaps, SwapData{Type: RAYDIUM, Data: transfer})
						}

					}
					if len(innerSwaps) >= 2 {
						break
					}
				}
				for i, swap := range innerSwaps {
					switch swap.Data.(type) {
					case *TransferData:
						transfer := swap.Data.(*TransferData)
						if i == 0 {
							tx.InputMint = solana.MustPublicKeyFromBase58(transfer.Mint)
							tx.InputMintDecimals = transfer.Decimals
							tx.InputAmount = transfer.Info.Amount
							continue
						}

						if tx.InputMint.Equals(solana.MustPublicKeyFromBase58(transfer.Mint)) {
							tx.InputAmount = transfer.Info.Amount
							continue
						}

						tx.OutputMint = solana.MustPublicKeyFromBase58(transfer.Mint)
						tx.OutputMintDecimals = transfer.Decimals
						tx.OutputAmount = transfer.Info.Amount
					case *TransferCheck:
						transfer := swap.Data.(*TransferCheck)
						amount, _ := strconv.ParseFloat(transfer.Info.TokenAmount.Amount, 64)
						if i == 0 {
							tx.InputMint = solana.MustPublicKeyFromBase58(transfer.Info.Mint)
							tx.InputMintDecimals = transfer.Info.TokenAmount.Decimals
							tx.InputAmount = uint64(amount)
							continue
						}

						if tx.InputMint.Equals(solana.MustPublicKeyFromBase58(transfer.Info.Mint)) {
							tx.InputAmount = uint64(amount)
							continue
						}

						tx.OutputMint = solana.MustPublicKeyFromBase58(transfer.Info.Mint)
						tx.OutputMintDecimals = transfer.Info.TokenAmount.Decimals
						tx.OutputAmount = uint64(amount)
					}
				}

				if err := p.setTxPoolInfo(pID, tx, innerInstruction); err != nil {
					p.Log.Error(err)
					return swaps
				}

				swaps = append(swaps, SwapData{
					Data: nil,
					Tx:   tx,
					Type: RAYDIUM,
				})
			}
		}
		return swaps
	}

	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			var innerSwaps []SwapData
			for i, innerInstruction := range innerInstructionSet.Instructions {
				// When invoked as an inner instruction by a router, only scan
				// transfers that appear *after* the current AMM instruction so
				// we don't mix transfers from earlier legs (e.g. PumpFun).
				if isInner && i < innerIdx {
					continue
				}
				switch {
				case p.isTransfer(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransfer(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: RAYDIUM, Data: transfer})
					}
				case p.isTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: RAYDIUM, Data: transfer})
					}
				}
			}
			var inst *solana.CompiledInstruction
			if isInner {
				inst = instruction
			}
			tx, err := p.parseTransferTxInfo(router, instructionIndex, RAYDIUM, innerSwaps, inst)
			if err == nil {
				swaps = append(swaps, SwapData{Type: RAYDIUM, Tx: tx})
			}
		}
	}
	return swaps
}
func (p *Parser) parseTransferTxInfo(progId solana.PublicKey, instructionIndex int, protocal SwapType, swaps []SwapData, instruction *solana.CompiledInstruction) (tx *TxInfo, err error) {
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
				continue
			}

			if tx.InputMint.Equals(solana.MustPublicKeyFromBase58(transfer.Mint)) {
				tx.InputAmount = transfer.Info.Amount
				continue
			}

			if !tx.OutputMint.IsZero() {
				continue
			}
			tx.OutputMint = solana.MustPublicKeyFromBase58(transfer.Mint)
			tx.OutputMintDecimals = transfer.Decimals
			tx.OutputAmount = transfer.Info.Amount

		case *TransferCheck:
			transfer := swap.Data.(*TransferCheck)
			amount, _ := strconv.ParseFloat(transfer.Info.TokenAmount.Amount, 64)
			if i == 0 {
				tx.InputMint = solana.MustPublicKeyFromBase58(transfer.Info.Mint)
				tx.InputMintDecimals = transfer.Info.TokenAmount.Decimals
				tx.InputAmount = uint64(amount)
				continue
			}

			if tx.InputMint.Equals(solana.MustPublicKeyFromBase58(transfer.Info.Mint)) {
				tx.InputAmount = uint64(amount)
				continue
			}

			if !tx.OutputMint.IsZero() {
				continue
			}
			tx.OutputMint = solana.MustPublicKeyFromBase58(transfer.Info.Mint)
			tx.OutputMintDecimals = transfer.Info.TokenAmount.Decimals
			tx.OutputAmount = uint64(amount)

		}
	}
	if instruction == nil {
		instruction = &p.txInfo.Message.Instructions[instructionIndex]
	}
	err = p.setTxPoolInfo(progId, tx, *instruction)
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
						innerSwaps = append(innerSwaps, SwapData{Type: ORCA, Data: transfer})
					}
				}
			}
			tx, err := p.parseTransferTxInfo(ORCA_PROGRAM_ID, instructionIndex, ORCA, innerSwaps, nil)
			if err == nil {
				swaps = append(swaps, SwapData{Type: ORCA, Tx: tx})
			}
		}
	}
	return swaps
}

func (p *Parser) processHumidifiSwaps(instructionIndex int, innerIdx int, instruction *solana.CompiledInstruction) []SwapData {
	var swaps []SwapData
	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			var innerSwaps []SwapData
			for i, innerInstruction := range innerInstructionSet.Instructions {
				if i < innerIdx {
					continue
				}
				// Stop if we encounter another known AMM (avoid mixing multi-leg transfers)
				if i > innerIdx {
					progID := p.allAccountKeys[p.convertRPCToSolanaInstruction(innerInstruction).ProgramIDIndex]
					if progID.Equals(PUMPFUN_AMM_PROGRAM_ID) ||
						progID.Equals(RAYDIUM_V4_PROGRAM_ID) ||
						progID.Equals(RAYDIUM_CPMM_PROGRAM_ID) ||
						progID.Equals(RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID) ||
						progID.Equals(ORCA_PROGRAM_ID) ||
						progID.Equals(METEORA_PROGRAM_ID) ||
						progID.Equals(METEORA_POOLS_PROGRAM_ID) ||
						progID.Equals(METEORA_DLMM_PROGRAM_ID) ||
						progID.Equals(METEORA_DAMM_V2) ||
						progID.Equals(ZEROFI) {
						break
					}
				}
				switch {
				case p.isTransfer(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransfer(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: HUMIDIDI, Data: transfer})
					}
				case p.isTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction)):
					transfer := p.processTransferCheck(p.convertRPCToSolanaInstruction(innerInstruction))
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: HUMIDIDI, Data: transfer})
					}
				}
			}
			tx, err := p.parseTransferTxInfo(HUMIDIDI_PROGRAM_ID, instructionIndex, HUMIDIDI, innerSwaps, instruction)
			if err == nil {
				// HumidiFi emits output transfer (pool -> user) first, then input transfer
				// (user -> pool). parseTransferTxInfo assumes first transfer is input,
				// so we need to swap input/output for HumidiFi.
				if len(innerSwaps) >= 2 {
					tx.InputMint, tx.OutputMint = tx.OutputMint, tx.InputMint
					tx.InputAmount, tx.OutputAmount = tx.OutputAmount, tx.InputAmount
					tx.InputMintDecimals, tx.OutputMintDecimals = tx.OutputMintDecimals, tx.InputMintDecimals
				}
				swaps = append(swaps, SwapData{Type: HUMIDIDI, Tx: tx})
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

func (p *Parser) processGeneralRouter(instructionIndex int) []SwapData {

	router := p.allAccountKeys[p.txInfo.Message.Instructions[instructionIndex].ProgramIDIndex]
	innerInstructions := p.getInnerInstructions(instructionIndex)
	var swaps []SwapData
	for i, innerInstruction := range innerInstructions {
		pID := p.allAccountKeys[innerInstruction.ProgramIDIndex]
		if len(innerInstruction.Data) > 8 && swapDiscriminator[hex.EncodeToString(innerInstruction.Data[:8])] == true {
			tx := &TxInfo{}
			tx.Router = router
			tx.Amm = pID
			tx.Owner = *p.txInfo.Message.Signers().Last()
			tx.Index = uint(instructionIndex*256) + uint(i)

			innerSwaps := []SwapData{}
			for _, inner := range innerInstructions[i:] {
				switch {
				case p.isTransfer(inner):
					transfer := p.processTransfer(inner)
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: RAYDIUM, Data: transfer})
					}
				case p.isTransferCheck(inner):
					transfer := p.processTransferCheck(inner)
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: RAYDIUM, Data: transfer})
					}

				}
				if len(innerSwaps) >= 2 {
					break
				}
			}
			for i, swap := range innerSwaps {
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

			if err := p.setTxPoolInfo(pID, tx, innerInstruction); err != nil {
				p.Log.Error(err)
				return swaps
			}

			swaps = append(swaps, SwapData{
				Data: nil,
				Tx:   tx,
				Type: RAYDIUM,
			})
		}
	}
	return swaps

}
