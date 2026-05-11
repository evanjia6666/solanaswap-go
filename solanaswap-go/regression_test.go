package solanaswapgo

import (
	"context"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

type legExpectation struct {
	protocol   string
	inputMint  string
	outputMint string
	inputAmt   uint64
	outputAmt  uint64
	// pool       string
	// poolIn     string
	// poolOut    string
	// amm        string
}

func TestRegression_AllCases(t *testing.T) {
	rpcURL := os.Getenv("SOLANA_RPC_URL")
	if rpcURL == "" {
		rpcURL = rpc.MainNetBeta_RPC
	}
	client := rpc.New(rpcURL)

	tests := []struct {
		name     string
		sig      string
		expected []legExpectation
	}{
		{
			name: "Bitget Swap — Multi-leg",
			sig:  "5JtAbkDqdDqKRd5dfEpYFBAFiBP6zTDwtx6kEfUJxiyK197Vgb5yYnTw7DYxjzSdbnqTr6CknpgErLADEa2SrkQh",
			expected: []legExpectation{
				{"PumpFun.AMM", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 315168076744, 274135327},
				{"ZeroFi", "So11111111111111111111111111111111111111112", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 274135327, 23823600},
			},
		},
		{
			name: "Arbitrage Bot (3s1r)",
			sig:  "44JDfDCPZub9aPtgEe7z89ot7iaLvgbMC52YagQWVDsVubmXsayFo6Jpk6aG1v3E9sg9hBKcX1CUdFjotwzKuTZz",
			expected: []legExpectation{
				{"PumpFun.AMM", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 2724543, 3159124095},
				{"Meteora_DAMM_V2", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 3159124095, 2755904},
			},
		},
		{
			name: "Arbitrage Bot (B7qnn)",
			sig:  "5kCsHh9W6CPjxEpv9JGyvkJjSGHmPPxPtJCTk8wJB9xX81DUa7uBkRFQYUj9HeXTxsV1zTbjGCSDhQGmPBEzczZR",
			expected: []legExpectation{
				{"PumpFun.AMM", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 15451535, 19312159098},
				{"Meteora_DAMM_V2", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 19312159098, 16036463},
			},
		},
		{
			name: "Binance Wallet — Single Leg",
			sig:  "2rwTmdPNtUNysZtUafAW9FBW7Vn6y2LyhQr9ZRATgPihFxFEgCUnLvRkQTGyG6g6h9cCKdLSag96DPuqYdm4j1df",
			expected: []legExpectation{
				{"PumpFun.AMM", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 2270000000000, 1954407176},
			},
		},
		{
			name: "Axiom Trade — Single Leg",
			sig:  "5yQZpKcu3gmSKKwX3zMC4EWKDB9z8UFYZ4fweModAv2cSY1b38GUzmG1FQt7a39WWqdRX3S5fcnCZ2od3rZqGyGe",
			expected: []legExpectation{
				{"PumpFun.AMM", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 490108694, 607963442552},
			},
		},
		{
			name: "OKX Labs 2 — Single Leg",
			sig:  "2QzkwCkLd3mP2TSPQ5M7eL3qGMwZmyWzPShdUC7TrTzL5TmNRdsvKMH5GNG7L8VMfFAoGBRGt1ohSCpJqsF9rhTM",
			expected: []legExpectation{
				{"PumpFun.AMM", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 198300000, 245048222036},
			},
		},
		{
			name: "Bitget DEX Aggregator",
			sig:  "52ctk8ybpLmqfJvjPBj59d7tZsZxasREVCBEkdxbueu6ZuhD6WvrJjD4ouMEV5TTeAMSeK5yB43VUnQ3pynkHQ9k",
			expected: []legExpectation{
				{"PumpFun.AMM", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 1970335967, 2032255254700},
			},
		},
		{
			name: "Jupiter Aggregator v6 (Event)",
			sig:  "5LFEcHCGdXRn9FdmZvm1neT1jLm5T3dJToY1utcpsyHMifc7UNNJdeS6bQhXXtAwmQveR58ELyZfmi83wJinXYQH",
			expected: []legExpectation{
				{"PumpFun.AMM", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 114521290827, 100409303},
			},
		},
		{
			name: "Jupiter Aggregator v6 (RouteV2)",
			sig:  "4iaTjbw7nJ3aqeavwCyrMZvF69u8mU9zbJQMQaxifjyEayvVDUwcyHHQkPNbWhU4EAqkXSCQERR3y6DZ41jTxF1T",
			expected: []legExpectation{
				{"PumpFun.AMM", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 32143500000, 25657769},
				{"Raydium", "So11111111111111111111111111111111111111112", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 25657769, 2241421},
			},
		},
		{
			name: "HumidiFi — Multi-leg",
			sig:  "3aqonRWReqZUoZiRJuq9KX2uMUUVTKL64dTg2krM8enEwv6zw3tjm8c27HasVde3CdNofxwgpCHupSUHcSuGsubA",
			expected: []legExpectation{
				{"HumidiFi", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "So11111111111111111111111111111111111111112", 100986821, 1154784782},
				{"PumpFun.AMM", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 1143373865, 1402190646613},
			},
		},
		{
			name: "Raydium+Whirlpools Program (regression)",
			sig:  "HwPpFnBuyxCLRsuJNEZ5SBHx6xgtXy9TeLUkk8KNVjNXmbZsyyYyfkvdsEeD3mgNi4TYBs3A1wrbFHDDafqdwHm",
			expected: []legExpectation{
				{"Raydium Concentrated Liquidity", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", 449550000, 208773150},
				{"Whirlpools Program", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", 49950000, 23197311},
			},
		},
		{
			name: "Raydium swap (regression)",
			sig:  "3XgeS99txr7YDwyw14aVT1tewhQgEgBMzzxT6ZGAVPzusNrgx392wstsbgPrBxnKw6xJLtUfVrQpGvFFU4cQQfj5",
			expected: []legExpectation{
				{"Raydium Concentrated Liquidity", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 15367000, 33015387},
				{"Whirlpools Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 124333000, 267254823},
			},
		},
		{
			name: "Meteora DLMM (regression)",
			sig:  "qUMyimWMctuUAcTGFzJ8WZ7ncq3UJoAZTXaNh19He2NLyTnG47Y3rzFup42jnh8HtjbRyWxisgwgfS7etgzUmoA",
			expected: []legExpectation{
				{"Byreal CLMM", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 4513500, 9691643},
				{"Meteora_DLMM_Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 501501, 1077056},
				{"Manifest", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Xs8S1uUs1zvS2p7iwtsG3b6fkhpvmwz4GYU3gWAmWHZ", 10768699, 1506086},
			},
		},
		{
			name: "OKX Router — Multi-leg (Xsc9 anomaly)",
			sig:  "34jX27qt8bMeufVQAhEHtofcMiZSMasp5DcZHvJj6o5QNxfvmcFAAckUMbrfs26NPqqYxe2M6dEQM28eaCoEe787",
			expected: []legExpectation{
				{"Meteora_DLMM_Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "So11111111111111111111111111111111111111112", 510946, 11581194},
				{"Meteora_DLMM_Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 188981, 407737},
				{"Pancake Swap", "So11111111111111111111111111111111111111112", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 11581194, 1102408},
				{"Stable Swap", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", 1510145, 5820395},
			},
		},
		{
			name: "Jupiter RouteV2 multi-leg (regression)",
			sig:  "4jRhd8zs2pjhTCEuJ2argxZiw6wiQdHb4iK2u3CwgLmdFEp3tn21nASBKedV3544qEuqfqLcD6nydLNgt4kPQZwm",
			expected: []legExpectation{
				{"ZeroFi", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "So11111111111111111111111111111111111111112", 9659601, 103407029},
				{"Raydium", "So11111111111111111111111111111111111111112", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", 103407029, 4489017},
				{"Manifest", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 4489017, 9660288},
			},
		},
		{
			name: "Jupiter SharedAccountsRouteV2 — SwapsEvent (3KHMZ anomaly)",
			sig:  "4Knpk5HUzdeSvE3hmiUHcPGcZzwsMUSspUvchTj5W6dDSB1dDVewbqpwdvcqf2XcrWweckKT9j89tPHzVVNnagLd",
			expected: []legExpectation{
				{"Meteora_DLMM_Program", "3KHMZhpthXuiCcgfTv7vVu9PpEz64KAEURFwi6Lopump", "So11111111111111111111111111111111111111112", 101946555003, 5594069444},
				{"Meteora_DLMM_Program", "So11111111111111111111111111111111111111112", "F5tfztTnE4sYsMhZT5KrFpWvHmYSfJZoRjCuxKPbpump", 5594069444, 353707088201},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := solana.MustSignatureFromBase58(tt.sig)
			var maxTxVersion uint64 = 0
			tx, err := client.GetTransaction(context.TODO(), sig, &rpc.GetTransactionOpts{
				Commitment:                     rpc.CommitmentConfirmed,
				MaxSupportedTransactionVersion: &maxTxVersion,
			})
			require.NoError(t, err)
			require.NotNil(t, tx)
			require.NotNil(t, tx.Transaction)

			parser, err := NewParser(tx)
			require.NoError(t, err)

			legs, err := parser.ParseTransaction()
			require.NoError(t, err)
			require.Len(t, legs, len(tt.expected), "leg count mismatch")

			for i, exp := range tt.expected {
				leg := legs[i]
				require.NotNil(t, leg.Tx, "leg %d nil Tx", i)
				require.Equal(t, exp.protocol, leg.Tx.Protocol, "leg %d protocol mismatch", i)
				require.Equal(t, exp.inputMint, leg.Tx.InputMint.String(), "leg %d inputMint mismatch", i)
				require.Equal(t, exp.outputMint, leg.Tx.OutputMint.String(), "leg %d outputMint mismatch", i)
				require.Equal(t, exp.inputAmt, leg.Tx.InputAmount, "leg %d inputAmount mismatch", i)
				require.Equal(t, exp.outputAmt, leg.Tx.OutputAmount, "leg %d outputAmount mismatch", i)
			}
		})
	}
}
