package solanaswapgo

import (
	"bytes"
	"math/big"

	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/pumpswap"
	"github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
)

// PumpSwap deposit/withdraw legs.
//
// The buy/sell parser only covers swaps, so LP transactions produced no leg
// and the pool's vault balances went stale between swaps (the dex keeps v2
// reserves from the vault post balances of parsed legs). The deposit and
// withdraw instructions carry everything needed for an LP leg in their
// accounts (layout from the generated IDL, 15 accounts):
//
//	[0] pool  [2] user  [3] base_mint  [4] quote_mint
//	[9] pool_base_token_account (base vault)  [10] pool_quote_token_account
//
// Vault post balances are read from the tx meta (same source the swap legs
// use); the user-side amounts are the vault balance deltas (pre vs post),
// which for a deposit/withdraw are exactly the liquidity added/removed.

// isPumpFunAMMLiquidityInstruction reports whether the instruction is a
// PumpSwap deposit or withdraw.
func (p *Parser) isPumpFunAMMLiquidityInstruction(instruction solana.CompiledInstruction) (isDeposit bool, ok bool) {
	if !p.allAccountKeys[instruction.ProgramIDIndex].Equals(PUMPFUN_AMM_PROGRAM_ID) || len(instruction.Data) < 8 {
		return false, false
	}
	d := instruction.Data[:8]
	if bytes.Equal(d, pumpswap.Instruction_Deposit[:]) {
		return true, true
	}
	if bytes.Equal(d, pumpswap.Instruction_Withdraw[:]) {
		return false, true
	}
	return false, false
}

// processPumpFunAMMLiquidity builds the LP leg for a PumpSwap deposit or
// withdraw instruction; nil when the layout does not check out.
func (p *Parser) processPumpFunAMMLiquidity(progID solana.PublicKey, instruction solana.CompiledInstruction, index uint) *TxInfo {
	isDeposit, ok := p.isPumpFunAMMLiquidityInstruction(instruction)
	if !ok || len(instruction.Accounts) < 11 {
		return nil
	}
	p.ensurePostBalances()

	baseVaultIdx := instruction.Accounts[9]
	quoteVaultIdx := instruction.Accounts[10]
	baseBal, okBase := p.postBalance[baseVaultIdx]
	quoteBal, okQuote := p.postBalance[quoteVaultIdx]
	// both vaults are always written by deposit/withdraw, so both must carry
	// post balances; a missing one means the account guess is wrong
	if !okBase || !okQuote {
		return nil
	}

	tx := &TxInfo{
		Type:     TxTypeRemove,
		Amm:      progID,
		Owner:    *p.txInfo.Message.Signers().Last(),
		Protocol: string(PUMPSWAP),
		Index:    index,
		Pool:     p.allAccountKeys[instruction.Accounts[0]],
	}
	if isDeposit {
		tx.Type = TxTypeAdd
	}

	// keep InputMint/OutputMint bound to the vaults at PoolIn/PoolOut: the
	// mint a vault holds decides the pairing, the nominal base/quote slots
	// only seed it
	inMint, outMint := p.allAccountKeys[instruction.Accounts[3]], p.allAccountKeys[instruction.Accounts[4]]
	tx.PoolIn, tx.PoolOut = p.allAccountKeys[baseVaultIdx], p.allAccountKeys[quoteVaultIdx]
	if !baseBal.Mint.Equals(inMint) {
		if baseBal.Mint.Equals(outMint) && quoteBal.Mint.Equals(inMint) {
			inMint, outMint = outMint, inMint
			tx.PoolIn, tx.PoolOut = tx.PoolOut, tx.PoolIn
			baseBal, quoteBal = quoteBal, baseBal
			baseVaultIdx, quoteVaultIdx = quoteVaultIdx, baseVaultIdx
		} else {
			return nil
		}
	}
	tx.InputMint, tx.OutputMint = inMint, outMint

	tx.InputMintDecimals = baseBal.UiTokenAmount.Decimals
	tx.OutputMintDecimals = quoteBal.UiTokenAmount.Decimals
	if a, err := decimal.NewFromString(baseBal.UiTokenAmount.Amount); err == nil {
		tx.PoolInAmount = a.BigInt()
	}
	if a, err := decimal.NewFromString(quoteBal.UiTokenAmount.Amount); err == nil {
		tx.PoolOutAmount = a.BigInt()
	}

	// user-side amounts: vault balance deltas (post - pre), absolute; a
	// missing pre balance (fresh vault) treats pre as zero
	tx.InputAmount = vaultBalanceDelta(p.txMeta.PreTokenBalances, baseVaultIdx, tx.PoolInAmount)
	tx.OutputAmount = vaultBalanceDelta(p.txMeta.PreTokenBalances, quoteVaultIdx, tx.PoolOutAmount)
	return tx
}

// vaultBalanceDelta returns |post - pre| for one vault token account.
func vaultBalanceDelta(pre []solrpc.TokenBalance, accountIndex uint16, post *big.Int) uint64 {
	if post == nil {
		return 0
	}
	delta := new(big.Int).Set(post)
	for i := range pre {
		b := &pre[i]
		if b.AccountIndex != accountIndex {
			continue
		}
		if preAmt, err := decimal.NewFromString(b.UiTokenAmount.Amount); err == nil {
			delta.Sub(delta, preAmt.BigInt())
		}
		break
	}
	return delta.Uint64()
}

// ensurePostBalances lazily builds the post-balance index (shared with the
// jupiter path, which may not have run for a direct deposit).
func (p *Parser) ensurePostBalances() {
	if p.postBalance != nil {
		return
	}
	p.postBalance = make(map[uint16]*solrpc.TokenBalance, len(p.txMeta.PostTokenBalances))
	for i := range p.txMeta.PostTokenBalances {
		p.postBalance[p.txMeta.PostTokenBalances[i].AccountIndex] = &p.txMeta.PostTokenBalances[i]
	}
}
