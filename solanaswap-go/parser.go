package solanaswapgo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

const (
	PROTOCOL_RAYDIUM = "raydium"
	PROTOCOL_ORCA    = "orca"
	PROTOCOL_METEORA = "meteora"
	PROTOCOL_PUMPFUN = "pumpfun"
)

type TokenTransfer struct {
	mint     string
	amount   uint64
	decimals uint8
}

type Parser struct {
	txMeta          *rpc.TransactionMeta
	txInfo          *solana.Transaction
	allAccountKeys  solana.PublicKeySlice
	splTokenInfoMap map[string]TokenInfo
	splDecimalsMap  map[string]uint8
	Log             *logrus.Logger

	postBalance map[uint16]*rpc.TokenBalance
}

var (
	swapDiscriminator = map[string]bool{
		calculateDiscriminator("global:swap"):                   true,
		calculateDiscriminator("global:swap_exact_out"):         true,
		calculateDiscriminator("global:swap_exact_in"):          true,
		calculateDiscriminator("global:swap_base_input"):        true,
		calculateDiscriminator("global:swap_base_output"):       true,
		calculateDiscriminator("global:swap_v2"):                true,
		calculateDiscriminator("global:swap_with_price_impact"): true,
		calculateDiscriminator("global:swap_exact_amount_in"):   true,
		calculateDiscriminator("global:sell_token"):             true,
		calculateDiscriminator("global:swap_with_partner"):      true,
		calculateDiscriminator("global:redeem_v0"):              true,
		calculateDiscriminator("global:sell"):                   true, // pumpfun AMM
		calculateDiscriminator("global:buy"):                    true, // pumpfun AMM
	}

	removeDiscriminator = map[string]bool{
		calculateDiscriminator("global:remove_liquidity_by_range"): true,
		calculateDiscriminator("global:remove_liquidity"):          true,
		calculateDiscriminator("global:remove_all_liquidity"):      true,
		calculateDiscriminator("global:decrease_liquidity"):        true,
		calculateDiscriminator("global:decrease_liquidity_v2"):     true,
		calculateDiscriminator("global:withdraw"):                  true,
	}

	addDiscriminator = map[string]bool{
		calculateDiscriminator("global:add_liquidity"):             true,
		calculateDiscriminator("global:add_liquidity_by_weight"):   true,
		calculateDiscriminator("global:add_liquidity_by_strategy"): true,
		calculateDiscriminator("global:increase_liquidity"):        true,
		calculateDiscriminator("global:increase_liquidity_v2"):     true,
		calculateDiscriminator("global:deposit"):                   true,
		calculateDiscriminator("global:initialize"):                true,
	}
)

func NewTransactionParser(tx *solana.Transaction, txMeta *rpc.TransactionMeta) (*Parser, error) {
	return NewTransactionParserFromTransaction(tx, txMeta)
}

func NewParser(tx *rpc.GetTransactionResult) (*Parser, error) {
	txInfo, err := tx.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return NewTransactionParserFromTransaction(txInfo, tx.Meta)
}

func NewTransactionParserFromTransaction(tx *solana.Transaction, txMeta *rpc.TransactionMeta) (*Parser, error) {
	allAccountKeys := append(tx.Message.AccountKeys, txMeta.LoadedAddresses.Writable...)
	allAccountKeys = append(allAccountKeys, txMeta.LoadedAddresses.ReadOnly...)

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	parser := &Parser{
		txMeta:         txMeta,
		txInfo:         tx,
		allAccountKeys: allAccountKeys,
		Log:            log,
	}

	if err := parser.extractSPLTokenInfo(); err != nil {
		return nil, fmt.Errorf("failed to extract SPL Token Addresses: %w", err)
	}

	if err := parser.extractSPLDecimals(); err != nil {
		return nil, fmt.Errorf("failed to extract SPL decimals: %w", err)
	}

	if err := parser.extractAccountPostBalance(); err != nil {
		return nil, fmt.Errorf("failed to extract SPL decimals: %w", err)
	}

	return parser, nil
}

