package solanaswapgo

import (
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

// Manifest swap discriminators
const (
	manifestSwapDiscriminator  = 4  // Swap instruction
	manifestSwapV2Discriminator = 13 // SwapV2 instruction (separate owner)
)

// ManifestSwapParams mirrors the Borsh-serialized SwapParams struct
type ManifestSwapParams struct {
	InAtoms    uint64
	OutAtoms   uint64
	IsBaseIn   bool
	IsExactIn  bool
}

func (p *Parser) processManifestSwaps(instructionIndex int, isInner bool) []SwapData {
	if isInner {
		// Called from processRouterSwaps: Manifest is an inner instruction
		// Find the Manifest inner instruction within the router's inner instructions
		innerInstructions := p.getInnerInstructions(instructionIndex)
		for i, innerInstr := range innerInstructions {
			pID := p.allAccountKeys[innerInstr.ProgramIDIndex]
			if !pID.Equals(MANIFEST_PROGRAM_ID) {
				continue
			}
			if len(innerInstr.Data) < 19 {
				continue
			}
			discriminator := innerInstr.Data[0]
			if discriminator != manifestSwapDiscriminator && discriminator != manifestSwapV2Discriminator {
				continue
			}
			router := p.allAccountKeys[p.txInfo.Message.Instructions[instructionIndex].ProgramIDIndex]
			return p.processManifestSwapsFromInner(instructionIndex, i, &innerInstr, router)
		}
		return nil
	}

	// isInner = false: Manifest is the outer instruction
	outerInstr := p.txInfo.Message.Instructions[instructionIndex]
	if !p.allAccountKeys[outerInstr.ProgramIDIndex].Equals(MANIFEST_PROGRAM_ID) {
		return nil
	}

	if len(outerInstr.Data) < 19 {
		return nil
	}

	discriminator := outerInstr.Data[0]
	if discriminator != manifestSwapDiscriminator && discriminator != manifestSwapV2Discriminator {
		return nil
	}

	// Parse SwapParams from instruction data (Borsh serialized)
	params := parseManifestSwapParams(outerInstr.Data[1:])
	if params == nil {
		return nil
	}

	// Extract accounts
	accounts := outerInstr.Accounts
	if len(accounts) < 8 {
		return nil
	}

	// Determine base/quote mints
	var baseMint, quoteMint solana.PublicKey

	tokenProgramBaseIdx := 8
	if len(accounts) > tokenProgramBaseIdx {
		tokenProgramBase := p.allAccountKeys[accounts[tokenProgramBaseIdx]]
		if tokenProgramBase.Equals(Token2022ProgramID) && len(accounts) > 9 {
			baseMint = p.allAccountKeys[accounts[9]]
		}
	}

	tokenProgramQuoteIdx := 9
	if !baseMint.IsZero() {
		tokenProgramQuoteIdx = 10
	}
	if len(accounts) > tokenProgramQuoteIdx {
		tokenProgramQuote := p.allAccountKeys[accounts[tokenProgramQuoteIdx]]
		if tokenProgramQuote.Equals(Token2022ProgramID) && len(accounts) > tokenProgramQuoteIdx+1 {
			quoteMint = p.allAccountKeys[accounts[tokenProgramQuoteIdx+1]]
		} else if !baseMint.IsZero() && !tokenProgramQuote.Equals(p.allAccountKeys[accounts[8]]) && len(accounts) > tokenProgramQuoteIdx+1 {
			quoteMint = p.allAccountKeys[accounts[tokenProgramQuoteIdx+1]]
		}
	}

	if baseMint.IsZero() && len(accounts) > 6 {
		vaultBase := p.allAccountKeys[accounts[6]].String()
		if info, ok := p.splTokenInfoMap[vaultBase]; ok && info.Mint != "" {
			baseMint = solana.MustPublicKeyFromBase58(info.Mint)
		}
	}
	if quoteMint.IsZero() && len(accounts) > 7 {
		vaultQuote := p.allAccountKeys[accounts[7]].String()
		if info, ok := p.splTokenInfoMap[vaultQuote]; ok && info.Mint != "" {
			quoteMint = solana.MustPublicKeyFromBase58(info.Mint)
		}
	}

	if baseMint.IsZero() || quoteMint.IsZero() {
		return nil
	}

	var inputMint, outputMint solana.PublicKey
	var inputAmount, outputAmount uint64
	var inputDecimals, outputDecimals uint8

	if params.IsBaseIn {
		inputMint = baseMint
		outputMint = quoteMint
		inputAmount = params.InAtoms
		outputAmount = params.OutAtoms
		inputDecimals = p.splDecimalsMap[baseMint.String()]
		outputDecimals = p.splDecimalsMap[quoteMint.String()]
	} else {
		inputMint = quoteMint
		outputMint = baseMint
		inputAmount = params.InAtoms
		outputAmount = params.OutAtoms
		inputDecimals = p.splDecimalsMap[quoteMint.String()]
		outputDecimals = p.splDecimalsMap[baseMint.String()]
	}

	// SwapV2 may have OutAtoms=0; derive output from subsequent Transfer instructions
	if outputAmount == 0 {
		outputAmount = p.extractManifestOutputFromOuterTransfers(instructionIndex, accounts, inputMint, outputMint)
	}

	tx := &TxInfo{
		Type:               TxTypeSwap,
		Router:             MANIFEST_PROGRAM_ID,
		Amm:                MANIFEST_PROGRAM_ID,
		Protocol:           "Manifest",
		Owner:              *p.txInfo.Message.Signers().Last(),
		Index:              uint(instructionIndex * 256),
		InputMint:          inputMint,
		InputAmount:        inputAmount,
		InputMintDecimals:  inputDecimals,
		OutputMint:         outputMint,
		OutputAmount:       outputAmount,
		OutputMintDecimals: outputDecimals,
		Pool:               p.allAccountKeys[accounts[2]],
		PoolIn:             p.allAccountKeys[accounts[6]],
		PoolOut:            p.allAccountKeys[accounts[7]],
	}

	return []SwapData{{
		Type: MANIFEST,
		Data: nil,
		Tx:   tx,
	}}
}

func parseManifestSwapParams(data []byte) *ManifestSwapParams {
	if len(data) < 18 {
		return nil
	}

	params := &ManifestSwapParams{
		InAtoms:   binary.LittleEndian.Uint64(data[0:8]),
		OutAtoms:  binary.LittleEndian.Uint64(data[8:16]),
		IsBaseIn:  data[16] != 0,
		IsExactIn: data[17] != 0,
	}

	return params
}

// processManifestSwapsFromInner parses Manifest swap from an inner instruction within a router
func (p *Parser) processManifestSwapsFromInner(outerIdx, innerIdx int, innerInstr *solana.CompiledInstruction, router solana.PublicKey) []SwapData {
	if len(innerInstr.Data) < 19 {
		return nil
	}

	discriminator := innerInstr.Data[0]
	if discriminator != manifestSwapDiscriminator && discriminator != manifestSwapV2Discriminator {
		return nil
	}

	params := parseManifestSwapParams(innerInstr.Data[1:])
	if params == nil {
		return nil
	}

	accounts := innerInstr.Accounts
	if len(accounts) < 8 {
		return nil
	}

	var baseMint, quoteMint solana.PublicKey

	tokenProgramBaseIdx := 8
	if len(accounts) > tokenProgramBaseIdx {
		tokenProgramBase := p.allAccountKeys[accounts[tokenProgramBaseIdx]]
		if tokenProgramBase.Equals(Token2022ProgramID) && len(accounts) > 9 {
			baseMint = p.allAccountKeys[accounts[9]]
		}
	}

	tokenProgramQuoteIdx := 9
	if !baseMint.IsZero() {
		tokenProgramQuoteIdx = 10
	}
	if len(accounts) > tokenProgramQuoteIdx {
		tokenProgramQuote := p.allAccountKeys[accounts[tokenProgramQuoteIdx]]
		if tokenProgramQuote.Equals(Token2022ProgramID) && len(accounts) > tokenProgramQuoteIdx+1 {
			quoteMint = p.allAccountKeys[accounts[tokenProgramQuoteIdx+1]]
		} else if !baseMint.IsZero() && !tokenProgramQuote.Equals(p.allAccountKeys[accounts[8]]) && len(accounts) > tokenProgramQuoteIdx+1 {
			quoteMint = p.allAccountKeys[accounts[tokenProgramQuoteIdx+1]]
		}
	}

	if baseMint.IsZero() && len(accounts) > 6 {
		vaultBase := p.allAccountKeys[accounts[6]].String()
		if info, ok := p.splTokenInfoMap[vaultBase]; ok && info.Mint != "" {
			baseMint = solana.MustPublicKeyFromBase58(info.Mint)
		}
	}
	if quoteMint.IsZero() && len(accounts) > 7 {
		vaultQuote := p.allAccountKeys[accounts[7]].String()
		if info, ok := p.splTokenInfoMap[vaultQuote]; ok && info.Mint != "" {
			quoteMint = solana.MustPublicKeyFromBase58(info.Mint)
		}
	}

	if baseMint.IsZero() || quoteMint.IsZero() {
		return nil
	}

	var inputMint, outputMint solana.PublicKey
	var inputAmount, outputAmount uint64
	var inputDecimals, outputDecimals uint8

	if params.IsBaseIn {
		inputMint = baseMint
		outputMint = quoteMint
		inputAmount = params.InAtoms
		outputAmount = params.OutAtoms
		inputDecimals = p.splDecimalsMap[baseMint.String()]
		outputDecimals = p.splDecimalsMap[quoteMint.String()]
	} else {
		inputMint = quoteMint
		outputMint = baseMint
		inputAmount = params.InAtoms
		outputAmount = params.OutAtoms
		inputDecimals = p.splDecimalsMap[quoteMint.String()]
		outputDecimals = p.splDecimalsMap[baseMint.String()]
	}

	// SwapV2 may have OutAtoms=0; derive output from subsequent Transfer instructions
	if outputAmount == 0 {
		outputAmount = p.extractManifestOutputFromTransfers(outerIdx, innerIdx, accounts)
	}

	tx := &TxInfo{
		Type:               TxTypeSwap,
		Router:             router,
		Amm:                MANIFEST_PROGRAM_ID,
		Protocol:           "Manifest",
		Owner:              *p.txInfo.Message.Signers().Last(),
		Index:              uint(outerIdx*256) + uint(innerIdx),
		InputMint:          inputMint,
		InputAmount:        inputAmount,
		InputMintDecimals:  inputDecimals,
		OutputMint:         outputMint,
		OutputAmount:       outputAmount,
		OutputMintDecimals: outputDecimals,
		Pool:               p.allAccountKeys[accounts[2]],
		PoolIn:             p.allAccountKeys[accounts[6]],
		PoolOut:            p.allAccountKeys[accounts[7]],
	}

	return []SwapData{{
		Type: MANIFEST,
		Data: nil,
		Tx:   tx,
	}}
}

// extractManifestOutputFromTransfers scans inner instructions after the Manifest
// swap to find the Transfer/TransferChecked that sends output tokens to the user.
func (p *Parser) extractManifestOutputFromTransfers(outerIdx, manifestInnerIdx int, accounts []uint16) uint64 {
	innerInstructions := p.getInnerInstructions(outerIdx)
	if innerInstructions == nil {
		return 0
	}

	quoteVaultIdx := accounts[7]

	for i := manifestInnerIdx + 1; i < len(innerInstructions); i++ {
		inst := innerInstructions[i]
		progID := p.allAccountKeys[inst.ProgramIDIndex]

		if !progID.Equals(solana.TokenProgramID) && !progID.Equals(Token2022ProgramID) {
			continue
		}

		if len(inst.Data) < 9 {
			continue
		}

		opcode := inst.Data[0]
		if opcode != 3 && opcode != 12 {
			continue
		}

		var amount uint64
		var srcIdx uint16

		if opcode == 3 {
			if len(inst.Accounts) < 2 {
				continue
			}
			amount = binary.LittleEndian.Uint64(inst.Data[1:9])
			srcIdx = inst.Accounts[0]
		} else {
			if len(inst.Accounts) < 3 {
				continue
			}
			amount = binary.LittleEndian.Uint64(inst.Data[1:9])
			srcIdx = inst.Accounts[0]
		}

		if srcIdx == quoteVaultIdx && amount > 0 {
			return amount
		}
	}

	return 0
}

func (p *Parser) extractManifestOutputFromOuterTransfers(manifestOuterIdx int, accounts []uint16, inputMint, outputMint solana.PublicKey) uint64 {
	quoteVaultIdx := accounts[7]

	for i := manifestOuterIdx + 1; i < len(p.txInfo.Message.Instructions); i++ {
		inst := p.txInfo.Message.Instructions[i]
		progID := p.allAccountKeys[inst.ProgramIDIndex]

		if !progID.Equals(solana.TokenProgramID) && !progID.Equals(Token2022ProgramID) {
			continue
		}

		if len(inst.Data) < 9 {
			continue
		}

		opcode := inst.Data[0]
		if opcode != 3 && opcode != 12 {
			continue
		}

		var amount uint64
		var srcIdx uint16

		if opcode == 3 {
			if len(inst.Accounts) < 2 {
				continue
			}
			amount = binary.LittleEndian.Uint64(inst.Data[1:9])
			srcIdx = inst.Accounts[0]
		} else {
			if len(inst.Accounts) < 3 {
				continue
			}
			amount = binary.LittleEndian.Uint64(inst.Data[1:9])
			srcIdx = inst.Accounts[0]
		}

		if srcIdx == quoteVaultIdx && amount > 0 {
			return amount
		}
	}

	return 0
}
