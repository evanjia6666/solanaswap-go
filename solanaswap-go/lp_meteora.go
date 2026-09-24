package solanaswapgo

import (
	"bytes"
	"math/big"
	"strconv"

	meteora_damm_v2 "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/meteora/meteora_damm_v2"
	meteora_dlmm "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/meteora/meteora_dlmm"
	meteora_pools "github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/meteora/meteora_pools"
	solana "github.com/gagliardetto/solana-go"
)

// Liquidity-instruction legs for the Meteora family (DAMM v2, Pools v1,
// DLMM). The swap scanners above only pair one-in/one-out transfers, so an
// add/remove liquidity instruction produced nothing — its reserves, fundmap
// and K-line contributions were silently lost. Each supported instruction is
// pinned to the account positions that identify the pool, its two token
// vaults and (where the instruction lists them) the mints, verified against
// the on-chain IDLs (cp_amm v0.2.0, amm v0.5.2, lb_clmm v0.12.0):
//
//   - cp_amm add_liquidity:      [pool, position, user_a, user_b, vault_a, vault_b, mint_a, mint_b, ...]
//   - cp_amm remove_*:           [pool_authority, pool, position, user_a, user_b, vault_a, vault_b, mint_a, mint_b, ...]
//   - amm (pools v1) LP variants: [pool, owner, position_nft, a_vault_lp, b_vault_lp, a_vault, b_vault,
//                                  a_vault_lp_mint, b_vault_lp_mint, a_token_vault, b_token_vault, ...]
//     (the SPL token vaults are at 9/10; mints are not instruction accounts
//     and come from the vaults' post balances)
//   - lb_clmm LP variants:        [position, lb_pair, (bitmap), user_x, user_y, reserve_x, reserve_y, mint_x, mint_y, ...]

type meteoraLPLayout struct {
	poolIdx, vaultAIdx, vaultBIdx uint16
	mintAIdx, mintBIdx            uint16
	mintsInAccounts               bool
	remove                        bool
}

var meteoraLPLayouts = []struct {
	program solana.PublicKey
	disc    [8]byte
	layout  meteoraLPLayout
}{
	{METEORA_DAMM_V2, meteora_damm_v2.Instruction_AddLiquidity,
		meteoraLPLayout{0, 4, 5, 6, 7, true, false}},
	{METEORA_DAMM_V2, meteora_damm_v2.Instruction_RemoveLiquidity,
		meteoraLPLayout{1, 5, 6, 7, 8, true, true}},
	{METEORA_DAMM_V2, meteora_damm_v2.Instruction_RemoveAllLiquidity,
		meteoraLPLayout{1, 5, 6, 7, 8, true, true}},

	{METEORA_POOLS_PROGRAM_ID, meteora_pools.Instruction_AddBalanceLiquidity,
		meteoraLPLayout{0, 9, 10, 0, 0, false, false}},
	{METEORA_POOLS_PROGRAM_ID, meteora_pools.Instruction_RemoveBalanceLiquidity,
		meteoraLPLayout{0, 9, 10, 0, 0, false, true}},
	{METEORA_POOLS_PROGRAM_ID, meteora_pools.Instruction_AddImbalanceLiquidity,
		meteoraLPLayout{0, 9, 10, 0, 0, false, false}},
	{METEORA_POOLS_PROGRAM_ID, meteora_pools.Instruction_RemoveLiquiditySingleSide,
		meteoraLPLayout{0, 9, 10, 0, 0, false, true}},
	{METEORA_POOLS_PROGRAM_ID, meteora_pools.Instruction_BootstrapLiquidity,
		meteoraLPLayout{0, 9, 10, 0, 0, false, false}},

	{METEORA_PROGRAM_ID, meteora_dlmm.Instruction_AddLiquidity,
		meteoraLPLayout{1, 5, 6, 7, 8, true, false}},
	{METEORA_PROGRAM_ID, meteora_dlmm.Instruction_AddLiquidity2,
		meteoraLPLayout{1, 5, 6, 7, 8, true, false}},
	{METEORA_PROGRAM_ID, meteora_dlmm.Instruction_RemoveLiquidity,
		meteoraLPLayout{1, 5, 6, 7, 8, true, true}},
	{METEORA_PROGRAM_ID, meteora_dlmm.Instruction_RemoveLiquidity2,
		meteoraLPLayout{1, 5, 6, 7, 8, true, true}},
	{METEORA_PROGRAM_ID, meteora_dlmm.Instruction_RemoveAllLiquidity,
		meteoraLPLayout{1, 5, 6, 7, 8, true, true}},
}

// matchMeteoraLPLayout resolves a liquidity-instruction discriminator on a
// Meteora family program to its account layout.
func matchMeteoraLPLayout(program solana.PublicKey, discriminator []byte) (meteoraLPLayout, bool) {
	if len(discriminator) < 8 {
		return meteoraLPLayout{}, false
	}
	for _, e := range meteoraLPLayouts {
		if e.program.Equals(program) && bytes.Equal(e.disc[:], discriminator[:8]) {
			return e.layout, true
		}
	}
	return meteoraLPLayout{}, false
}

