package solanaswapgo

import (
	"encoding/json"
	"fmt"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mr-tron/base58"
)

var SharedAccountsRouteV2Discriminator = [8]byte{209, 152, 83, 147, 124, 254, 216, 233}

type JupiterSwapEvent struct {
	Amm          solana.PublicKey
	InputMint    solana.PublicKey
	InputAmount  uint64
	OutputMint   solana.PublicKey
	OutputAmount uint64
}

type JupiterSwapEventData struct {
	JupiterSwapEvent
	InputMintDecimals  uint8
	OutputMintDecimals uint8
}

var JupiterRouteEventDiscriminator = [16]byte{228, 69, 165, 46, 81, 203, 154, 29, 64, 198, 205, 232, 38, 8, 113, 226}

var AnchorSelfCPIDiscriminator = [8]byte{228, 69, 165, 46, 81, 203, 154, 29}
var SwapEventDiscriminator = [8]byte{64, 198, 205, 232, 38, 8, 113, 226}
var SwapsEventDiscriminator = [8]byte{152, 47, 78, 235, 192, 96, 110, 106}

type SwapEventV2 struct {
	InputMint    solana.PublicKey
	InputAmount  uint64
	OutputMint   solana.PublicKey
	OutputAmount uint64
}

func (p *Parser) processJupiterSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData

	isSharedAccountsV2 := false
	outerInstr := p.txInfo.Message.Instructions[instructionIndex]
	if len(outerInstr.Data) >= 8 {
		match := true
		for k := 0; k < 8; k++ {
			if outerInstr.Data[k] != SharedAccountsRouteV2Discriminator[k] {
				match = false
				break
			}
		}
		isSharedAccountsV2 = match
	}

	for _, innerInstructionSet := range p.txMeta.InnerInstructions {
		if innerInstructionSet.Index == uint16(instructionIndex) {
			last := 0
			inferredAMMs := p.collectAMMsFromInnerInstructions(innerInstructionSet)
			useSwapsEvent := isSharedAccountsV2 || p.hasUnknownAMMsInInnerInstructions(innerInstructionSet)

			// Collect AMM instructions between Jupiter events for pool info
			ammInstrs := p.collectAMMInstructionsBetweenEvents(innerInstructionSet)

			for i, innerInstruction := range innerInstructionSet.Instructions {
				solInstr := p.convertRPCToSolanaInstruction(innerInstruction)
				if p.isJupiterRouteEventInstruction(solInstr) {
					eventData, err := p.parseJupiterRouteEventInstruction(solInstr)
					if err != nil {
						p.Log.Errorf("error processing Jupiter trade event: %s", err)
					}
					if eventData != nil {
						tx := p.parseJupiterTxInfo(eventData, innerInstructionSet, last)
						if !p.setPoolInfoFromAMM(tx, ammInstrs, len(swaps)) {
							continue
						}
						swaps = append(swaps, SwapData{Type: JUPITER, Data: eventData, Tx: tx})
					}
					last = i
				} else if useSwapsEvent && p.isSwapsEventInstruction(solInstr) {
					eventDataList, err := p.parseSwapsEventInstruction(solInstr)
					if err != nil {
						p.Log.Errorf("error processing SwapsEvent: %s", err)
					}
				startIdx := len(swaps)
				for ei, eventData := range eventDataList {
					tx := p.parseJupiterTxInfo(eventData, innerInstructionSet, last)
					if ei < len(inferredAMMs) {
						tx.Amm = inferredAMMs[ei]
						tx.Protocol = protocolFromAMM(inferredAMMs[ei])
					}
					if !p.setPoolInfoFromAMM(tx, ammInstrs, startIdx+ei) {
						continue
					}
					swaps = append(swaps, SwapData{Type: JUPITER, Data: eventData, Tx: tx})
				}
					last = i
				}
			}
		}
	}
	return swaps
}

// collectAMMInstructionsBetweenEvents finds AMM instructions between Jupiter/DFlow events.
func (p *Parser) collectAMMInstructionsBetweenEvents(innerSet rpc.InnerInstruction) []*solana.CompiledInstruction {
	var ammInstrs []*solana.CompiledInstruction
	for _, inner := range innerSet.Instructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]
		if progID.Equals(JUPITER_PROGRAM_ID) || progID.Equals(DFLOW_AGGREGATOR_V4) {
			continue
		}
		if p.isUtilityProgram(progID) {
			continue
		}
		solInstr := p.convertRPCToSolanaInstruction(inner)
		if len(solInstr.Data) < 8 {
			continue
		}
		if solInstr.Data[0] == AnchorSelfCPIDiscriminator[0] {
			continue
		}
		ammInstrs = append(ammInstrs, &solInstr)
	}
	return ammInstrs
}

// setPoolInfoFromAMM calls setTxPoolInfo for the nth AMM instruction if available.
// Returns true if a valid AMM instruction was found, false otherwise.
func (p *Parser) setPoolInfoFromAMM(tx *TxInfo, ammInstrs []*solana.CompiledInstruction, idx int) bool {
	for i := idx; i < len(ammInstrs); i++ {
		ammInstr := ammInstrs[i]
		ammProgID := p.allAccountKeys[ammInstr.ProgramIDIndex]
		err := p.setTxPoolInfo(ammProgID, tx, *ammInstr)
		if err == nil {
			return true
		}
		// If this wasn't a valid swap instruction, try the next one
	}
	return false
}

