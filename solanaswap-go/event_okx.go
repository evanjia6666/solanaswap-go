package solanaswapgo

import (
	"bytes"
	"encoding/binary"
	"fmt"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

var (
	OKX_SWAP_DISCRIMINATOR                 = [8]byte{248, 198, 158, 145, 225, 117, 135, 200}
	OKX_SWAP2_DISCRIMINATOR                = [8]byte{65, 75, 63, 76, 235, 91, 91, 136}
	OKX_COMMISSION_SPL_SWAP2_DISCRIMINATOR = [8]byte{173, 131, 78, 38, 150, 165, 123, 15}
	OKX_SWAP3_DISCRIMINATOR                = [8]byte{19, 44, 130, 148, 72, 56, 44, 238}
	OKX_SWAP_V3_DISCRIMINATOR              = [8]byte{240, 224, 38, 33, 176, 31, 241, 175}
	OKX_SWAP_TOB_V3_DISCRIMINATOR          = [8]byte{14, 191, 44, 246, 142, 225, 224, 157}
)

type OKXDex uint8

const (
	OKXDexSplTokenSwap        OKXDex = 0
	OKXDexStableSwap          OKXDex = 1
	OKXDexWhirlpool           OKXDex = 2
	OKXDexMeteoraDynamicpool  OKXDex = 3
	OKXDexRaydiumSwap         OKXDex = 4
	OKXDexRaydiumStableSwap   OKXDex = 5
	OKXDexRaydiumClmmSwap     OKXDex = 6
	OKXDexAldrinExchangeV1    OKXDex = 7
	OKXDexAldrinExchangeV2    OKXDex = 8
	OKXDexLifinityV1          OKXDex = 9
	OKXDexLifinityV2          OKXDex = 10
	OKXDexRaydiumClmmSwapV2   OKXDex = 11
	OKXDexFluxBeam            OKXDex = 12
	OKXDexMeteoraDlmm         OKXDex = 13
	OKXDexRaydiumCpmmSwap     OKXDex = 14
	OKXDexOpenBookV2          OKXDex = 15
	OKXDexWhirlpoolV2         OKXDex = 16
	OKXDexPhoenix             OKXDex = 17
	OKXDexObricV2             OKXDex = 18
	OKXDexSanctumAddLiq       OKXDex = 19
	OKXDexSanctumRemoveLiq    OKXDex = 20
	OKXDexSanctumNonWsolSwap  OKXDex = 21
	OKXDexSanctumWsolSwap     OKXDex = 22
	OKXDexPumpfunBuy          OKXDex = 23
	OKXDexPumpfunSell         OKXDex = 24
	OKXDexStabbleSwap         OKXDex = 25
	OKXDexSanctumRouter       OKXDex = 26
	OKXDexZerofi              OKXDex = 33
	OKXDexPumpfunammBuy       OKXDex = 34
	OKXDexPumpfunammSell      OKXDex = 35
	OKXDexMeteoraDlmmSwap2    OKXDex = 46
	OKXDexMeteoraDAMMV2       OKXDex = 47
	OKXDexManifest            OKXDex = 59
	OKXDexByreal              OKXDex = 60
	OKXDexPancakeSwapV3Swap   OKXDex = 61
	OKXDexPancakeSwapV3SwapV2 OKXDex = 62
	OKXDexHumidifi            OKXDex = 65
	OKXDexPumpfunammBuy2      OKXDex = 112
	OKXDexPumpfunammSell2     OKXDex = 113
)

func (d OKXDex) String() string {
	switch d {
	case OKXDexWhirlpool, OKXDexWhirlpoolV2:
		return "Whirlpools Program"
	case OKXDexRaydiumSwap, OKXDexRaydiumCpmmSwap:
		return "Raydium"
	case OKXDexRaydiumClmmSwap, OKXDexRaydiumClmmSwapV2:
		return "Raydium Concentrated Liquidity"
	case OKXDexMeteoraDynamicpool:
		return "Meteora Dynamic Pool"
	case OKXDexMeteoraDlmm, OKXDexMeteoraDlmmSwap2, OKXDexMeteoraDAMMV2:
		return "Meteora_DLMM_Program"
	case OKXDexSplTokenSwap:
		return "SPL Token Swap"
	case OKXDexStableSwap, OKXDexStabbleSwap:
		return "Stable Swap"
	case OKXDexRaydiumStableSwap:
		return "Raydium Stable Swap"
	case OKXDexAldrinExchangeV1:
		return "Aldrin V1"
	case OKXDexAldrinExchangeV2:
		return "Aldrin V2"
	case OKXDexManifest:
		return "Manifest"
	case OKXDexHumidifi:
		return "HumidiFi"
	case OKXDexZerofi:
		return "ZeroFi"
	case OKXDexPhoenix:
		return "Phoenix"
	case OKXDexPancakeSwapV3Swap, OKXDexPancakeSwapV3SwapV2:
		return "Pancake Swap"
	case OKXDexPumpfunBuy, OKXDexPumpfunSell, OKXDexPumpfunammBuy, OKXDexPumpfunammSell, OKXDexPumpfunammBuy2, OKXDexPumpfunammSell2:
		return string(PUMPSWAP)
	case OKXDexOpenBookV2:
		return "OpenBook V2"
	case OKXDexByreal:
		return "Byreal CLMM"
	default:
		return fmt.Sprintf("OKX DEX %d", d)
	}
}

type OKXLabs2SwapEvent struct {
	Dex       OKXDex
	AmountIn  uint64
	AmountOut uint64
}

func parseOKXLabs2SwapEvent(data []byte) (*OKXLabs2SwapEvent, error) {
	decoder := ag_binary.NewBorshDecoder(data)
	var event OKXLabs2SwapEvent
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

func transferAmount(data []byte) uint64 {
	if len(data) < 9 {
		return 0
	}
	return binary.LittleEndian.Uint64(data[1:9])
}

func amountsMatch(a, b uint64) bool {
	if a == b {
		return true
	}
	diff := a
	if b > a {
		diff = b - a
	} else {
		diff = a - b
	}
	return diff <= 2
}

func (p *Parser) processOKXLabs2SwapEvents(instructionIndex int) []SwapData {
	var swaps []SwapData
	innerInstructions := p.getInnerInstructions(instructionIndex)
	if len(innerInstructions) == 0 {
		return swaps
	}

	var swapEvents []struct {
		data  *OKXLabs2SwapEvent
		index int
	}
	for i, inner := range innerInstructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]
		if !progID.Equals(OKX_LABS_2_PROGRAM_ID) {
			continue
		}
		data, err := base58.Decode(inner.Data.String())
		if err != nil || len(data) < 16 {
			continue
		}
		if !bytes.Equal(data[:8], AnchorSelfCPIDiscriminator[:]) {
			continue
		}
		if bytes.Equal(data[8:16], SwapEventDiscriminator[:]) {
			event, err := parseOKXLabs2SwapEvent(data[16:])
			if err == nil {
				swapEvents = append(swapEvents, struct {
					data  *OKXLabs2SwapEvent
					index int
				}{event, i})
			}
		}
	}

	for _, se := range swapEvents {
		tx := &TxInfo{}
		tx.Type = TxTypeSwap
		tx.Amm = solana.PublicKey{}
		tx.Protocol = se.data.Dex.String()
		tx.InputAmount = se.data.AmountIn
		tx.OutputAmount = se.data.AmountOut
		tx.Router = OKX_LABS_2_PROGRAM_ID

		prevBoundary := -1
		for j := se.index - 1; j >= 0; j-- {
			if p.allAccountKeys[innerInstructions[j].ProgramIDIndex].Equals(OKX_LABS_2_PROGRAM_ID) {
				prevBoundary = j
				break
			}
		}

		var inputMint, outputMint string
		var ammInstr *solana.CompiledInstruction
		for j := prevBoundary + 1; j < se.index; j++ {
			inner := innerInstructions[j]
			if inner.ProgramIDIndex >= uint16(len(p.allAccountKeys)) {
				continue
			}
			innerProgID := p.allAccountKeys[inner.ProgramIDIndex]

			if innerProgID.Equals(solana.TokenProgramID) || innerProgID.Equals(solana.Token2022ProgramID) {
				if len(inner.Data) == 0 || (inner.Data[0] != 3 && inner.Data[0] != 12) {
					continue
				}
				amt := transferAmount(inner.Data)

				mint := p.splTokenInfoMap[p.allAccountKeys[inner.Accounts[0]].String()].Mint
				if mint == "" {
					mint = p.splTokenInfoMap[p.allAccountKeys[inner.Accounts[1]].String()].Mint
				}
				if mint == "" {
					continue
				}

				if amountsMatch(amt, se.data.AmountIn) {
					inputMint = mint
				}
				if amountsMatch(amt, se.data.AmountOut) {
					outputMint = mint
				}
			} else if ammInstr == nil && !innerProgID.Equals(solana.SystemProgramID) {
				if len(inner.Data) < 8 || !bytes.Equal(inner.Data[:8], AnchorSelfCPIDiscriminator[:]) {
					cp := inner
					ammInstr = &cp
				}
			}
		}

		if inputMint != "" {
			tx.InputMint = solana.MustPublicKeyFromBase58(inputMint)
			if dec, ok := p.splDecimalsMap[inputMint]; ok {
				tx.InputMintDecimals = dec
			}
		}
		if outputMint != "" {
			tx.OutputMint = solana.MustPublicKeyFromBase58(outputMint)
			if dec, ok := p.splDecimalsMap[outputMint]; ok {
				tx.OutputMintDecimals = dec
			}
		}

		if ammInstr != nil {
			ammProgID := p.allAccountKeys[ammInstr.ProgramIDIndex]
			if err := p.setTxPoolInfo(ammProgID, tx, *ammInstr); err != nil {
				continue
			}
			// Keep the layout-resolved protocol name (e.g. "GoonFi V2") unless
			// it was not set — in that case fall back to the OKX dex label.
			if tx.Protocol == "" {
				tx.Protocol = se.data.Dex.String()
			}
			tx.Amm = ammProgID
		}

		tx.Owner = *p.txInfo.Message.Signers().Last()
		tx.Index = uint(instructionIndex*256) + uint(se.index)
		swaps = append(swaps, SwapData{Type: OKX, Data: se.data, Tx: tx})
	}

	if len(swaps) == 0 {
		return p.processOKXRouterSwaps(instructionIndex)
	}
	return swaps
}