// processMeteoraLP builds one add/remove leg for a Meteora family liquidity
// instruction. Pool/vaults/mints come from the pinned account table; the
// vaults' post balances become the reserve snapshot the dex writer consumes;
// the user<->vault transfers carry the per-side amounts (single-side
// operations legitimately move only one side).
func (p *Parser) processMeteoraLP(router, program solana.PublicKey, outerIndex int, instr solana.CompiledInstruction, l meteoraLPLayout, transfers []solana.CompiledInstruction) []SwapData {
	need := int(l.poolIdx)
	for _, i := range []int{int(l.vaultAIdx), int(l.vaultBIdx)} {
		if i > need {
			need = i
		}
	}
	if l.mintsInAccounts && int(l.mintBIdx) > need {
		need = int(l.mintBIdx)
	}
	if len(instr.Accounts) <= need {
		return nil
	}

	vaultAIdx, vaultBIdx := instr.Accounts[l.vaultAIdx], instr.Accounts[l.vaultBIdx]
	mintA, mintB := "", ""
	if l.mintsInAccounts {
		mintA = p.allAccountKeys[instr.Accounts[l.mintAIdx]].String()
		mintB = p.allAccountKeys[instr.Accounts[l.mintBIdx]].String()
	} else {
		// pools v1: mints are not instruction accounts; the vault post
		// balances carry them
		if b := p.postBalance[vaultAIdx]; b != nil {
			mintA = b.Mint.String()
		}
		if b := p.postBalance[vaultBIdx]; b != nil {
			mintB = b.Mint.String()
		}
	}
	if mintA == "" || mintB == "" || mintA == mintB {
		return nil
	}

	tx := &TxInfo{
		Type:    TxTypeAdd,
		Router:  router,
		Amm:     program,
		Owner:   *p.txInfo.Message.Signers().Last(),
		Protocol: string(METEORA),
		Index:   uint(outerIndex * 256),
		Pool:    p.allAccountKeys[instr.Accounts[l.poolIdx]],
		PoolIn:  p.allAccountKeys[vaultAIdx],
		PoolOut: p.allAccountKeys[vaultBIdx],
	}
	if l.remove {
		tx.Type = TxTypeRemove
	}
	if b := p.postBalance[vaultAIdx]; b != nil {
		if amount, err := strconv.ParseUint(b.UiTokenAmount.Amount, 10, 64); err == nil {
			tx.PoolInAmount = new(big.Int).SetUint64(amount)
			tx.InputMintDecimals = b.UiTokenAmount.Decimals
		}
	}
	if b := p.postBalance[vaultBIdx]; b != nil {
		if amount, err := strconv.ParseUint(b.UiTokenAmount.Amount, 10, 64); err == nil {
			tx.PoolOutAmount = new(big.Int).SetUint64(amount)
			tx.OutputMintDecimals = b.UiTokenAmount.Decimals
		}
	}

	for _, ti := range transfers {
		var mint, source, destination string
		var amount uint64
		switch {
		case p.isTransfer(ti):
			td := p.processTransfer(ti)
			if td == nil {
				continue
			}
			mint, source, destination, amount = td.Mint, td.Info.Source, td.Info.Destination, td.Info.Amount
		case p.isTransferCheck(ti):
			td := p.processTransferCheck(ti)
			if td == nil {
				continue
			}
			mint, source, destination = td.Info.Mint, td.Info.Source, td.Info.Destination
			if a, err := strconv.ParseFloat(td.Info.TokenAmount.Amount, 64); err == nil {
				amount = uint64(a)
			}
		default:
			continue
		}
		isIn := destination == p.allAccountKeys[vaultAIdx].String() || destination == p.allAccountKeys[vaultBIdx].String()
		isOut := source == p.allAccountKeys[vaultAIdx].String() || source == p.allAccountKeys[vaultBIdx].String()
		if !isIn && !isOut {
			continue
		}
		if mint == mintA && tx.InputAmount == 0 {
			tx.InputMint = solana.MustPublicKeyFromBase58(mintA)
			tx.InputAmount = amount
		} else if mint == mintB && tx.OutputAmount == 0 {
			tx.OutputMint = solana.MustPublicKeyFromBase58(mintB)
			tx.OutputAmount = amount
		}
	}
	// mints must be set even for the untouched side of a single-side op
	if tx.InputMint.IsZero() {
		tx.InputMint = solana.MustPublicKeyFromBase58(mintA)
	}
	if tx.OutputMint.IsZero() {
		tx.OutputMint = solana.MustPublicKeyFromBase58(mintB)
	}
	if tx.InputMintDecimals == 0 {
		tx.InputMintDecimals = p.splDecimalsMap[mintA]
	}
	if tx.OutputMintDecimals == 0 {
		tx.OutputMintDecimals = p.splDecimalsMap[mintB]
	}

	return []SwapData{{Type: METEORA, Tx: tx}}
}
