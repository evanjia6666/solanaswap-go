package solanaswapgo

import (
	"strconv"

	"github.com/gagliardetto/solana-go"
)

func (p *Parser) processZerofiSwaps(instructionIndex int, isInner bool) []SwapData {
	var swaps []SwapData
	inners := p.getInnerInstructions(instructionIndex)

	if isInner {
		parentInstruction := p.txInfo.Message.Instructions[instructionIndex]
		router := p.allAccountKeys[parentInstruction.ProgramIDIndex]

		for i, inner := range inners {
			if !p.allAccountKeys[inner.ProgramIDIndex].Equals(ZEROFI) || len(inner.Data) < 1 || inner.Data[0] != 6 {
				continue
			}

			innerSwaps := []SwapData{}
			for _, innerInstruction := range inners[i+1:] {
				switch {
				case p.isTransfer(innerInstruction):
					transfer := p.processTransfer(innerInstruction)
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: ZEROFI_SWAP, Data: transfer})
					}
				case p.isTransferCheck(innerInstruction):
					transfer := p.processTransferCheck(innerInstruction)
					if transfer != nil {
						innerSwaps = append(innerSwaps, SwapData{Type: ZEROFI_SWAP, Data: transfer})
					}
				}
				if len(innerSwaps) >= 2 {
					break
				}
			}

			tx := &TxInfo{
				Router:   router,
				Amm:      ZEROFI,
				Owner:    *p.txInfo.Message.Signers().Last(),
				Protocol: "ZeroFi",
				Index:    uint(instructionIndex*256) + uint(i),
			}

			for j, swap := range innerSwaps {
				switch swap.Data.(type) {
				case *TransferData:
					transfer := swap.Data.(*TransferData)
					if j == 0 {
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
					if j == 0 {
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

			if err := p.setTxPoolInfo(ZEROFI, tx, inner); err != nil {
				p.Log.Errorf("failed to parse ZeroFi tx info: %v, signatures: %v", err, p.txInfo.Signatures)
				continue
			}

			swaps = append(swaps, SwapData{
				Type: ZEROFI_SWAP,
				Data: nil,
				Tx:   tx,
			})
		}
		return swaps
	}

	// isInner = false: ZeroFi is the outer instruction
	outerInstr := p.txInfo.Message.Instructions[instructionIndex]
	if !p.allAccountKeys[outerInstr.ProgramIDIndex].Equals(ZEROFI) || len(outerInstr.Data) < 1 || outerInstr.Data[0] != 6 {
		return nil
	}

	var innerSwaps []SwapData
	for _, innerInstruction := range inners {
		switch {
		case p.isTransfer(innerInstruction):
			transfer := p.processTransfer(innerInstruction)
			if transfer != nil {
				innerSwaps = append(innerSwaps, SwapData{Type: ZEROFI_SWAP, Data: transfer})
			}
		case p.isTransferCheck(innerInstruction):
			transfer := p.processTransferCheck(innerInstruction)
			if transfer != nil {
				innerSwaps = append(innerSwaps, SwapData{Type: ZEROFI_SWAP, Data: transfer})
			}
		}
	}

	tx := &TxInfo{
		Router:   ZEROFI,
		Amm:      ZEROFI,
		Owner:    *p.txInfo.Message.Signers().Last(),
		Protocol: "ZeroFi",
		Index:    uint(instructionIndex * 256),
	}

	for j, swap := range innerSwaps {
		switch swap.Data.(type) {
		case *TransferData:
			transfer := swap.Data.(*TransferData)
			if j == 0 {
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
			if j == 0 {
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

	if err := p.setTxPoolInfo(ZEROFI, tx, outerInstr); err != nil {
		p.Log.Errorf("failed to parse ZeroFi tx info: %v, signatures: %v", err, p.txInfo.Signatures)
		return nil
	}

	return []SwapData{{
		Type: ZEROFI_SWAP,
		Data: nil,
		Tx:   tx,
	}}
}