type SwapData struct {
	Type SwapType
	Data interface{}
	Tx   *TxInfo
}

func (p *Parser) ParseTransaction() ([]SwapData, error) {
	var parsedSwaps []SwapData

	skip := false
	for i, outerInstruction := range p.txInfo.Message.Instructions {
		progID := p.allAccountKeys[outerInstruction.ProgramIDIndex]
		switch {
		case progID.Equals(JUPITER_PROGRAM_ID) || progID.Equals(DFLOW_AGGREGATOR_V4):
			skip = true
			parsedSwaps = append(parsedSwaps, p.processJupiterSwaps(i)...)
		case progID.Equals(MOONSHOT_PROGRAM_ID):
			skip = true
			parsedSwaps = append(parsedSwaps, p.processMoonshotSwaps()...)
		case progID.Equals(BANANA_GUN_PROGRAM_ID) ||
			progID.Equals(MINTECH_PROGRAM_ID) ||
			progID.Equals(BLOOM_PROGRAM_ID) ||
			progID.Equals(NOVA_PROGRAM_ID) ||
			progID.Equals(MAESTRO_PROGRAM_ID):
			if innerSwaps := p.processRouterSwaps(i); len(innerSwaps) > 0 {
				parsedSwaps = append(parsedSwaps, innerSwaps...)
			}
		case progID.Equals(OKX_DEX_ROUTER_PROGRAM_ID):
			skip = true
			parsedSwaps = append(parsedSwaps, p.processOKXSwaps(i)...)
		}
	}
	if skip {
		return parsedSwaps, nil
	}

	for i, outerInstruction := range p.txInfo.Message.Instructions {
		progID := p.allAccountKeys[outerInstruction.ProgramIDIndex]
		switch {
		case progID.Equals(RAYDIUM_V4_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_CPMM_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_AMM_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_LAUNCHLAB_PROGRAM_ID) ||
			progID.Equals(solana.MustPublicKeyFromBase58("AP51WLiiqTdbZfgyRMs35PsZpdmLuPDdHYmrB23pEtMU")):
			parsedSwaps = append(parsedSwaps, p.processRaydSwaps(i)...)
		case progID.Equals(ORCA_PROGRAM_ID):
			parsedSwaps = append(parsedSwaps, p.processOrcaSwaps(i)...)
		case progID.Equals(METEORA_PROGRAM_ID) || progID.Equals(METEORA_POOLS_PROGRAM_ID) || progID.Equals(METEORA_DLMM_PROGRAM_ID):
			parsedSwaps = append(parsedSwaps, p.processMeteoraSwaps(i)...)
		case progID.Equals(PUMPFUN_AMM_PROGRAM_ID):
			parsedSwaps = append(parsedSwaps, p.processPumpfunAMMSwaps(i)...)
		case progID.Equals(PUMP_FUN_PROGRAM_ID) ||
			progID.Equals(solana.MustPublicKeyFromBase58("BSfD6SHZigAfDWSjzD5Q41jw8LmKwtmjskPH9XW1mrRW")):
			parsedSwaps = append(parsedSwaps, p.processPumpfunSwaps(i)...)
		}
	}

	return parsedSwaps, nil
}

type SwapInfo struct {
	Signers    []solana.PublicKey
	Signatures []solana.Signature
	AMMs       []string
	Timestamp  time.Time

	TokenInMint     solana.PublicKey
	TokenInAmount   uint64
	TokenInDecimals uint8

	TokenOutMint     solana.PublicKey
	TokenOutAmount   uint64
	TokenOutDecimals uint8
}

