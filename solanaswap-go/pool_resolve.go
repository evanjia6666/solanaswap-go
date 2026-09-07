package solanaswapgo

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/shopspring/decimal"
)

// poolLayout declares, for one program, where the pool and its two vaults live in an
// instruction's account list, plus how to validate the instruction discriminator.
// This is the per-protocol domain knowledge that an IDL cannot supply; each protocol
// parser owns its own layout(s) instead of a case in the central setTxPoolInfo switch.
type poolLayout struct {
	poolIdx    uint16 // account index of the pool/market
	poolInIdx  uint16 // account index of the input-side vault
	poolOutIdx uint16 // account index of the output-side vault
	protocol   string // human-readable label written to TxInfo.Protocol

	// discriminatorLen is the number of leading data bytes that form the discriminator
	// (8 for Anchor, 1 for legacy programs). Defaults to 8 when zero.
	discriminatorLen int
	// whitelist, when non-empty, is the exact set of accepted discriminators. When empty
	// the global swapDiscriminator map is used (plus the add/remove maps as fallthrough).
	whitelist [][]byte
}

// resolvePoolInfo validates the instruction discriminator against the layout, resolves
// Pool/PoolIn/PoolOut from the instruction accounts, and — when post balances are
// available — disambiguates vault direction and fills PoolInAmount/PoolOutAmount and
// the authoritative mint decimals.
//
// This is the shared service extracted verbatim from the tail of the legacy
// setTxPoolInfo, so every migrated protocol keeps byte-identical behaviour.
func (p *ParseContext) resolvePoolInfo(tx *TxInfo, instruction solana.CompiledInstruction, l poolLayout) error {
	tx.Type = TxTypeSwap
	discriminatorLen := l.discriminatorLen
	if discriminatorLen == 0 {
		discriminatorLen = 8
	}

	if len(instruction.Data) < discriminatorLen {
		return errors.New("invalid instruction data length")
	}

	discriminator := hex.EncodeToString(instruction.Data[:discriminatorLen])
	var m map[string]bool
	if len(l.whitelist) > 0 {
		m = map[string]bool{}
		for _, d := range l.whitelist {
			m[hex.EncodeToString(d)] = true
		}
	} else {
		m = swapDiscriminator
	}
	if _, ok := m[discriminator]; ok {
	} else if _, ok := removeDiscriminator[discriminator]; ok {
		tx.Type = TxTypeRemove
	} else if _, ok := addDiscriminator[discriminator]; ok {
		tx.Type = TxTypeAdd
	} else {
		err := errors.New("discriminator unmatched")
		p.Log.Errorln(err, p.txInfo.Signatures, discriminator)
		return err
	}

	poolAccountIndex := l.poolIdx
	poolInAccountIndex := l.poolInIdx
	poolOutAccountIndex := l.poolOutIdx

	accLen := len(instruction.Accounts)
	if accLen < int(poolAccountIndex) || accLen <= int(poolOutAccountIndex) || accLen < int(poolInAccountIndex) {
		return fmt.Errorf("account index out of range %d/%d-%d-%d", accLen, poolAccountIndex, poolInAccountIndex, poolOutAccountIndex)
	}

	poolInAccountIndex = instruction.Accounts[poolInAccountIndex]
	poolOutAccountIndex = instruction.Accounts[poolOutAccountIndex]

	// a pool is never a token account: reject the leg when the layout guess
	// landed on a vault/user ATA (it would register a bogus pool upstream)
	if _, isTokenAccount := p.postBalance[instruction.Accounts[poolAccountIndex]]; isTokenAccount {
		return fmt.Errorf("pool account %s is a token account (layout mismatch)", p.allAccountKeys[instruction.Accounts[poolAccountIndex]])
	}

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
			return errors.New("no inputMint for account")
		}
		if !poolOutBalance.Mint.Equals(tx.OutputMint) {
			return errors.New("no outputMint for account")
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

	tx.Protocol = l.protocol
	return nil
}
