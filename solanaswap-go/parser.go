package solanaswapgo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/orca/orca_whirlpool"
	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/pumpfun"
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

// 新增此类 router 时只需把 Program ID 加进这个 slice，无需修改 ParseTransaction 的 switch。

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
		calculateDiscriminator("global:swap"):                    true,
		calculateDiscriminator("global:swap_exact_out"):          true,
		calculateDiscriminator("global:swap_exact_in"):           true,
		calculateDiscriminator("global:swap_base_input"):         true,
		calculateDiscriminator("global:swap_base_output"):        true,
		calculateDiscriminator("global:swap_v2"):                 true,
		calculateDiscriminator("global:swap_with_price_impact"):  true,
		calculateDiscriminator("global:swap_exact_amount_in"):    true,
		calculateDiscriminator("global:sell_token"):              true,
		calculateDiscriminator("global:swap_with_partner"):       true,
		calculateDiscriminator("global:redeem_v0"):               true,
		calculateDiscriminator("global:sell"):                    true, // pumpfun AMM
		calculateDiscriminator("global:buy"):                     true, // pumpfun AMM
		calculateDiscriminator("global:buy_exact_quote_in"):      true, // pumpfun AMM
		calculateDiscriminator("global:sell_exact_in"):           true, // pumpfun AMM
		calculateDiscriminator("global:swap2"):                   true, // meteora dlmm
		calculateDiscriminator("global:swap_exact_out2"):         true, // meteora dlmm
		calculateDiscriminator("global:swap_with_price_impact2"): true, // meteora dlmm
		calculateDiscriminator("global:route_v2"):                true, // raydium cl (via jupiter)
		calculateDiscriminator("global:swap_router_base_in"):     true, // raydium cl / pancakeswap
	}

	removeDiscriminator = map[string]bool{
		calculateDiscriminator("global:remove_liquidity_by_range"):    true,
		calculateDiscriminator("global:remove_liquidity_by_range2"):   true, // meteora dlmm
		calculateDiscriminator("global:remove_liquidity"):             true,
		calculateDiscriminator("global:remove_liquidity2"):            true, // meteora dlmm
		calculateDiscriminator("global:remove_all_liquidity"):         true,
		calculateDiscriminator("global:remove_balance_liquidity"):     true, // meteora pools
		calculateDiscriminator("global:remove_liquidity_single_side"): true, // meteora pools
		calculateDiscriminator("global:decrease_liquidity"):           true,
		calculateDiscriminator("global:decrease_liquidity_v2"):        true,
		calculateDiscriminator("global:decrease_liquidity_v3"):        true,
		calculateDiscriminator("global:withdraw"):                     true,
		calculateDiscriminator("global:close_position"):               true,
		calculateDiscriminator("global:collect_protocol_fee"):         true,
	}

	addDiscriminator = map[string]bool{
		calculateDiscriminator("global:add_liquidity"):              true,
		calculateDiscriminator("global:add_liquidity2"):             true, // meteora dlmm
		calculateDiscriminator("global:add_liquidity_by_weight"):    true,
		calculateDiscriminator("global:add_liquidity_by_strategy"):  true,
		calculateDiscriminator("global:add_liquidity_by_strategy2"): true, // meteora dlmm
		calculateDiscriminator("global:add_balance_liquidity"):      true, // meteora pools
		calculateDiscriminator("global:add_imbalance_liquidity"):    true, // meteora pools
		calculateDiscriminator("global:increase_liquidity"):         true,
		calculateDiscriminator("global:increase_liquidity_v2"):      true,
		calculateDiscriminator("global:deposit"):                    true,
		calculateDiscriminator("global:initialize"):                 true,
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
		pr, ok := lookupParser(progID)
		if !ok {
			continue
		}
		if pr.Kind() != KindAMM {
			skip = true
			parsedSwaps = append(parsedSwaps, pr.ParseOuter(p, i)...)
		}
	}
	if skip {
		return p.filterInvalidSwaps(parsedSwaps), nil
	}

	for i, outerInstruction := range p.txInfo.Message.Instructions {
		progID := p.allAccountKeys[outerInstruction.ProgramIDIndex]
		pr, ok := lookupParser(progID)
		if !ok || pr.Kind() != KindAMM {
			continue
		}
		parsedSwaps = append(parsedSwaps, pr.ParseOuter(p, i)...)
	}

	return p.filterInvalidSwaps(parsedSwaps), nil
}