func (p *Parser) ProcessSwapData(swapDatas []SwapData) (*SwapInfo, *TxInfo, error) {
	if len(swapDatas) == 0 {
		return nil, nil, fmt.Errorf("no swap data provided")
	}
	var tx *TxInfo
	if len(swapDatas) == 1 && swapDatas[0].Tx != nil {
		tx = swapDatas[0].Tx
	}

	swapInfo := &SwapInfo{
		Signatures: p.txInfo.Signatures,
	}

	if p.containsDCAProgram() {
		swapInfo.Signers = []solana.PublicKey{p.allAccountKeys[2]}
	} else {
		swapInfo.Signers = []solana.PublicKey{p.allAccountKeys[0]}
	}

	jupiterSwaps := make([]SwapData, 0)
	pumpfunSwaps := make([]SwapData, 0)
	otherSwaps := make([]SwapData, 0)

	for _, swapData := range swapDatas {
		switch swapData.Type {
		case JUPITER:
			jupiterSwaps = append(jupiterSwaps, swapData)
		case PUMP_FUN:
			pumpfunSwaps = append(pumpfunSwaps, swapData)
		default:
			otherSwaps = append(otherSwaps, swapData)
		}
	}

	if len(jupiterSwaps) > 0 {
		jupiterInfo, err := parseJupiterEvents(jupiterSwaps)
		if err != nil {
			return nil, tx, fmt.Errorf("failed to parse Jupiter events: %w", err)
		}

		swapInfo.TokenInMint = jupiterInfo.TokenInMint
		swapInfo.TokenInAmount = jupiterInfo.TokenInAmount
		swapInfo.TokenInDecimals = jupiterInfo.TokenInDecimals
		swapInfo.TokenOutMint = jupiterInfo.TokenOutMint
		swapInfo.TokenOutAmount = jupiterInfo.TokenOutAmount
		swapInfo.TokenOutDecimals = jupiterInfo.TokenOutDecimals
		swapInfo.AMMs = jupiterInfo.AMMs

		return swapInfo, tx, nil
	}

	if len(pumpfunSwaps) > 0 {
		switch data := pumpfunSwaps[0].Data.(type) {
		case *PumpfunTradeEvent:
			if data.IsBuy {
				swapInfo.TokenInMint = NATIVE_SOL_MINT_PROGRAM_ID
				swapInfo.TokenInAmount = data.SolAmount
				swapInfo.TokenInDecimals = 9
				swapInfo.TokenOutMint = data.Mint
				swapInfo.TokenOutAmount = data.TokenAmount
				swapInfo.TokenOutDecimals = p.splDecimalsMap[data.Mint.String()]
			} else {
				swapInfo.TokenInMint = data.Mint
				swapInfo.TokenInAmount = data.TokenAmount
				swapInfo.TokenInDecimals = p.splDecimalsMap[data.Mint.String()]
				swapInfo.TokenOutMint = NATIVE_SOL_MINT_PROGRAM_ID
				swapInfo.TokenOutAmount = data.SolAmount
				swapInfo.TokenOutDecimals = 9
			}
			swapInfo.AMMs = append(swapInfo.AMMs, string(pumpfunSwaps[0].Type))
			swapInfo.Timestamp = time.Unix(int64(data.Timestamp), 0)
			return swapInfo, tx, nil
		default:
			otherSwaps = append(otherSwaps, pumpfunSwaps...)
		}
	}

	if len(otherSwaps) > 0 {
		var uniqueTokens []TokenTransfer
		seenTokens := make(map[string]bool)

		for _, swapData := range otherSwaps {
			transfer := getTransferFromSwapData(swapData)
			if transfer != nil && !seenTokens[transfer.mint] {
				uniqueTokens = append(uniqueTokens, *transfer)
				seenTokens[transfer.mint] = true
			}
		}

		if len(uniqueTokens) >= 2 {
			inputTransfer := uniqueTokens[0]
			outputTransfer := uniqueTokens[len(uniqueTokens)-1]

			seenInputs := make(map[string]bool)
			seenOutputs := make(map[string]bool)
			var totalInputAmount uint64 = 0
			var totalOutputAmount uint64 = 0

			for _, swapData := range otherSwaps {
				transfer := getTransferFromSwapData(swapData)
				if transfer == nil {
					continue
				}

				amountStr := fmt.Sprintf("%d-%s", transfer.amount, transfer.mint)
				if transfer.mint == inputTransfer.mint && !seenInputs[amountStr] {
					totalInputAmount += transfer.amount
					seenInputs[amountStr] = true
				}
				if transfer.mint == outputTransfer.mint && !seenOutputs[amountStr] {
					totalOutputAmount += transfer.amount
					seenOutputs[amountStr] = true
				}
			}

			swapInfo.TokenInMint = solana.MustPublicKeyFromBase58(inputTransfer.mint)
			swapInfo.TokenInAmount = totalInputAmount
			swapInfo.TokenInDecimals = inputTransfer.decimals
			swapInfo.TokenOutMint = solana.MustPublicKeyFromBase58(outputTransfer.mint)
			swapInfo.TokenOutAmount = totalOutputAmount
			swapInfo.TokenOutDecimals = outputTransfer.decimals

			seenAMMs := make(map[string]bool)
			for _, swapData := range otherSwaps {
				if !seenAMMs[string(swapData.Type)] {
					swapInfo.AMMs = append(swapInfo.AMMs, string(swapData.Type))
					seenAMMs[string(swapData.Type)] = true
				}
			}

			swapInfo.Timestamp = time.Now()

			return swapInfo, tx, nil
		}
	}

	return nil, nil, fmt.Errorf("no valid swaps found")
}