func (p *Parser) processOKXSwaps(instructionIndex int) []SwapData {
	// p.Log.Infof("starting okx swap parsing for instruction index: %d", instructionIndex)

	parentInstruction := p.txInfo.Message.Instructions[instructionIndex]
	programID := p.allAccountKeys[parentInstruction.ProgramIDIndex]

	if !programID.Equals(OKX_LABS_1_PROGRAM_ID) {
		p.Log.Warnf("instruction %d skipped: not okx dex router program", instructionIndex)
		return nil
	}

	if len(parentInstruction.Data) < 8 {
		p.Log.Warnf("instruction %d skipped: data too short (%d)", instructionIndex, len(parentInstruction.Data))
		return nil
	}

	decodedBytes, err := base58.Decode(parentInstruction.Data.String())
	if err != nil {
		p.Log.Errorf("failed to decode okx swap instruction %d: %s", instructionIndex, err)
		return nil
	}

	discriminator := decodedBytes[:8]
	// p.Log.Infof("decoded okx swap instruction %d with discriminator: %x", instructionIndex, discriminator)

	switch {
	case bytes.Equal(discriminator, OKX_SWAP_DISCRIMINATOR[:]):
		//	p.Log.Infof("processing okx swap type: okx_swap for instruction %d", instructionIndex)
		return p.processOKXRouterSwaps(instructionIndex)

	case bytes.Equal(discriminator, OKX_SWAP2_DISCRIMINATOR[:]):
		//	p.Log.Infof("processing okx swap type: okx_swap2 for instruction %d", instructionIndex)
		return p.processOKXRouterSwaps(instructionIndex)

	case bytes.Equal(discriminator, OKX_COMMISSION_SPL_SWAP2_DISCRIMINATOR[:]):
		//	p.Log.Infof("processing okx swap type: okx_commission_spl_swap2 for instruction %d", instructionIndex)
		return p.processOKXRouterSwaps(instructionIndex)

	case bytes.Equal(discriminator, OKX_SWAP3_DISCRIMINATOR[:]):
		//	p.Log.Infof("processing okx swap type: okx_swap3 for instruction %d", instructionIndex)
		return p.processOKXRouterSwaps(instructionIndex)

	case bytes.Equal(discriminator, OKX_SWAP_V3_DISCRIMINATOR[:]):
		//	p.Log.Infof("processing okx swap type: okx_swap_v3 for instruction %d", instructionIndex)
		return p.processOKXRouterSwaps(instructionIndex)
	case bytes.Equal(discriminator, OKX_SWAP_TOB_V3_DISCRIMINATOR[:]):
		//	p.Log.Infof("processing okx swap type: okx_swap_tob_v3 for instruction %d", instructionIndex)
		return p.processOKXRouterSwaps(instructionIndex)
	default:
		//	p.Log.Warnf("unknown okx swap discriminator %x for instruction %d", discriminator, instructionIndex)
		swaps := p.processOKXRouterSwaps(instructionIndex)
		if len(swaps) > 0 {
			//	p.Log.Infof("successfully processed %d swaps with unknown discriminator", len(swaps))
			return swaps
		}
		p.Log.Warnf("no swaps found with unknown discriminator %x", discriminator)
		return nil
	}
}

