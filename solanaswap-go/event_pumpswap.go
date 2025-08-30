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

/*{28 items
timestamp:
"1756367596"
baseAmountOut:
"675970993982"
maxQuoteAmountIn:
"100000000"
userBaseTokenReserves:
"0"
userQuoteTokenReserves:
"19121288468"
poolBaseTokenReserves:
"473967232409269"
poolQuoteTokenReserves:
"41103736846"
quoteAmountIn:
"58705783"
lpFeeBasisPoints:
"20"
lpFee:
"117412"
protocolFeeBasisPoints:
"5"
protocolFee:
"29353"
quoteAmountInWithLpFee:
"58823195"
userQuoteAmountIn:
"58881901"
pool:
"6C6vbZXZdxAkgDKWRFxxzRJ2vMCkfqptyRLWz3C7svMq"
user:
"A59dYap98vTZ5icj3JMrqiTsPZHusrVmUsmNv8fG6U7T"
userBaseTokenAccount:
"B1QdVzDfTcDm3E3NjTPiyZkkqmYmTqxMbxt4o9r7a6hc"
userQuoteTokenAccount:
"4AD8g7vPyr9XRr6hCkP1KBysNk9h73hsepSfcudQPPJv"
protocolFeeRecipient:
"7hTckgnGnLQR6sdH7YkqFTAA7VwTfYFaZ6EhEsU3saCX"
protocolFeeRecipientTokenAccount:
"X5QPJcpph4mBAJDzc4hRziFftSbcygV59kRb2Fu6Je1"
coinCreator:
"8rcz5xq9YUt6ZqmkjhEDZAagvFykpxGUMcbmoXChxptq"
coinCreatorFeeBasisPoints:
"5"
coinCreatorFee:
"29353"
trackVolume:
true
totalUnclaimedTokens:
"0"
totalClaimedTokens:
"0"
currentSolVolume:
"0"
lastUpdateTimestamp:
"0"
}*/

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

/*
{23 items
timestamp:
"1745581741"
baseAmountIn:
"805492382843"
minQuoteAmountOut:
"38973953"
userBaseTokenReserves:
"805492382843"
userQuoteTokenReserves:
"0"
poolBaseTokenReserves:
"489992403515873"
poolQuoteTokenReserves:
"39678164444"
quoteAmountOut:
"65119389"
lpFeeBasisPoints:
"20"
lpFee:
"130239"
protocolFeeBasisPoints:
"5"
protocolFee:
"32560"
quoteAmountOutWithoutLpFee:
"64989150"
userQuoteAmountOut:
"64956590"
pool:
"6iWvjfWom8cBAj8pKeRh4ymQuL6M251FK48C9E6uRa6h"
user:
"CPguAH6jQXQYQf7gzjqADdhmqz9yDBbNf2QxwUmgGYtJ"
userBaseTokenAccount:
"BY9Jjd8b9cXmMk9iVN3npyEEBB3TQExgvnvwy3i4eV9H"
userQuoteTokenAccount:
"6HF6AoZe1MgH4jq7efPzpUaiE7auHDqARcJkgxU5DJNq"
protocolFeeRecipient:
"9rPYyANsfQZw3DnDmKE3YCQF5E8oD89UXoHn9JFEhJUz"
protocolFeeRecipientTokenAccount:
"Bvtgim23rfocUzxVX9j9QFxTbBnH8JZxnaGLCEkXvjKS"
coinCreator:
"11111111111111111111111111111111"
coinCreatorFeeBasisPoints:
"0"
coinCreatorFee:
"0"
}*/

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