func getTransferFromSwapData(swapData SwapData) *TokenTransfer {
	switch data := swapData.Data.(type) {
	case *TransferData:
		return &TokenTransfer{
			mint:     data.Mint,
			amount:   data.Info.Amount,
			decimals: data.Decimals,
		}
	case *TransferCheck:
		amt, err := strconv.ParseUint(data.Info.TokenAmount.Amount, 10, 64)
		if err != nil {
			return nil
		}
		return &TokenTransfer{
			mint:     data.Info.Mint,
			amount:   amt,
			decimals: data.Info.TokenAmount.Decimals,
		}
	}
	return nil
}

func (p *Parser) processRouterSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData

	innerInstructions := p.getInnerInstructions(instructionIndex)
	if len(innerInstructions) == 0 {
		return swaps
	}

	processedProtocols := make(map[string]bool)

	for _, inner := range innerInstructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]

		switch {
		case (progID.Equals(RAYDIUM_V4_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_CPMM_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_AMM_PROGRAM_ID) ||
			progID.Equals(RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID)) && !processedProtocols[PROTOCOL_RAYDIUM]:
			processedProtocols[PROTOCOL_RAYDIUM] = true
			if raydSwaps := p.processRaydSwaps(instructionIndex); len(raydSwaps) > 0 {
				swaps = append(swaps, raydSwaps...)
			}

		case progID.Equals(ORCA_PROGRAM_ID) && !processedProtocols[PROTOCOL_ORCA]:
			processedProtocols[PROTOCOL_ORCA] = true
			if orcaSwaps := p.processOrcaSwaps(instructionIndex); len(orcaSwaps) > 0 {
				swaps = append(swaps, orcaSwaps...)
			}

		case (progID.Equals(METEORA_PROGRAM_ID) ||
			progID.Equals(METEORA_POOLS_PROGRAM_ID) ||
			progID.Equals(METEORA_DLMM_PROGRAM_ID)) && !processedProtocols[PROTOCOL_METEORA]:
			processedProtocols[PROTOCOL_METEORA] = true
			if meteoraSwaps := p.processMeteoraSwaps(instructionIndex); len(meteoraSwaps) > 0 {
				swaps = append(swaps, meteoraSwaps...)
			}

		case progID.Equals(PUMPFUN_AMM_PROGRAM_ID) && !processedProtocols[PROTOCOL_PUMPFUN]:
			processedProtocols[PROTOCOL_PUMPFUN] = true
			if pumpfunAMMSwaps := p.processPumpfunAMMSwaps(instructionIndex); len(pumpfunAMMSwaps) > 0 {
				swaps = append(swaps, pumpfunAMMSwaps...)
			}

		case (progID.Equals(PUMP_FUN_PROGRAM_ID) ||
			progID.Equals(solana.MustPublicKeyFromBase58("BSfD6SHZigAfDWSjzD5Q41jw8LmKwtmjskPH9XW1mrRW"))) && !processedProtocols[PROTOCOL_PUMPFUN]:
			processedProtocols[PROTOCOL_PUMPFUN] = true
			if pumpfunSwaps := p.processPumpfunSwaps(instructionIndex); len(pumpfunSwaps) > 0 {
				swaps = append(swaps, pumpfunSwaps...)
			}
		}
	}

	return swaps
}

