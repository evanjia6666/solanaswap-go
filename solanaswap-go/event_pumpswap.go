package solanaswapgo

import (
	"bytes"
	"fmt"
	"math/big"

	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

var (
	PumpFunAMMSellEventDiscriminator = [16]byte{228, 69, 165, 46, 81, 203, 154, 29, 62, 47, 55, 10, 165, 3, 220, 42}
	PumpFunAMMBuyEventDiscriminator  = [16]byte{228, 69, 165, 46, 81, 203, 154, 29, 103, 244, 82, 31, 44, 245, 119, 119}

	PumpFunAMMSellDiscriminator = [8]byte{51, 230, 133, 164, 1, 127, 131, 173}
	PumpFunAMMBuyDiscriminator  = [8]byte{102, 6, 61, 18, 1, 218, 235, 234}
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

	if bytes.Equal(decodedBytes[:16], PumpFunAMMBuyEventDiscriminator[:]) {
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
	if bytes.Equal(decodedBytes[:16], PumpFunAMMSellEventDiscriminator[:]) {
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
	return bytes.Equal(decodedBytes[:8], PumpFunAMMBuyDiscriminator[:])
}

func (p *Parser) processPumFumAMMBuySwaps(router solana.PublicKey, instruction solana.CompiledInstruction) *TxInfo {
	tx := &TxInfo{
		Type:               TxTypeSwap,
		Amm:                p.allAccountKeys[instruction.ProgramIDIndex],
		Router:             router,
		Owner:              *p.txInfo.Message.Signers().Last(),
		InputMint:          p.allAccountKeys[instruction.Accounts[4]],
		InputMintDecimals:  p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[4]].String()].Decimals,
		OutputMint:         p.allAccountKeys[instruction.Accounts[3]],
		OutputMintDecimals: p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[3]].String()].Decimals,
		Pool:               p.allAccountKeys[instruction.Accounts[0]],
		PoolIn:             p.allAccountKeys[instruction.Accounts[8]],
		PoolOut:            p.allAccountKeys[instruction.Accounts[7]],
		Protocol:           string(PUMP_FUN),
	}
	return tx
}

func (p *Parser) processPumpFunAMMSellSwaps(router solana.PublicKey, instruction solana.CompiledInstruction) *TxInfo {
	tx := &TxInfo{
		Type:               TxTypeSwap,
		Amm:                p.allAccountKeys[instruction.ProgramIDIndex],
		Router:             router,
		Owner:              *p.txInfo.Message.Signers().Last(),
		InputMint:          p.allAccountKeys[instruction.Accounts[3]],
		InputMintDecimals:  p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[3]].String()].Decimals,
		OutputMint:         p.allAccountKeys[instruction.Accounts[4]],
		OutputMintDecimals: p.splTokenInfoMap[p.allAccountKeys[instruction.Accounts[4]].String()].Decimals,
		Pool:               p.allAccountKeys[instruction.Accounts[0]],
		PoolIn:             p.allAccountKeys[instruction.Accounts[7]],
		PoolOut:            p.allAccountKeys[instruction.Accounts[8]],
		Protocol:           string(PUMP_FUN),
	}
	return tx
}

func (p *Parser) isPumpFunAMMSellDiscriminator(instr solana.CompiledInstruction) bool {
	if !p.allAccountKeys[instr.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) || len(instr.Data) < 8 {
		return false
	}
	decodedBytes, err := base58.Decode(instr.Data.String())
	if err != nil {
		return false
	}
	return bytes.Equal(decodedBytes[:8], PumpFunAMMSellDiscriminator[:])
}
