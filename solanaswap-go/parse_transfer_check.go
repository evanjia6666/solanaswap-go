package solanaswapgo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	meteoradlmmprogram "github.com/franco-bianco/solanaswap-go/solanaswap-go/meteora_dlmm_program"
	"github.com/franco-bianco/solanaswap-go/solanaswap-go/meteora_pools_program"
	"github.com/gagliardetto/solana-go"
)

type TransferCheck struct {
	Info struct {
		Authority   string `json:"authority"`
		Destination string `json:"destination"`
		Mint        string `json:"mint"`
		Source      string `json:"source"`
		TokenAmount struct {
			Amount         string  `json:"amount"`
			Decimals       uint8   `json:"decimals"`
			UIAmount       float64 `json:"uiAmount"`
			UIAmountString string  `json:"uiAmountString"`
		} `json:"tokenAmount"`
	} `json:"info"`
	Type string `json:"type"`
}

func (p *Parser) processMeteoraSwaps(progID solana.PublicKey, outerIndex int, innerIndex int, isInner bool) []SwapData {
	if isInner {
		outerInstriction := p.txInfo.Message.Instructions[outerIndex]
		router := p.allAccountKeys[outerInstriction.ProgramIDIndex]
		inners := p.getInnerInstructions(outerIndex)[innerIndex:]
		switch {
		case progID.Equals(METEORA_POOLS_PROGRAM_ID) || progID.Equals(Meteora_Dynamic_Bonding_Curve_Program) || progID.Equals(METEORA_DAMM_V2) || progID.Equals(METEORA_PROGRAM_ID):
			for i, inner := range inners {
				discriminator := inner.Data[:8]
				inProgID := p.allAccountKeys[inner.ProgramIDIndex]
				if progID.Equals(inProgID) && bytes.Equal(discriminator, meteora_pools_program.Instruction_Swap[:]) {
					var innerSwaps []SwapData
					for _, innerInstruction := range inners[i+1:] {
						switch {
						case p.isTransferCheck(innerInstruction):
							transfer := p.processTransferCheck(innerInstruction)
							if transfer != nil {
								innerSwaps = append(innerSwaps, SwapData{Type: METEORA, Data: transfer})
							}
						case p.isTransfer(innerInstruction):
							transfer := p.processTransfer(innerInstruction)
							if transfer != nil {
								innerSwaps = append(innerSwaps, SwapData{Type: METEORA, Data: transfer})
							}
						}
						if len(innerSwaps) >= 3 {
							break
						}
					}
					tx := &TxInfo{
						Router:   router,
						Amm:      progID,
						Owner:    *p.txInfo.Message.Signers().Last(),
						Protocol: string(METEORA),
						Index:    uint(outerIndex * 256),
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
					err := p.setTxPoolInfo(progID, tx, inner)
					// tx, err := p.parseTransferTxInfo(progID, outerIndex, METEORA, innerSwaps)
					if err != nil {
						p.Log.Errorf("failed to parse tx info: %v, program: %s, signatures: %v", err, progID, p.txInfo.Signatures)
						return nil
					}
					return []SwapData{
						{
							Type: METEORA,
							Tx:   tx,
						},
					}
				}
			}
		}
	}

	if !isInner { // init liquidity
		outerInstriction := p.txInfo.Message.Instructions[outerIndex]
		inners := p.getInnerInstructions(outerIndex)
		switch {
		case progID.Equals(METEORA_POOLS_PROGRAM_ID) || progID.Equals(Meteora_Dynamic_Bonding_Curve_Program) || progID.Equals(METEORA_DAMM_V2) || progID.Equals(METEORA_PROGRAM_ID):

			discriminator := outerInstriction.Data[:8]
			if bytes.Equal(discriminator, meteora_pools_program.Instruction_Swap[:]) || bytes.Equal(meteoradlmmprogram.Instruction_Swap2[:], discriminator) {
				var innerSwaps []SwapData
				for _, innerInstruction := range inners {
					switch {
					case p.isTransferCheck(innerInstruction):
						transfer := p.processTransferCheck(innerInstruction)
						if transfer != nil {
							innerSwaps = append(innerSwaps, SwapData{Type: METEORA, Data: transfer})
						}
					case p.isTransfer(innerInstruction):
						transfer := p.processTransfer(innerInstruction)
						if transfer != nil {
							innerSwaps = append(innerSwaps, SwapData{Type: METEORA, Data: transfer})
						}
					}
					// if len(innerSwaps) >= 3 {
					// 	break
					// }
				}
				tx, err := p.parseTransferTxInfo(progID, outerIndex, METEORA, innerSwaps)
				if err != nil {
					p.Log.Errorf("failed to parse tx info: %v, program: %s, signatures: %v", err, progID, p.txInfo.Signatures)
					return nil
				}
				return []SwapData{
					{
						Type: METEORA,
						Tx:   tx,
					},
				}
			}
		}
	}
	return nil
}

func (p *Parser) processTransferCheck(instr solana.CompiledInstruction) *TransferCheck {
	amount := binary.LittleEndian.Uint64(instr.Data[1:9])

	transferData := &TransferCheck{
		Type: "transferChecked",
	}

	transferData.Info.Source = p.allAccountKeys[instr.Accounts[0]].String()
	transferData.Info.Destination = p.allAccountKeys[instr.Accounts[2]].String()
	transferData.Info.Mint = p.allAccountKeys[instr.Accounts[1]].String()
	transferData.Info.Authority = p.allAccountKeys[instr.Accounts[3]].String()

	transferData.Info.TokenAmount.Amount = fmt.Sprintf("%d", amount)
	transferData.Info.TokenAmount.Decimals = p.splDecimalsMap[transferData.Info.Mint]
	uiAmount := float64(amount) / math.Pow10(int(transferData.Info.TokenAmount.Decimals))
	transferData.Info.TokenAmount.UIAmount = uiAmount
	transferData.Info.TokenAmount.UIAmountString = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.9f", uiAmount), "0"), ".")

	return transferData
}
