package solanaswapgo

// Synthetic round-trip for the PumpSwap deposit/withdraw LP legs: the
// instruction is built with the generated IDL builder (authoritative
// layout), wrapped in a transaction with fabricated pre/post vault balances,
// and asserted against the parsed leg. A live-deposit golden case should be
// added once a mainnet signature is captured (LP traffic is ~0.1% of the
// program's).

import (
	"math/big"
	"testing"

	"github.com/franco-bianco/solanaswap-go/solanaswap-go/defi/pumpswap"
	"github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func TestPumpFunAMMLiquidityLegs(t *testing.T) {
	user := fixedKey(1)
	pool := fixedKey(2)
	globalConfig := fixedKey(3)
	baseMint := fixedKey(4)
	quoteMint := fixedKey(5)
	lpMint := fixedKey(6)
	userBase := fixedKey(7)
	userQuote := fixedKey(8)
	userPoolATA := fixedKey(9)
	vaultBase := fixedKey(10)
	vaultQuote := fixedKey(11)

	build := func(disc [8]byte) *solana.Transaction {
		instr := solana.NewInstruction(
			PUMPFUN_AMM_PROGRAM_ID,
			solana.AccountMetaSlice{
				{PublicKey: pool, IsWritable: true},
				{PublicKey: globalConfig},
				{PublicKey: user, IsSigner: true},
				{PublicKey: baseMint},
				{PublicKey: quoteMint},
				{PublicKey: lpMint, IsWritable: true},
				{PublicKey: userBase, IsWritable: true},
				{PublicKey: userQuote, IsWritable: true},
				{PublicKey: userPoolATA, IsWritable: true},
				{PublicKey: vaultBase, IsWritable: true},
				{PublicKey: vaultQuote, IsWritable: true},
				{PublicKey: solana.TokenProgramID},
				{PublicKey: solana.Token2022ProgramID},
				{PublicKey: fixedKey(99)},
				{PublicKey: PUMPFUN_AMM_PROGRAM_ID},
			},
			append(disc[:], make([]byte, 24)...), // three u64 params
		)
		tx, err := solana.NewTransaction([]solana.Instruction{instr}, solana.Hash{}, solana.TransactionPayer(user))
		require.NoError(t, err)
		return tx
	}

	// account indexes as the built message orders them: the builder prepends
	// the payer, then the metas in order
	idx := func(tx *solana.Transaction, key solana.PublicKey) uint16 {
		for i, k := range tx.Message.AccountKeys {
			if k.Equals(key) {
				return uint16(i)
			}
		}
		t.Fatalf("key %s missing from message", key)
		return 0
	}

	poolPtr := pool
	bal := func(amount string, mint solana.PublicKey, acctIdx uint16) solrpc.TokenBalance {
		return solrpc.TokenBalance{
			AccountIndex: acctIdx,
			Mint:         mint,
			Owner:        &poolPtr,
			UiTokenAmount: &solrpc.UiTokenAmount{
				Amount:   amount,
				Decimals: 6,
			},
		}
	}

	for _, tc := range []struct {
		name     string
		disc     [8]byte
		wantType string
	}{
		{"deposit", pumpswap.Instruction_Deposit, TxTypeAdd},
		{"withdraw", pumpswap.Instruction_Withdraw, TxTypeRemove},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := build(tc.disc)
			vb, vq := idx(tx, vaultBase), idx(tx, vaultQuote)
			meta := &solrpc.TransactionMeta{
				PreTokenBalances: []solrpc.TokenBalance{
					bal("1000000", baseMint, vb),
					bal("5000000", quoteMint, vq),
				},
				PostTokenBalances: []solrpc.TokenBalance{
					bal("1600000", baseMint, vb),  // +600000 base
					bal("7400000", quoteMint, vq), // +2400000 quote
				},
				LogMessages: []string{"Program log: instruction ok"},
			}
			parser, err := NewTransactionParser(tx, meta)
			require.NoError(t, err)

			legs, err := parser.ParseTransaction()
			require.NoError(t, err)
			require.Len(t, legs, 1)
			leg := legs[0].Tx
			require.NotNil(t, leg)
			require.Equal(t, tc.wantType, leg.Type)
			require.True(t, leg.Pool.Equals(pool))
			require.True(t, leg.Amm.Equals(PUMPFUN_AMM_PROGRAM_ID))
			require.True(t, leg.InputMint.Equals(baseMint))
			require.True(t, leg.OutputMint.Equals(quoteMint))
			require.True(t, leg.PoolIn.Equals(vaultBase))
			require.True(t, leg.PoolOut.Equals(vaultQuote))
			require.NotNil(t, leg.PoolInAmount)
			require.Equal(t, 0, big.NewInt(1600000).Cmp(leg.PoolInAmount))
			require.Equal(t, 0, big.NewInt(7400000).Cmp(leg.PoolOutAmount))
			require.Equal(t, uint64(600000), leg.InputAmount)
			require.Equal(t, uint64(2400000), leg.OutputAmount)
			require.Equal(t, uint8(6), leg.InputMintDecimals)
		})
	}
}

// fixedKey derives a stable distinct pubkey for fixtures (byte pattern id,
// then sequential bytes, zero padded) — hand-written base58 always ends in
// invalid-alphabet panics.
func fixedKey(id byte) solana.PublicKey {
	var b [32]byte
	b[0] = id
	for i := 1; i < 32; i++ {
		b[i] = byte(int(id)*7 + i)
	}
	return solana.PublicKeyFromBytes(b[:])
}