func (p *Parser) collectAMMsFromInnerInstructions(innerSet rpc.InnerInstruction) []solana.PublicKey {
	var amms []solana.PublicKey
	for _, inner := range innerSet.Instructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]
		if progID.Equals(JUPITER_PROGRAM_ID) || progID.Equals(DFLOW_AGGREGATOR_V4) {
			continue
		}
		if p.isUtilityProgram(progID) {
			continue
		}
		solInstr := p.convertRPCToSolanaInstruction(inner)
		if len(solInstr.Data) < 8 {
			continue
		}
		if solInstr.Data[0] == AnchorSelfCPIDiscriminator[0] {
			continue
		}
		amms = append(amms, progID)
	}
	return amms
}

func (p *Parser) hasUnknownAMMsInInnerInstructions(innerSet rpc.InnerInstruction) bool {
	for _, inner := range innerSet.Instructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]
		if progID.Equals(JUPITER_PROGRAM_ID) || progID.Equals(DFLOW_AGGREGATOR_V4) {
			continue
		}
		if p.isUtilityProgram(progID) {
			continue
		}
		if p.isKnownAMM(progID) {
			continue
		}
		solInstr := p.convertRPCToSolanaInstruction(inner)
		if len(solInstr.Data) < 8 {
			continue
		}
		if solInstr.Data[0] == AnchorSelfCPIDiscriminator[0] {
			continue
		}
		return true
	}
	return false
}

func (p *Parser) isUtilityProgram(progID solana.PublicKey) bool {
	return progID.Equals(solana.TokenProgramID) ||
		progID.Equals(solana.Token2022ProgramID) ||
		progID.Equals(solana.SPLAssociatedTokenAccountProgramID) ||
		progID.Equals(solana.ComputeBudget) ||
		progID.Equals(solana.SystemProgramID) ||
		progID.Equals(solana.MemoProgramID) ||
		progID.Equals(PFEE_PROGRAM_ID)
}

// containsDCAProgram checks if the transaction contains the Jupiter DCA program.
func (p *Parser) containsDCAProgram() bool {
	for _, accountKey := range p.allAccountKeys {
		if accountKey.Equals(JUPITER_DCA_PROGRAM_ID) {
			return true
		}
	}
	return false
}

func (p *Parser) parseJupiterRouteEventInstruction(instruction solana.CompiledInstruction) (*JupiterSwapEventData, error) {
	decodedBytes, err := base58.Decode(instruction.Data.String())
	if err != nil {
		return nil, fmt.Errorf("error decoding instruction data: %s", err)
	}
	decoder := ag_binary.NewBorshDecoder(decodedBytes[16:])

	jupSwapEvent, err := handleJupiterRouteEvent(decoder)
	if err != nil {
		return nil, fmt.Errorf("error decoding jupiter swap event: %s", err)
	}

	inputMintDecimals, exists := p.splDecimalsMap[jupSwapEvent.InputMint.String()]
	if !exists {
		inputMintDecimals = 0
	}

	outputMintDecimals, exists := p.splDecimalsMap[jupSwapEvent.OutputMint.String()]
	if !exists {
		outputMintDecimals = 0
	}

	return &JupiterSwapEventData{
		JupiterSwapEvent:   *jupSwapEvent,
		InputMintDecimals:  inputMintDecimals,
		OutputMintDecimals: outputMintDecimals,
	}, nil
}

func handleJupiterRouteEvent(decoder *ag_binary.Decoder) (*JupiterSwapEvent, error) {
	var event JupiterSwapEvent
	if err := decoder.Decode(&event); err != nil {
		return nil, fmt.Errorf("error unmarshaling JupiterSwapEvent: %s", err)
	}
	return &event, nil
}

func (p *Parser) parseSwapsEventInstruction(instruction solana.CompiledInstruction) ([]*JupiterSwapEventData, error) {
	decodedBytes, err := base58.Decode(instruction.Data.String())
	if err != nil {
		return nil, fmt.Errorf("error decoding instruction data: %s", err)
	}
	decoder := ag_binary.NewBorshDecoder(decodedBytes[16:])

	var vecLen uint32
	if err := decoder.Decode(&vecLen); err != nil {
		return nil, fmt.Errorf("error reading SwapsEvent vec length: %s", err)
	}

	var events []*JupiterSwapEventData
	for i := uint32(0); i < vecLen; i++ {
		var v2 SwapEventV2
		if err := decoder.Decode(&v2); err != nil {
			return nil, fmt.Errorf("error decoding SwapEventV2[%d]: %s", i, err)
		}

		inputMintDecimals, exists := p.splDecimalsMap[v2.InputMint.String()]
		if !exists {
			inputMintDecimals = 0
		}
		outputMintDecimals, exists := p.splDecimalsMap[v2.OutputMint.String()]
		if !exists {
			outputMintDecimals = 0
		}

		events = append(events, &JupiterSwapEventData{
			JupiterSwapEvent: JupiterSwapEvent{
				Amm:          solana.PublicKey{},
				InputMint:    v2.InputMint,
				InputAmount:  v2.InputAmount,
				OutputMint:   v2.OutputMint,
				OutputAmount: v2.OutputAmount,
			},
			InputMintDecimals:  inputMintDecimals,
			OutputMintDecimals: outputMintDecimals,
		})
	}
	return events, nil
}