// filterInvalidSwaps is a final safeguard: it drops any SwapData whose Tx carries the
// System Program address (11111111111111111111111111111111) in a key address field.
// Such a value means a pool/mint/vault account failed to resolve, so the leg is unsafe
// to return. Legs without a Tx (event-only data) are passed through untouched.
func (p *Parser) filterInvalidSwaps(swaps []SwapData) []SwapData {
	filtered := make([]SwapData, 0, len(swaps))
	for _, s := range swaps {
		if s.Tx != nil && txHasSystemProgramAddress(s.Tx) {
			p.Log.Warnf("dropping swap leg with System Program address, protocol=%s sig=%v", s.Tx.Protocol, p.txInfo.Signatures)
			continue
		}
		// For a direct swap (no aggregator/bot router), Router is the zero value, which
		// serializes to the System Program address. Default it to the AMM so callers don't
		// see an ambiguous "11111111111111111111111111111111" router.
		if s.Tx != nil && s.Tx.Router.IsZero() {
			s.Tx.Router = s.Tx.Amm
		}
		filtered = append(filtered, s)
	}
	return filtered
}

// txHasSystemProgramAddress reports whether any key address field of tx equals the
// System Program ID, which signals an unresolved account.
func txHasSystemProgramAddress(tx *TxInfo) bool {
	for _, addr := range []solana.PublicKey{
		tx.Amm, tx.InputMint, tx.OutputMint, tx.Pool, tx.PoolIn, tx.PoolOut,
	} {
		if addr.Equals(solana.SystemProgramID) {
			return true
		}
	}
	return false
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
	moonshotSwaps := make([]SwapData, 0)
	otherSwaps := make([]SwapData, 0)

	for _, swapData := range swapDatas {
		switch swapData.Type {
		case JUPITER:
			jupiterSwaps = append(jupiterSwaps, swapData)
		case PUMP_FUN:
			pumpfunSwaps = append(pumpfunSwaps, swapData)
		case MOONSHOT:
			moonshotSwaps = append(moonshotSwaps, swapData)
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

	if len(moonshotSwaps) > 0 {
		if data, ok := moonshotSwaps[0].Data.(*MoonshotTradeInstructionWithMint); ok {
			if data.TradeType == TradeTypeBuy {
				swapInfo.TokenInMint = NATIVE_SOL_MINT_PROGRAM_ID
				swapInfo.TokenInAmount = data.CollateralAmount
				swapInfo.TokenInDecimals = 9
				swapInfo.TokenOutMint = data.Mint
				swapInfo.TokenOutAmount = data.TokenAmount
				swapInfo.TokenOutDecimals = p.splDecimalsMap[data.Mint.String()]
			} else {
				swapInfo.TokenInMint = data.Mint
				swapInfo.TokenInAmount = data.TokenAmount
				swapInfo.TokenInDecimals = p.splDecimalsMap[data.Mint.String()]
				swapInfo.TokenOutMint = NATIVE_SOL_MINT_PROGRAM_ID
				swapInfo.TokenOutAmount = data.CollateralAmount
				swapInfo.TokenOutDecimals = 9
			}
			swapInfo.AMMs = append(swapInfo.AMMs, string(moonshotSwaps[0].Type))
			swapInfo.Timestamp = time.Now()
			return swapInfo, tx, nil
		}
	}

	if len(otherSwaps) > 0 {
		// Handle tx-based swaps (Data nil but Tx present) — e.g. PumpFun AMM, ZeroFi
		var firstTxBasedIdx int = -1
		for i, sd := range otherSwaps {
			if sd.Tx != nil && sd.Data == nil {
				if firstTxBasedIdx == -1 {
					firstTxBasedIdx = i
				}
			}
		}
		if firstTxBasedIdx != -1 {
			// Collect all tx-based swaps in ORIGINAL order (from swapDatas)
			var txSwaps []SwapData
			for _, sd := range swapDatas {
				if sd.Tx != nil && sd.Data == nil {
					txSwaps = append(txSwaps, sd)
				}
			}
			if len(txSwaps) > 0 {
				// Build input/output mint sets
				inputMints := make(map[string]bool)
				outputMints := make(map[string]bool)
				for _, sd := range txSwaps {
					inputMints[sd.Tx.InputMint.String()] = true
					outputMints[sd.Tx.OutputMint.String()] = true
				}
				var startSwap, endSwap *SwapData
				for i := range txSwaps {
					sd := &txSwaps[i]
					// Start: input mint is not anyone's output
					if !outputMints[sd.Tx.InputMint.String()] {
						startSwap = sd
					}
					// End: output mint is not anyone's input
					if !inputMints[sd.Tx.OutputMint.String()] {
						endSwap = sd
					}
				}
				if startSwap == nil {
					startSwap = &txSwaps[0]
				}
				if endSwap == nil {
					endSwap = &txSwaps[len(txSwaps)-1]
				}
				swapInfo.TokenInMint = startSwap.Tx.InputMint
				swapInfo.TokenInAmount = startSwap.Tx.InputAmount
				swapInfo.TokenInDecimals = startSwap.Tx.InputMintDecimals
				swapInfo.TokenOutMint = endSwap.Tx.OutputMint
				swapInfo.TokenOutAmount = endSwap.Tx.OutputAmount
				swapInfo.TokenOutDecimals = endSwap.Tx.OutputMintDecimals
			}

			seenAMMs := make(map[string]bool)
			for _, sd := range otherSwaps {
				if !seenAMMs[string(sd.Type)] {
					swapInfo.AMMs = append(swapInfo.AMMs, string(sd.Type))
					seenAMMs[string(sd.Type)] = true
				}
			}
			swapInfo.Timestamp = time.Now()
			return swapInfo, tx, nil
		}

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

	for idx, inner := range innerInstructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]

		pr, ok := lookupParser(progID)
		if !ok || pr.Kind() != KindAMM {
			continue
		}
		if d, ok := pr.(routerDeduper); ok {
			key := d.dedupKey()
			if processedProtocols[key] {
				continue
			}
			processedProtocols[key] = true
		}
		if innerSwaps := pr.ParseInner(p, instructionIndex, idx, inner); len(innerSwaps) > 0 {
			swaps = append(swaps, innerSwaps...)
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
	// New path: simple ACCOUNT_INDEX programs registered via RegisterLayout resolve
	// through the shared resolvePoolInfo service. Falls through to the legacy switch
	// for everything not yet migrated.
	if l, ok := lookupLayout(progID); ok {
		return p.resolvePoolInfo(tx, instruction, l)
	}
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
		protocol = string(ORCA)
		if len(instruction.Data) >= discriminatorLen {
			disc := instruction.Data[:discriminatorLen]
			switch {
			case bytes.Equal(disc, orca_whirlpool.Instruction_SwapV2[:]):
				// swap_v2: token_program_a, token_program_b, memo_program, token_authority, whirlpool, token_mint_a, token_mint_b, token_owner_account_a, token_vault_a, token_owner_account_b, token_vault_b, tick_arrays..., oracle
				poolAccountIndex = 4
				poolInAccountIndex = 8
				poolOutAccountIndex = 10
			case bytes.Equal(disc, orca_whirlpool.Instruction_Swap[:]):
				// swap: token_program, token_authority, whirlpool, token_owner_a, token_vault_a, token_owner_b, token_vault_b, tick_arrays..., oracle
				poolAccountIndex = 2
				poolInAccountIndex = 4
				poolOutAccountIndex = 6
			default:
				discHex := hex.EncodeToString(disc)
				if _, ok := removeDiscriminator[discHex]; ok {
					tx.Type = TxTypeRemove
					poolAccountIndex = 0
					poolInAccountIndex = uint16(len(instruction.Accounts)) - 4
					poolOutAccountIndex = uint16(len(instruction.Accounts)) - 3
				} else if _, ok := addDiscriminator[discHex]; ok {
					tx.Type = TxTypeAdd
					poolAccountIndex = 0
					poolInAccountIndex = uint16(len(instruction.Accounts)) - 4
					poolOutAccountIndex = uint16(len(instruction.Accounts)) - 3
				}
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
			if _, ok := removeDiscriminator[hex.EncodeToString(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 2
				poolInAccountIndex = 6
				poolOutAccountIndex = 7
			} else if _, ok := addDiscriminator[hex.EncodeToString(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeAdd
				poolAccountIndex = 2
				poolInAccountIndex = 6
				poolOutAccountIndex = 7
				if hex.EncodeToString(instruction.Data[:discriminatorLen]) == calculateDiscriminator("global:initialize") {
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
			if _, ok := removeDiscriminator[hex.EncodeToString(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 3
				poolInAccountIndex = 5
				poolOutAccountIndex = 6
			} else if _, ok := addDiscriminator[hex.EncodeToString(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeAdd
				poolAccountIndex = 2
				poolInAccountIndex = 9
				poolOutAccountIndex = 10
			}
		}
	case progID.Equals(RAYDIUM_LAUNCHLAB_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 8
		poolOutAccountIndex = 7
		protocol = "Raydium Launchlab"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{1},   // buy
			{3},   // sell
			{250}, // swap (0xfa)
		}
	case progID.Equals(METEORA_PROGRAM_ID) ||
		progID.Equals(METEORA_DLMM_PROGRAM_ID):
		poolAccountIndex = 0
		poolInAccountIndex = 2
		poolOutAccountIndex = 3
		//fromAccountIndex = 0
		//toAccountIndex = 10
		protocol = "Meteora_DLMM_Program"
		if len(instruction.Data) >= discriminatorLen {
			liquidity := false
			if _, ok := removeDiscriminator[hex.EncodeToString(instruction.Data[:discriminatorLen])]; ok {
				tx.Type = TxTypeRemove
				poolAccountIndex = 1
				poolInAccountIndex = 5
				poolOutAccountIndex = 6
				liquidity = true
			} else if _, ok := addDiscriminator[hex.EncodeToString(instruction.Data[:discriminatorLen])]; ok {
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
	case progID.Equals(BYREAL_CLMM_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "Byreal CLMM"
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
	case progID.Equals(METEORA_POOLS_PROGRAM_ID):
		poolAccountIndex = 0
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "Meteora Pools Program"
	case progID.Equals(Meteora_Dynamic_Bonding_Curve_Program):
		poolAccountIndex = 2
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "Meteora Dynamic Bonding Curve"
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
	case progID.Equals(PUMP_FUN_PROGRAM_ID):
		// Pump.fun bonding curve — discriminator determines V1 vs V2 account layout.
		// V1 (buy / buy_exact_sol_in / sell): pool=3, poolIn=4
		// V2 (buy_exact_quote_in_v2 / sell_v2): pool=10, poolIn=11, poolOut=12
		if len(instruction.Data) < 8 {
			err = errors.New("PumpFun bonding curve: instruction data too short")
			return
		}
		disc := [8]byte(instruction.Data[:8])
		switch disc {
		case pumpfun.Instruction_Buy, pumpfun.Instruction_BuyExactSolIn, pumpfun.Instruction_Sell, PumpFunAMMSellExactInDiscriminator:
			// V1 layout: accounts[3]=bonding_curve, accounts[4]=associated_bonding_curve
			poolAccountIndex = 3
			poolInAccountIndex = 4
			poolOutAccountIndex = 4
		case pumpfun.Instruction_BuyExactQuoteInV2, pumpfun.Instruction_SellV2, pumpfun.Instruction_BuyV2:
			// V2 layout: accounts[10]=bonding_curve, accounts[11]=associated_base_bonding_curve, accounts[12]=associated_quote_bonding_curve
			poolAccountIndex = 10
			poolInAccountIndex = 11
			poolOutAccountIndex = 12
		default:
			err = fmt.Errorf("PumpFun bonding curve: unknown discriminator %x", disc)
			return
		}
		// Set discriminatorWhiteList to the matched discriminator so it passes the
		// global discriminator check below (bonding curve discriminators don't appear
		// in the global swapDiscriminator table).
		discriminatorWhiteList = [][]byte{disc[:]}
		protocol = string(PUMP_FUN)
	case progID.Equals(METEORA_DAMM_V2):
		poolAccountIndex = 1
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Meteora_DAMM_V2"
	case progID.Equals(ZEROFI):
		poolAccountIndex = 0
		poolInAccountIndex = 2
		poolOutAccountIndex = 4
		protocol = "ZeroFi"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{6},
			{16},
		}
	case progID.Equals(PANCAKE_SWAP_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 5
		poolOutAccountIndex = 6
		protocol = "PancakeSwap"
	case progID.Equals(MANIFEST_PROGRAM_ID):
		poolAccountIndex = 2
		poolInAccountIndex = 6
		poolOutAccountIndex = 7
		protocol = "Manifest"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{4},
			{13},
		}
		if int(poolAccountIndex) < len(instruction.Accounts) && p.allAccountKeys[instruction.Accounts[poolAccountIndex]].Equals(solana.SystemProgramID) {
			poolAccountIndex = 1
			poolOutAccountIndex = 5
		}
	case progID.Equals(HUMIDIDI_PROGRAM_ID):
		poolAccountIndex = 1
		poolInAccountIndex = 2
		poolOutAccountIndex = 3
		protocol = "HumidiFi"
		discriminatorWhiteList = [][]byte{
			{149, 59, 131, 119, 245, 228, 249, 17},
			{61, 203, 246, 110, 148, 191, 54, 98},
			{94, 146, 200, 36, 3, 236, 122, 214},
		}
	case progID.Equals(TESSERA_V_PROGRAM_ID):
		poolAccountIndex = 1
		poolInAccountIndex = 4
		poolOutAccountIndex = 5
		protocol = "Tessera V"
		discriminatorLen = 1
		discriminatorWhiteList = [][]byte{
			{16},
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

	tx.Pool = p.allAccountKeys[instruction.Accounts[poolAccountIndex]]
	tx.PoolIn = p.allAccountKeys[poolInAccountIndex]
	tx.PoolOut = p.allAccountKeys[poolOutAccountIndex]

	poolInBalance, okIn := p.postBalance[poolInAccountIndex]
	poolOutBalance, okOut := p.postBalance[poolOutAccountIndex]

	if okIn && okOut {
		if poolInBalance.Mint.Equals(tx.OutputMint) {
			poolInAccountIndex, poolOutAccountIndex = poolOutAccountIndex, poolInAccountIndex
			poolInBalance, poolOutBalance = poolOutBalance, poolInBalance
			tx.PoolIn = p.allAccountKeys[poolInAccountIndex]
			tx.PoolOut = p.allAccountKeys[poolOutAccountIndex]
		}
		if !poolInBalance.Mint.Equals(tx.InputMint) {
			err = errors.New("no inputMint for account")
			return
		}
		if !poolOutBalance.Mint.Equals(tx.OutputMint) {
			err = errors.New("no outputMint for account")
			return
		}

		if a, err := decimal.NewFromString(poolInBalance.UiTokenAmount.Amount); err == nil {
			tx.PoolInAmount = a.BigInt()
		}
		tx.InputMintDecimals = poolInBalance.UiTokenAmount.Decimals

		if a, err := decimal.NewFromString(poolOutBalance.UiTokenAmount.Amount); err == nil {
			tx.PoolOutAmount = a.BigInt()
		}
		tx.OutputMintDecimals = poolOutBalance.UiTokenAmount.Decimals
	}

	tx.Protocol = protocol
	return
}

func calculateDiscriminator(instructionName string) string {
	hash := sha256.Sum256([]byte(instructionName))
	return hex.EncodeToString(hash[:8])
}