func (p *Parser) getInnerInstructions(index int) []solana.CompiledInstruction {
	if p.txMeta == nil || p.txMeta.InnerInstructions == nil {
		return nil
	}

	for _, inner := range p.txMeta.InnerInstructions {
		if inner.Index == uint16(index) {
			result := make([]solana.CompiledInstruction, len(inner.Instructions))
			for i, inst := range inner.Instructions {
				result[i] = p.convertRPCToSolanaInstruction(inst)
			}
			return result
		}
	}

	return nil
}

type TxInfo struct {
	Type               string
	Amm                solana.PublicKey
	InputMint          solana.PublicKey
	InputAmount        uint64
	InputMintDecimals  uint8
	OutputMint         solana.PublicKey
	OutputAmount       uint64
	OutputMintDecimals uint8
	Pool               solana.PublicKey
	PoolIn             solana.PublicKey
	PoolOut            solana.PublicKey
	PoolInAmount       *big.Int
	PoolOutAmount      *big.Int
	Owner              solana.PublicKey
	Router             solana.PublicKey
	Index              uint
	Protocol           string
}

func (p *Parser) setTxPoolInfo(progID solana.PublicKey, tx *TxInfo, instruction solana.CompiledInstruction) (err error) {
	var discriminatorLen = 8
	var discriminatorWhiteList [][]byte
	var poolAccountIndex, poolInAccountIndex, poolOutAccountIndex uint16
	var protocol string
	pid := progID.String()
	tx.Type = TxTypeSwap
	switch {
	case progID.Equals(RAYDIUM_V4_PROGRAM_ID):
		poolAccountIndex = 1
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		//fromAccountIndex = 16
		//toAccountIndex = 2
		if len(instruction.Accounts) == 18 {
			poolInAccountIndex = 5
			poolOutAccountIndex = 6
			//fromAccountIndex = 17
			//toAccountIndex = 2
		}
		protocol = string(RAYDIUM)
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},  // add
			{3},  // add
			{4},  // remove
			{9},  // swap
			{11}, // swap
		}
		if len(instruction.Data) > 0 {
			if instruction.Data[0] == 1 {
				tx.Type = TxTypeAdd
				poolAccountIndex = 4
				poolInAccountIndex = 10
				poolOutAccountIndex = 11
			} else if instruction.Data[0] == 3 {
				tx.Type = TxTypeAdd
				poolAccountIndex = 1
				poolInAccountIndex = 6
				poolOutAccountIndex = 7
			} else if instruction.Data[0] == 4 {
				tx.Type = TxTypeRemove
				poolAccountIndex = 1
				poolInAccountIndex = 6
				poolOutAccountIndex = 7
			}
		}
	case progID.Equals(ORCA_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 4
		poolOutAccountIndex = 6
		//fromAccountIndex = 1
		//toAccountIndex = 2
		protocol = string(ORCA)
		if len(instruction.Data) >= discriminatorLen {
			if _, ok := removeDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 0
				poolInAccountIndex = uint16(len(instruction.Accounts)) - 4
				poolOutAccountIndex = uint16(len(instruction.Accounts)) - 3
			} else if _, ok := addDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeAdd
				poolAccountIndex = 0
				poolInAccountIndex = uint16(len(instruction.Accounts)) - 4
				poolOutAccountIndex = uint16(len(instruction.Accounts)) - 3
			}
		}
	case progID.Equals(RAYDIUM_CPMM_PROGRAM_ID):
		poolAccountIndex = 3
		poolInAccountIndex = 6
		poolOutAccountIndex = 7
		//fromAccountIndex = 0
		//toAccountIndex = 1
		protocol = string(RAYDIUM)
		if len(instruction.Data) >= discriminatorLen {
			if _, ok := removeDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 2
				poolInAccountIndex = 6
				poolOutAccountIndex = 7
			} else if _, ok := addDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeAdd
				poolAccountIndex = 2
				poolInAccountIndex = 6
				poolOutAccountIndex = 7
				if string(instruction.Data[:discriminatorLen]) == calculateDiscriminator("global:initialize") {
					poolAccountIndex = 3
					poolInAccountIndex = 10
					poolOutAccountIndex = 11
				}
			}
		}
	case progID.Equals(RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		//fromAccountIndex = 0
		//toAccountIndex = 2
		protocol = string(RAYDIUM)
		if len(instruction.Data) >= discriminatorLen {
			if _, ok := removeDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 3
				poolInAccountIndex = 5
				poolOutAccountIndex = 6
			} else if _, ok := addDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeAdd
				poolAccountIndex = 2
				poolInAccountIndex = 9
				poolOutAccountIndex = 10
			}
		}
	case progID.Equals(METEORA_PROGRAM_ID):
		poolAccountIndex = 0
		poolInAccountIndex = 2
		poolOutAccountIndex = 3
		//fromAccountIndex = 0
		//toAccountIndex = 10
		protocol = string(METEORA)
		if len(instruction.Data) >= discriminatorLen {
			liquidity := false
			if _, ok := removeDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 1
				poolInAccountIndex = 5
				poolOutAccountIndex = 6
				liquidity = true
			} else if _, ok := addDiscriminator[string(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeAdd
				poolAccountIndex = 1
				poolInAccountIndex = 5
				poolOutAccountIndex = 6
				liquidity = true
			}
			if liquidity && tx.OutputAmount == 0 {
				if tx.InputMint.Equals(p.allAccountKeys[instruction.Accounts[7]]) {
					tx.OutputMint = p.allAccountKeys[instruction.Accounts[8]]
					tx.OutputMintDecimals = p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[6]].String()].Decimals
				} else {
					tx.OutputMint = p.allAccountKeys[instruction.Accounts[7]]
					tx.OutputMintDecimals = p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[5]].String()].Decimals
				}
			}
		}
	case pid == "swapFpHZwjELNnjvThjajtiVmkz3yPQEHjLtka2fwHW":
		poolAccountIndex = 6
		poolInAccountIndex = 3
		poolOutAccountIndex = 4
		if len(instruction.Accounts) == 15 {
			poolAccountIndex = 8
			poolInAccountIndex = 5
			poolOutAccountIndex = 6
		}
		protocol = "StableWeighted"
	case pid == "SoLFiHG9TfgtdUXUjWAxi3LtvYuFyDLVhBWxdMZxyCe":
		poolAccountIndex = 1
		poolInAccountIndex = 2
		poolOutAccountIndex = 3
		protocol = "SolFi"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{7},
		}
	case pid == "2wT8Yq49kHgDzXuPxZSaeLaH1qbmGXtEyPy64bL7aD3c":
		poolAccountIndex = 1
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "Lifinity Swap V2"
	case pid == "9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Orca Token Swap V2"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case progID.Equals(METEORA_PROGRAM_ID):
		poolAccountIndex = 0
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "Meteora Pools Program"
	case pid == "swapNyd8XiQwJ6ianp9snpu4brUqFxadzvHebnAXjJZ":
		poolAccountIndex = 6
		poolInAccountIndex = 3
		poolOutAccountIndex = 4
		if len(instruction.Accounts) == 15 {
			poolAccountIndex = 8
			poolInAccountIndex = 5
			poolOutAccountIndex = 6
		}
		protocol = "stabble Stable Swap"
	case progID.Equals(PHOENIX_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 6
		poolOutAccountIndex = 7
		protocol = "Phoenix"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{0},
		}
	case pid == "DEXYosS6oEGvk8uCDayvwEZz4qEyDJRf9nFgYCaqPMTm":
		poolAccountIndex = 2
		poolInAccountIndex = 3
		poolOutAccountIndex = 4
		protocol = "1Dex"
	case pid == "H8W3ctz92svYg6mkn1UtGfu2aQr2fnUFHM1RhScEtQDt":
		poolAccountIndex = 2
		poolInAccountIndex = 4
		poolOutAccountIndex = 6
		protocol = "Cropper"
	case pid == "HyaB3W9q6XdA5xwpU4XnSZV94htfmbmqJXZcEbRaJutt":
		poolAccountIndex = 1
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "Invariant"
	case pid == "SSwpkEEcbUqx4vtoEByFjSkhKdCT862DNVb52nZg1UZ":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Saber Stable Swap"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "SSwapUtytfBdBn1b9NUGG6foMVPtcWgpRU32HToDUZr":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Saros"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "FLUXubRmkEi2q6K3Y9kBPg9248ggaZVsoSFhtJHSrm1X":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Fluxbeam"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "Gswppe6ERWKpUTXvRPfXdzHhiCyJvLadVvXGfdpBqcE1":
		poolAccountIndex = 1
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Guac"
	case pid == "BSwp6bEBihVLdqJRKGgzjcGLHkcTuzmSo1TQkHepzH8p":
		poolAccountIndex = 1
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "BonkSwap"
	case pid == "DSwpgjMvXhtGn6BsbqmacdBZyfLj6jSWf3HJpdJtmg6N":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "DexlabSwap"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "CURVGoZn8zycx6FXwwevgBTB2gVvdbGTEpvMJDbgs2t4":
		poolAccountIndex = 0
		poolInAccountIndex = 3
		poolOutAccountIndex = 4
		protocol = "Aldrin"
	case pid == "AMM55ShdkoGRB5jVYPjWziwk8m5MpwyDgsMWHaMSQWH6":
		poolAccountIndex = 0
		poolInAccountIndex = 3
		poolOutAccountIndex = 4
		protocol = "Aldrin"
	case pid == "DjVE6JNiYqPL2QXyCUUh8rNjHrbz9hXHNYt99MQ59qw1":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Orca Token Swap"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Swap Program"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "5jnapfrAN47UYkLkEf7HnprPPBCQLvkYWGZDeKkaP5hv":
		poolAccountIndex = 4
		poolInAccountIndex = 7
		poolOutAccountIndex = 8
		protocol = "DaoFun"
	case pid == "CLMM9tUoggJu2wagPkkqs9eFG4BWhVBZWkP1qv3Sp7tR":
		poolAccountIndex = 1
		poolInAccountIndex = 6
		poolOutAccountIndex = 7
		protocol = "Crema Finance Program"
	case pid == "Dooar9JkhdZ7J3LHN3A7YCuoGRUggXhQaG4kijfLGU2j":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "StepN DOOAR Swap"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case pid == "treaf4wWBBty3fHdyBpo35Mz84M8k3heKXmjmi9vFt5":
		poolAccountIndex = 0
		poolInAccountIndex = 3
		poolOutAccountIndex = 4
		protocol = "Helium Treasury Management"
	case pid == "PSwapMdSai8tjrEXcxFeQth87xC4rRsa4VA5mhGhXkP":
		poolAccountIndex = 0
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Penguin Finance"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},
		}
	case progID.Equals(PUMPFUN_AMM_PROGRAM_ID):
		poolAccountIndex = 0
		poolInAccountIndex = 7
		poolOutAccountIndex = 8
		protocol = string(PUMP_FUN)
	case progID.Equals(METEORA_DAMM_V2):
		poolAccountIndex = 1
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Meteora DAMM V2"
	case progID.Equals(ZEROFI):
		poolAccountIndex = 0
		poolInAccountIndex = 2
		poolOutAccountIndex = 4
		protocol = "ZeroFi"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{6},
		}
	default:
		err = errors.New("unknown progID")
		log.Println("unknown progID", p.txInfo.Signatures, progID)
		return
	}

	if len(instruction.Data) < discriminatorLen {
		err = errors.New("invalid instruction data length")
		return
	}

	discriminator := hex.EncodeToString(instruction.Data[:discriminatorLen])
	var m map[string]bool
	if len(discriminatorWhiteList) > 0 {
		m = map[string]bool{}
		for _, d := range discriminatorWhiteList {
			m[hex.EncodeToString(d)] = true
		}
	} else {
		m = swapDiscriminator
	}
	if _, ok := m[discriminator]; ok {
	} else if _, ok := removeDiscriminator[discriminator]; ok {
	} else if _, ok := addDiscriminator[discriminator]; ok {
	} else {
		err = errors.New("discriminator unmatched")
		log.Println(err, p.txInfo.Signatures, progID, hex.EncodeToString(instruction.Data), discriminator)
		return
	}

	accLen := len(instruction.Accounts)
	if accLen < int(poolAccountIndex) || accLen <= int(poolOutAccountIndex) || accLen < int(poolInAccountIndex) {
		err = fmt.Errorf("account index out of range %d/%d-%d-%d", len(instruction.Accounts), poolAccountIndex, poolInAccountIndex, poolOutAccountIndex)
		return
	}

	poolInAccountIndex = instruction.Accounts[poolInAccountIndex]
	poolOutAccountIndex = instruction.Accounts[poolOutAccountIndex]

	poolInBalance, ok := p.postBalance[poolInAccountIndex]
	if !ok {
		err = errors.New("no postBalance for account")
		return
	}
	poolOutBalance, ok := p.postBalance[poolOutAccountIndex]
	if !ok {
		err = errors.New("no postBalance for account")
		return
	}

	if poolInBalance.Mint.Equals(tx.OutputMint) {
		poolInAccountIndex, poolOutAccountIndex = poolOutAccountIndex, poolInAccountIndex
		poolInBalance, poolOutBalance = poolOutBalance, poolInBalance
	}
	if !poolInBalance.Mint.Equals(tx.InputMint) {
		err = errors.New("no inputMint for account")
		return
	}
	if !poolOutBalance.Mint.Equals(tx.OutputMint) {
		err = errors.New("no outputMint for account")
		return
	}

	tx.Pool = p.allAccountKeys[instruction.Accounts[poolAccountIndex]]
	tx.PoolIn = p.allAccountKeys[poolInAccountIndex]
	tx.PoolOut = p.allAccountKeys[poolOutAccountIndex]

	if a, err := decimal.NewFromString(poolInBalance.UiTokenAmount.Amount); err == nil {
		tx.PoolInAmount = a.BigInt()
	}

	if a, err := decimal.NewFromString(poolOutBalance.UiTokenAmount.Amount); err == nil {
		tx.PoolOutAmount = a.BigInt()
	}

	tx.Protocol = protocol
	return
}

func calculateDiscriminator(instructionName string) string {
	hash := sha256.Sum256([]byte(instructionName))
	return hex.EncodeToString(hash[:8])
}