func (p *Parser) extractSPLDecimals() error {
	mintToDecimals := make(map[string]uint8)

	for _, accountInfo := range p.txMeta.PostTokenBalances {
		if !accountInfo.Mint.IsZero() {
			mintAddress := accountInfo.Mint.String()
			mintToDecimals[mintAddress] = uint8(accountInfo.UiTokenAmount.Decimals)
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

		mint := p.allAccountKeys[instr.Accounts[1]].String()
		if _, exists := mintToDecimals[mint]; !exists {
			mintToDecimals[mint] = 0
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

	// Add Native SOL if not present
	if _, exists := mintToDecimals[NATIVE_SOL_MINT_PROGRAM_ID.String()]; !exists {
		mintToDecimals[NATIVE_SOL_MINT_PROGRAM_ID.String()] = 9 // Native SOL has 9 decimal places
	}

	p.splDecimalsMap = mintToDecimals

	return nil
}

// parseJupiterEvents parses Jupiter swap events and returns a SwapInfo representing the entire route
func parseJupiterEvents(events []SwapData) (*SwapInfo, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events provided")
	}

	var firstSwap *JupiterSwapEventData
	var lastSwap *JupiterSwapEventData
	inputAmounts := make(map[string]uint64)
	outputAmounts := make(map[string]uint64)
	seenAMMs := make(map[string]bool)
	var amms []string

	for _, event := range events {
		if event.Type != JUPITER {
			continue
		}

		var jupiterEvent JupiterSwapEventData
		eventData, err := json.Marshal(event.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event data: %v", err)
		}

		if err := json.Unmarshal(eventData, &jupiterEvent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Jupiter event data: %v", err)
		}

		if firstSwap == nil {
			firstSwap = &jupiterEvent
		}
		lastSwap = &jupiterEvent

		inputKey := jupiterEvent.InputMint.String()
		outputKey := jupiterEvent.OutputMint.String()
		inputAmounts[inputKey] += jupiterEvent.InputAmount
		outputAmounts[outputKey] += jupiterEvent.OutputAmount

		if event.Tx != nil && !seenAMMs[event.Tx.Protocol] {
			seenAMMs[event.Tx.Protocol] = true
			amms = append(amms, event.Tx.Protocol)
		}
	}

	if firstSwap == nil || lastSwap == nil {
		return nil, fmt.Errorf("no valid Jupiter swaps found")
	}

	totalInputAmount := inputAmounts[firstSwap.InputMint.String()]
	totalOutputAmount := outputAmounts[firstSwap.OutputMint.String()]

	if len(amms) == 0 {
		amms = []string{string(JUPITER)}
	}

	swapInfo := &SwapInfo{
		AMMs:             amms,
		TokenInMint:      firstSwap.InputMint,
		TokenInAmount:    totalInputAmount,
		TokenInDecimals:  firstSwap.InputMintDecimals,
		TokenOutMint:     firstSwap.OutputMint,
		TokenOutAmount:   totalOutputAmount,
		TokenOutDecimals: firstSwap.OutputMintDecimals,
	}

	return swapInfo, nil
}

func (p *Parser) parseJupiterTxInfo(eventData *JupiterSwapEventData, instr rpc.InnerInstruction, last int) *TxInfo {
	tx := &TxInfo{}
	tx.Type = TxTypeSwap
	tx.Amm = eventData.Amm
	tx.InputMint = eventData.InputMint
	tx.OutputMint = eventData.OutputMint
	tx.InputAmount = eventData.InputAmount
	tx.OutputAmount = eventData.OutputAmount
	tx.InputMintDecimals = eventData.InputMintDecimals
	tx.OutputMintDecimals = eventData.OutputMintDecimals
	tx.Protocol = protocolFromAMM(eventData.Amm)
	tx.Owner = *p.txInfo.Message.Signers().Last()
	tx.Router = p.allAccountKeys[p.txInfo.Message.Instructions[instr.Index].ProgramIDIndex]
	tx.Index = uint(instr.Index*256) + uint(last)
	return tx
}

func protocolFromAMM(amm solana.PublicKey) string {
	if amm.IsZero() {
		return string(JUPITER)
	}
	return ProtocolName(amm)
}

func (p *Parser) extractAccountPostBalance() error {
	p.postBalance = make(map[uint16]*rpc.TokenBalance)
	for _, accountInfo := range p.txMeta.PostTokenBalances {
		if !accountInfo.Mint.IsZero() {
			if accountInfo.UiTokenAmount == nil || accountInfo.UiTokenAmount.Amount == "" {
				continue
			}
			p.postBalance[accountInfo.AccountIndex] = &accountInfo
		}
	}
	return nil
}
