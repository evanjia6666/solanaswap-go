package solanaswapgo

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/pumpswap"
	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

var (

	// PumpFun AMM (pAMMBay…) swap discriminators, sourced from the pumpswap IDL.
	// The buy/sell values coincide with the bonding-curve global:buy / global:sell hashes
	// but are used in AMM context here. sell_exact_in is not in the IDL and stays hardcoded.
	PumpFunAMMSellDiscriminator            = pumpswap.Instruction_Sell
	PumpFunAMMBuyDiscriminator             = pumpswap.Instruction_Buy
	PumpFunAMMBuyExactQuoteInDiscriminator = pumpswap.Instruction_BuyExactQuoteIn
	PumpFunAMMSellExactInDiscriminator     = [8]byte{149, 39, 222, 155, 211, 124, 152, 26}
)

type PumpfunAMMBuyEvent struct {
	Timestamp                        uint64
	BaseAmountOut                    uint64
	MaxQuoteAmountIn                 uint64
	UserBaseTokenReserves            uint64
	UserQuoteTokenReserves           uint64
	PoolBaseTokenReserves            uint64
	PoolQuoteTokenReserves           uint64
	QuoteAmountIn                    uint64
	LpFeeBasisPoints                 uint64
	LpFee                            uint64
	ProtocolFeeBasisPoints           uint64
	ProtocolFee                      uint64
	QuoteAmountInWithLpFee           uint64
	UserQuoteAmountIn                uint64
	Pool                             solana.PublicKey
	User                             solana.PublicKey
	UserBaseTokenAccount             solana.PublicKey
	UserQuoteTokenAccount            solana.PublicKey
	ProtocolFeeRecipient             solana.PublicKey
	ProtocolFeeRecipientTokenAccount solana.PublicKey
	CoinCreator                      solana.PublicKey
	CoinCreatorFeeBasisPoints        uint64
	CoinCreatorFee                   uint64
	TrackVolume                      bool
	TotalUnclaimedTokens             uint64
	TotalClaimedTokens               uint64
	CurrentSolVolume                 uint64
	LastUpdateTimestamp              uint64
}

type PumpfunAMMSellEvent struct {
	Timestamp                        uint64
	BaseAmountIn                     uint64
	MinQuoteAmountOut                uint64
	UserBaseTokenReserves            uint64
	UserQuoteTokenReserves           uint64
	PoolBaseTokenReserves            uint64
	PoolQuoteTokenReserves           uint64
	QuoteAmountOut                   uint64
	LpFeeBasisPoints                 uint64
	LpFee                            uint64
	ProtocolFeeBasisPoints           uint64
	ProtocolFee                      uint64
	QuoteAmountOutWithoutLpFee       uint64
	UserQuoteAmountOut               uint64
	Pool                             solana.PublicKey
	User                             solana.PublicKey
	UserBaseTokenAccount             solana.PublicKey
	UserQuoteTokenAccount            solana.PublicKey
	ProtocolFeeRecipient             solana.PublicKey
	ProtocolFeeRecipientTokenAccount solana.PublicKey
}

// Is
func (p *Parser) parsePumpfunAMMSwapEvent(tx *TxInfo, instruction solana.CompiledInstruction) error {
	decodedBytes, err := base58.Decode(instruction.Data.String())
	if err != nil {
		return fmt.Errorf("error decoding instruction data: %s", err)
	}
	decoder := ag_binary.NewBorshDecoder(decodedBytes[16:])

	if bytes.Equal(decodedBytes[:8], AnchorSelfCPIDiscriminator[:]) && bytes.Equal(decodedBytes[8:16], pumpswap.Event_BuyEvent[:]) {
		buyEvent, err := handlePumpFunAMMBuyEvent(decoder)
		if err != nil {
			return fmt.Errorf("error decoding pumpfun amm buy event: %s", err)
		}

		tx.InputAmount = buyEvent.QuoteAmountInWithLpFee
		tx.OutputAmount = buyEvent.BaseAmountOut
		tx.PoolInAmount = new(big.Int).SetUint64(buyEvent.PoolQuoteTokenReserves)
		tx.PoolOutAmount = new(big.Int).SetUint64(buyEvent.PoolBaseTokenReserves)

		return nil
	}
	if bytes.Equal(decodedBytes[:8], AnchorSelfCPIDiscriminator[:]) && bytes.Equal(decodedBytes[8:16], pumpswap.Event_SellEvent[:]) {
		sellEvent, err := handlePumpFunAMMSellEvent(decoder)
		if err != nil {
			return fmt.Errorf("error decoding pumpfun amm sell event: %s", err)
		}

		tx.InputAmount = sellEvent.BaseAmountIn
		tx.OutputAmount = sellEvent.UserQuoteAmountOut
		tx.PoolInAmount = new(big.Int).SetUint64(sellEvent.PoolBaseTokenReserves)
		tx.PoolOutAmount = new(big.Int).SetUint64(sellEvent.PoolQuoteTokenReserves)

		return nil
	}

	return fmt.Errorf("unhandled pumpfun amm swap event type")
}