func (p *Parser) processOKXRouterSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData
	seen := make(map[string]bool)
	processedProtocols := make(map[SwapType]bool)

	innerInstructions := p.getInnerInstructions(instructionIndex)
	// p.Log.Infof("processing okx router swaps for instruction %d: %d inner instructions", instructionIndex, len(innerInstructions))
	if len(innerInstructions) == 0 {
		p.Log.Warnf("no inner instructions for instruction %d", instructionIndex)
		return swaps
	}

	for idx, inner := range innerInstructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]

		switch {
		case progID.Equals(RAYDIUM_V4_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_CPMM_PROGRAM_ID) ||
			// progID.Equals(RAYDIUM_AMM_ROUTER_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID):
			if processedProtocols[RAYDIUM] {
				continue
			}
			if raydSwaps := p.processRaydSwaps(progID, instructionIndex, idx, &inner, true); len(raydSwaps) > 0 {
				// for _, swap := range raydSwaps {
				// 	key := getSwapKey(swap)
				// 	if !seen[key] {
				swaps = append(swaps, raydSwaps...)
				// 		seen[key] = true
				// 	}
				// }
				processedProtocols[RAYDIUM] = true
			}

		case progID.Equals(ORCA_PROGRAM_ID):
			if processedProtocols[ORCA] {
				continue
			}
			if orcaSwaps := p.processOrcaSwaps(instructionIndex, &inner); len(orcaSwaps) > 0 {
				for _, swap := range orcaSwaps {
					key := getSwapKey(swap)
					if !seen[key] {
						swaps = append(swaps, swap)
						seen[key] = true
					}
				}
				processedProtocols[ORCA] = true
			}

		case progID.Equals(METEORA_PROGRAM_ID) ||
			progID.Equals(METEORA_POOLS_PROGRAM_ID) ||
			progID.Equals(METEORA_DLMM_PROGRAM_ID) ||
			progID.Equals(BYREAL_CLMM_PROGRAM_ID) ||
			progID.Equals(METEORA_DAMM_V2):
			// if processedProtocols[METEORA] {
			// 	continue
			// }
			if meteoraSwaps := p.processMeteoraSwaps(progID, instructionIndex, idx, true); len(meteoraSwaps) > 0 {
				// for _, swap := range meteoraSwaps {
				// 	key := getSwapKey(swap)
				// 	if !seen[key] {
				swaps = append(swaps, meteoraSwaps...)
				// 		seen[key] = true
				// 	}
				// }
				// processedProtocols[METEORA] = true
			}

		case progID.Equals(PUMP_FUN_PROGRAM_ID):
			if processedProtocols[PUMP_FUN] {
				continue
			}
			if pumpfunSwaps := p.processPumpfunSwaps(instructionIndex); len(pumpfunSwaps) > 0 {
				for _, swap := range pumpfunSwaps {
					key := getSwapKey(swap)
					if !seen[key] {
						swaps = append(swaps, swap)
						seen[key] = true
					}
				}
				processedProtocols[PUMP_FUN] = true
			}

		case progID.Equals(PUMPFUN_AMM_PROGRAM_ID):
			if processedProtocols[PUMP_FUN] {
				continue
			}
			pumpfunAMMSwaps := p.processPumpfunAMMSwaps(instructionIndex, true)
			swaps = append(swaps, pumpfunAMMSwaps...)
			//; len(pumpfunAMMSwaps) > 0 {
			// 	for _, swap := range pumpfunAMMSwaps {
			// 		key := getSwapKey(swap)
			// 		if !seen[key] {
			// 			swaps = append(swaps, swap)
			// 			seen[key] = true
			// 		}
			// 	}
			// }
			processedProtocols[PUMP_FUN] = true
		}

	}

	// p.Log.Infof("processed okx router swaps: %d unique swaps", len(swaps))
	return swaps
}

func getSwapKey(swap SwapData) string {
	switch data := swap.Data.(type) {
	case *TransferCheck:
		return fmt.Sprintf("%s-%s-%s", swap.Type, data.Info.TokenAmount.Amount, data.Info.Mint)
	case *TransferData:
		return fmt.Sprintf("%s-%d-%s", swap.Type, data.Info.Amount, data.Mint)
	default:
		return fmt.Sprintf("%s-%v", swap.Type, data)
	}
}
