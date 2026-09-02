package solanaswapgo

import (
	"encoding/binary"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
)

const (
	solMint  = "So11111111111111111111111111111111111111112"
	usdcMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	usdtMint = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
)

// Real SwapsEvent CPI from mainnet tx
// 4ohi3z3AZFckSZn6d2HfhEqXhx9N9DRTMAz9oJ4YhiDbc86dptbtDkZpxhrpGX1AYUKrVoGmvHC65qEE3E9iMWA1
// (USDC->USDT Jupiter split route: Manifest USDC/USDT + Tessera V SOL/USDC + Orca SOL/USDT).
// On-chain balances: 7.081025 USDC -> 7.082902 USDT (Manifest),
// 0.253785 USDC -> 0.00247581 SOL (Tessera), 0.00247581 SOL -> 0.253798 USDT (Orca).
// Entries carry a trailing 32-byte Amm (112 bytes each) since Jupiter's 2026-08 upgrade.
const swapsevent112Payload = "2oyEjxZKejR7UaG3VuhoNicTTVtRUmwvdb6hEQMk8CSw6UE9uEdewApVTATGjjR5if3mZJLnxLnvjCETJXHV7i6L1UjpxxiDiTRNkRnfvFdffLsXYDWsuh3nBdzLkfpr9RTJsiogzyR7rEEpyPwuDCKXsyxAvuxqgSR4ucfcgVjMo6kcnproJqfiUM5uv2pfxmuuBdGA5yhF4pH1LfVegGEJUfKVYo7jnrUfCE78ZPaEfDQwDsNfvxtFqqykeJMRFXbGLh6XoGggGWL8BVoKbVYQETYQoQK7ssQqqYsLsDLghmLMBjmk1BL43cEFFqd4AKBg7XHLo58wu518saKH7ffpyPcKcDb6tPmFzmWw4iDjkfXD76oadikuvD845JzYrPDF4FSoj3N96iKjSXXo7meTiDLtniubZuPJQrJKMVEw7Me9T5QyxBbkdi77eLEXs1o5BSLKvNCt5WeiC1hn1dkwYEwwsbasc2C6FNQ"

func TestParseSwapsEvent_112ByteEntries(t *testing.T) {
	p := &Parser{splDecimalsMap: map[string]uint8{
		solMint: 9, usdcMint: 6, usdtMint: 6,
	}}

	// CompiledInstruction.Data holds raw bytes and is re-encoded to base58 by
	// .String(), so decode the fixture text into raw event bytes first.
	raw, err := base58.Decode(swapsevent112Payload)
	require.NoError(t, err)

	events, err := p.parseSwapsEventInstruction(solana.CompiledInstruction{
		Data: solana.Base58(raw),
	})
	require.NoError(t, err)
	require.Len(t, events, 3)

	require.Equal(t, usdcMint, events[0].InputMint.String())
	require.Equal(t, uint64(7081025), events[0].InputAmount)
	require.Equal(t, usdtMint, events[0].OutputMint.String())
	require.Equal(t, uint64(7082902), events[0].OutputAmount)
	require.Equal(t, "MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms", events[0].Amm.String())

	require.Equal(t, usdcMint, events[1].InputMint.String())
	require.Equal(t, uint64(253785), events[1].InputAmount)
	require.Equal(t, solMint, events[1].OutputMint.String())
	require.Equal(t, uint64(2475810), events[1].OutputAmount)
	require.Equal(t, "TessVdML9pBGgG9yGks7o4HewRaXVAMuoVj4x83GLQH", events[1].Amm.String())

	require.Equal(t, solMint, events[2].InputMint.String())
	require.Equal(t, uint64(2475810), events[2].InputAmount)
	require.Equal(t, usdtMint, events[2].OutputMint.String())
	require.Equal(t, uint64(253798), events[2].OutputAmount)
	require.Equal(t, "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc", events[2].Amm.String())

	// The pre-fix 80-byte stride misread mint pubkey bytes as amounts; these are the
	// exact garbage values that hit production (SOL 24h volume inflated ~1e9x).
	for _, magic := range []uint64{4228277238345956038, 9548101793880775430} {
		for _, e := range events {
			require.NotEqual(t, magic, e.InputAmount)
			require.NotEqual(t, magic, e.OutputAmount)
		}
	}
}

func TestParseSwapsEvent_Legacy80ByteEntries(t *testing.T) {
	mintA := solana.MustPublicKeyFromBase58(solMint)
	mintB := solana.MustPublicKeyFromBase58(usdcMint)

	buf := make([]byte, 0, 16+4+2*80)
	buf = append(buf, AnchorSelfCPIDiscriminator[:]...)
	buf = append(buf, SwapsEventDiscriminator[:]...)
	buf = binary.LittleEndian.AppendUint32(buf, 2)
	for i, amt := range []uint64{1000000, 2000000} {
		in, out := mintA, mintB
		if i%2 == 1 {
			in, out = out, in
		}
		buf = append(buf, in.Bytes()...)
		buf = binary.LittleEndian.AppendUint64(buf, amt)
		buf = append(buf, out.Bytes()...)
		buf = binary.LittleEndian.AppendUint64(buf, amt*2)
	}

	p := &Parser{}
	events, err := p.parseSwapsEventInstruction(solana.CompiledInstruction{
		Data: solana.Base58(buf),
	})
	require.NoError(t, err)
	require.Len(t, events, 2)

	require.Equal(t, solMint, events[0].InputMint.String())
	require.Equal(t, uint64(1000000), events[0].InputAmount)
	require.Equal(t, usdcMint, events[0].OutputMint.String())
	require.Equal(t, uint64(2000000), events[0].OutputAmount)
	require.True(t, events[0].Amm.IsZero())

	require.Equal(t, usdcMint, events[1].InputMint.String())
	require.Equal(t, uint64(2000000), events[1].InputAmount)
	require.Equal(t, solMint, events[1].OutputMint.String())
	require.Equal(t, uint64(4000000), events[1].OutputAmount)
}

func TestParseSwapsEvent_RejectsUnknownStride(t *testing.T) {
	buf := make([]byte, 0, 16+4+200)
	buf = append(buf, AnchorSelfCPIDiscriminator[:]...)
	buf = append(buf, SwapsEventDiscriminator[:]...)
	buf = binary.LittleEndian.AppendUint32(buf, 2)
	buf = append(buf, make([]byte, 200)...)

	p := &Parser{}
	_, err := p.parseSwapsEventInstruction(solana.CompiledInstruction{
		Data: solana.Base58(buf),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported SwapsEvent entry size")
}

// The Tessera V out vault is account 3; account 5 is a fee/treasury account, not a
// token account. Pointing poolOutIdx at it left every Tessera pool with zero reserves.
func TestTesseraVLayout(t *testing.T) {
	l, ok := lookupLayout(solana.MustPublicKeyFromBase58("TessVdML9pBGgG9yGks7o4HewRaXVAMuoVj4x83GLQH"))
	require.True(t, ok)
	require.Equal(t, uint16(1), l.poolIdx)
	require.Equal(t, uint16(4), l.poolInIdx)
	require.Equal(t, uint16(3), l.poolOutIdx)
}