func handlePumpFunAMMBuyEvent(decoder *ag_binary.Decoder) (*PumpfunAMMBuyEvent, error) {
	var event PumpfunAMMBuyEvent
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

func handlePumpFunAMMSellEvent(decoder *ag_binary.Decoder) (*PumpfunAMMSellEvent, error) {
	var event PumpfunAMMSellEvent
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (p *Parser) isPumpFunAMMBuyDiscriminator(instr solana.CompiledInstruction) bool {
	if !p.allAccountKeys[instr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) || len(instr.Data) < 8 {
		return false
	}
	decodedBytes, err := base58.Decode(instr.Data.String())
	if err != nil {
		return false
	}
	return bytes.Equal(decodedBytes[:8], pumpswap.Instruction_Buy[:])
}

// pumpfunDecimals returns the decimals for a PumpFun token.
// PumpFun ecosystem tokens always use 6 decimals. When the mint is not found
// in splDecimalsMap and is not native SOL, fall back to 6.
func (p *Parser) pumpfunDecimals(mint solana.PublicKey) uint8 {
	if d := p.splDecimalsMap[mint.String()]; d > 0 {
		return d
	}
	if mint.Equals(NATIVE_SOL_MINT_PROGRAM_ID) {
		return 9
	}
	return 6 // PumpFun default
}

func (p *Parser) processPumFumAMMBuySwaps(router solana.PublicKey, instruction solana.CompiledInstruction) *TxInfo {
	inputMint := p.allAccountKeys[instruction.Accounts[4]]
	outputMint := p.allAccountKeys[instruction.Accounts[3]]
	tx := &TxInfo{
		Type:               TxTypeSwap,
		Amm:                p.allAccountKeys[instruction.ProgramIDIndex],
		Router:             router,
		Owner:              *p.txInfo.Message.Signers().Last(),
		InputMint:          inputMint,
		InputMintDecimals:  p.pumpfunDecimals(inputMint),
		OutputMint:         outputMint,
		OutputMintDecimals: p.pumpfunDecimals(outputMint),
		Pool:               p.allAccountKeys[instruction.Accounts[0]],
		PoolIn:             p.allAccountKeys[instruction.Accounts[8]],
		PoolOut:            p.allAccountKeys[instruction.Accounts[7]],
		Protocol:           string(PUMPSWAP),
	}
	return tx
}

func (p *Parser) processPumpFunAMMSellSwaps(router solana.PublicKey, instruction solana.CompiledInstruction) *TxInfo {
	inputMint := p.allAccountKeys[instruction.Accounts[3]]
	outputMint := p.allAccountKeys[instruction.Accounts[4]]
	tx := &TxInfo{
		Type:               TxTypeSwap,
		Amm:                p.allAccountKeys[instruction.ProgramIDIndex],
		Router:             router,
		Owner:              *p.txInfo.Message.Signers().Last(),
		InputMint:          inputMint,
		InputMintDecimals:  p.pumpfunDecimals(inputMint),
		OutputMint:         outputMint,
		OutputMintDecimals: p.pumpfunDecimals(outputMint),
		Pool:               p.allAccountKeys[instruction.Accounts[0]],
		PoolIn:             p.allAccountKeys[instruction.Accounts[7]],
		PoolOut:            p.allAccountKeys[instruction.Accounts[8]],
		Protocol:           string(PUMPSWAP),
	}
	return tx
}

func (p *Parser) isPumpFunAMMBuyExactQuoteInDiscriminator(instr solana.CompiledInstruction) bool {
	if !p.allAccountKeys[instr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) || len(instr.Data) < 8 {
		return false
	}
	decodedBytes, err := base58.Decode(instr.Data.String())
	if err != nil {
		return false
	}
	return bytes.Equal(decodedBytes[:8], pumpswap.Instruction_BuyExactQuoteIn[:])
}

func (p *Parser) isPumpFunAMMSellExactInDiscriminator(instr solana.CompiledInstruction) bool {
	if !p.allAccountKeys[instr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) || len(instr.Data) < 8 {
		return false
	}
	decodedBytes, err := base58.Decode(instr.Data.String())
	if err != nil {
		return false
	}
	return bytes.Equal(decodedBytes[:8], PumpFunAMMSellExactInDiscriminator[:])
}

func (p *Parser) isPumpFunAMMSellDiscriminator(instr solana.CompiledInstruction) bool {
	if !p.allAccountKeys[instr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) || len(instr.Data) < 8 {
		return false
	}
	decodedBytes, err := base58.Decode(instr.Data.String())
	if err != nil {
		return false
	}
	return bytes.Equal(decodedBytes[:8], pumpswap.Instruction_Sell[:])
}
