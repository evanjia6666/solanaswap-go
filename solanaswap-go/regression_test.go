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
	pool       string
	poolIn     string
	poolOut    string
	amm        string
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
			// Jupiter SwapsEvent (112-byte entries with trailing Amm, added 2026-08).
			// Pre-fix decoding desynced from entry 1 onward and read mint pubkey bytes
			// as amounts (e.g. 4228277238345956038 = first 8 bytes of the USDC mint),
			// inflating production volume ~1e9x. Real amounts verified against the
			// tx token balance deltas.
			name: "Jupiter SwapsEvent 112B (Tessera V)",
			sig:  "4ohi3z3AZFckSZn6d2HfhEqXhx9N9DRTMAz9oJ4YhiDbc86dptbtDkZpxhrpGX1AYUKrVoGmvHC65qEE3E9iMWA1",
			expected: []legExpectation{
				{"Manifest", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 7081025, 7082902, "8sjV1AqBFvFuADBCQHhotaRq5DFFYSjjg1jMyVWMqXvZ", "FGQoLafigpyVb7mLa6pvsDDpDaEE3JetrzQoAggTo3n7", "CNRQ2Q5YURFcQrATzYeKUWgKUoBDfqzkDrRWf21UXCVo", "MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms"},
				{"Tessera V", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "So11111111111111111111111111111111111111112", 253785, 2475810, "FLckHLGMJy5gEoXWwcE68Nprde1D4araK4TGLw4pQq2n", "9t4P5wMwfFkyn92Z7hf463qYKEZf8ERVZsGBEPNp8uJx", "5pVN5XZB8cYBjNLFrsBCPWkCQBan5K5Mq2dWGzwPgGJV", "TessVdML9pBGgG9yGks7o4HewRaXVAMuoVj4x83GLQH"},
				{"Orca", "So11111111111111111111111111111111111111112", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 2475810, 253798, "B6LL9aCWVuo1tTcJoYvCTDqYrq1vjMfci8uHxsm4UxTR", "5PAmrLHaPnH95QqiCQ5x9Hn5MPGQZmQhKuL1kyS24r7G", "vZ7uh4khfcUHKyc1dyaDhg21jDH5p5q4Pugr3R4v4Mp", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
			},
		},
		{
			name: "Bitget Swap — Multi-leg",
			sig:  "5JtAbkDqdDqKRd5dfEpYFBAFiBP6zTDwtx6kEfUJxiyK197Vgb5yYnTw7DYxjzSdbnqTr6CknpgErLADEa2SrkQh",
			expected: []legExpectation{
				{"PumpSwap", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 315168076744, 274135327, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
				{"ZeroFi", "So11111111111111111111111111111111111111112", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 274135327, 23823600, "5AfaGqxPH115joNwSqEEzMF5BjKfkUMki6zg6TGCXLKX", "YAh1vvZPnuCjZ354LF8ria76cJaSEPvPyPYuRoxVkuH", "Fiw6wDGrD5SC9vEJWPQumb499JbPVBuVsmKbNgzM22jT", "ZERor4xhbUycZ6gb9ntrhqscUcZmAbQDjEAtCf4hbZY"},
			},
		},
		{
			name: "Arbitrage Bot (3s1r)",
			sig:  "44JDfDCPZub9aPtgEe7z89ot7iaLvgbMC52YagQWVDsVubmXsayFo6Jpk6aG1v3E9sg9hBKcX1CUdFjotwzKuTZz",
			expected: []legExpectation{
				{"PumpSwap", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 2724543, 3159124095, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
				{"Meteora_DAMM_V2", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 3159124095, 2755904, "3XT92xxgZqSyexNpoGkHFU87rFVeVK2e8wDa7sZhbvSC", "6jrK2GazooyVUW4qtJwDUSFrpi7ibSxTmefKp1nrzHz8", "6B7GDbqNorFMqsPZzLe9wPnydWkAe7uWbScvujm6aejm", "cpamdpZCGKUy5JxQXB4dcpGPiikHawvSWAd6mEn1sGG"},
			},
		},
		{
			name: "Arbitrage Bot (B7qnn)",
			sig:  "5kCsHh9W6CPjxEpv9JGyvkJjSGHmPPxPtJCTk8wJB9xX81DUa7uBkRFQYUj9HeXTxsV1zTbjGCSDhQGmPBEzczZR",
			expected: []legExpectation{
				{"PumpSwap", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 15451535, 19312159098, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
				{"Meteora_DAMM_V2", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 19312159098, 16036463, "3XT92xxgZqSyexNpoGkHFU87rFVeVK2e8wDa7sZhbvSC", "6jrK2GazooyVUW4qtJwDUSFrpi7ibSxTmefKp1nrzHz8", "6B7GDbqNorFMqsPZzLe9wPnydWkAe7uWbScvujm6aejm", "cpamdpZCGKUy5JxQXB4dcpGPiikHawvSWAd6mEn1sGG"},
			},
		},
		{
			name: "Binance Wallet — Single Leg",
			sig:  "2rwTmdPNtUNysZtUafAW9FBW7Vn6y2LyhQr9ZRATgPihFxFEgCUnLvRkQTGyG6g6h9cCKdLSag96DPuqYdm4j1df",
			expected: []legExpectation{
				{"PumpSwap", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 2270000000000, 1954407176, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
			},
		},
		{
			name: "Axiom Trade — Single Leg",
			sig:  "5yQZpKcu3gmSKKwX3zMC4EWKDB9z8UFYZ4fweModAv2cSY1b38GUzmG1FQt7a39WWqdRX3S5fcnCZ2od3rZqGyGe",
			expected: []legExpectation{
				{"PumpSwap", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 490108694, 607963442552, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
			},
		},
		{
			name: "OKX Labs 2 — Single Leg",
			sig:  "2QzkwCkLd3mP2TSPQ5M7eL3qGMwZmyWzPShdUC7TrTzL5TmNRdsvKMH5GNG7L8VMfFAoGBRGt1ohSCpJqsF9rhTM",
			expected: []legExpectation{
				{"PumpSwap", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 198300000, 245048222036, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
			},
		},
		{
			name: "Bitget DEX Aggregator",
			sig:  "52ctk8ybpLmqfJvjPBj59d7tZsZxasREVCBEkdxbueu6ZuhD6WvrJjD4ouMEV5TTeAMSeK5yB43VUnQ3pynkHQ9k",
			expected: []legExpectation{
				{"PumpSwap", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 1970335967, 2032255254700, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
			},
		},
		{
			name: "Jupiter Aggregator v6 (Event)",
			sig:  "5LFEcHCGdXRn9FdmZvm1neT1jLm5T3dJToY1utcpsyHMifc7UNNJdeS6bQhXXtAwmQveR58ELyZfmi83wJinXYQH",
			expected: []legExpectation{
				{"PumpSwap", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 114521290827, 100409303, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
			},
		},
		{
			name: "Jupiter Aggregator v6 (RouteV2)",
			sig:  "4iaTjbw7nJ3aqeavwCyrMZvF69u8mU9zbJQMQaxifjyEayvVDUwcyHHQkPNbWhU4EAqkXSCQERR3y6DZ41jTxF1T",
			expected: []legExpectation{
				{"PumpSwap", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", "So11111111111111111111111111111111111111112", 32143500000, 25657769, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
				{"Raydium", "So11111111111111111111111111111111111111112", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 25657769, 2241421, "8sLbNZoA1cfnvMJLPfp98ZLAnFSYCFApfJKMbiXNLwxj", "6P4tvbzRY6Bh3MiWDHuLqyHywovsRwRpfskPvyeSoHsz", "6mK4Pxs6GhwnessH7CvPivqDYauiHZmAdbEFDpXFk9zt", "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"},
			},
		},
		{
			name: "HumidiFi — Multi-leg",
			sig:  "3aqonRWReqZUoZiRJuq9KX2uMUUVTKL64dTg2krM8enEwv6zw3tjm8c27HasVde3CdNofxwgpCHupSUHcSuGsubA",
			expected: []legExpectation{
				{"HumidiFi", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "So11111111111111111111111111111111111111112", 100986821, 1154784782, "8sKQHfjNhvmAw94PhfvfMcytmqW6jmxvwieYyzXCCPu", "H292B1VbSvD6GuUmSvUvfQstg1Acfzog796uQ7d1ccCw", "A3C9xwv4Hfx92M5HQpxUiibSqCa5pYhD2kTwnU5fEPq", "9H6tua7jkLhdm3w8BvgpTn5LZNU7g4ZynDmCiNN3q6Rp"},
				{"PumpSwap", "So11111111111111111111111111111111111111112", "E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump", 1143373865, 1402190646613, "EbnKTMYx3UR2jUhkVhK1sR86eLV6xVUAwVr6jZpNARnf", "9jCG3eGHDLGC9MBusRtmPMdHwUcRMJGK2oiaYrUiYaqG", "DEkHtJReTcwCuf1dnbXwwHLwaRNB2jGPXkZdDNCz2CAJ", "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA"},
			},
		},
		{
			name: "Raydium+Whirlpools Program (regression)",
			sig:  "HwPpFnBuyxCLRsuJNEZ5SBHx6xgtXy9TeLUkk8KNVjNXmbZsyyYyfkvdsEeD3mgNi4TYBs3A1wrbFHDDafqdwHm",
			expected: []legExpectation{
				{"Raydium", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", 449550000, 208773150, "49iMatQtoyabsYAQc8GafVq6aeBFVDxSRH44oiatyyw6", "4JEtq7NraU9U5URcCKSv6sWRRgDSuSnUjYDqpSJSWohY", "DyKsypuzQvhi37K8UvjCMBC43h4HtW4r6jhWoqHyrSSe", "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"},
				{"Orca", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", 49950000, 23197311, "6R4r93V5fcMzc13CL2enEepDSYcr4Qx3ptZBDwudTXCo", "5TSHEwRAgHLTYkchrUNiKUL2RvuZgdh3vExbMptWrHoX", "FaHQ9Ny2U2RkcdapsKVr9pvnt4Mg7n92NdKnvyRzuibH", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
			},
		},
		{
			// Raydium AMM Router (routeUGWg…) wrapping CPMM→CLMM. The CLMM hop uses
			// the deployed versioned swap encoding (0x01 + constant, no anchor
			// sighash) and used to be dropped twice over: the family-level router
			// dedup skipped it after the CPMM leg was parsed, and the discriminator
			// whitelist rejected the versioned prefix. Both legs must resolve with
			// vaults attributed to the router.
			name: "Raydium AMM Router — CPMM→CLMM (versioned swapV2)",
			sig:  "2xQeZg6v9bWRgeVgisLSWEYmvV1FNkKMQRbTnKACwAvP32xxz93j4iJCeeDB8nMRBu2XduGQ1veew3oMaQ42ezTm",
			expected: []legExpectation{
				{"Raydium", "9RS8C1Tfa3tvG8Wv5cXqfFwcGrFBdzxrkhnjMA6qb6iy", "XsaBXg8dU5cPM6ehmVctMkVqoiRG2ZjMo1cyBJ3AykQ", 470622344, 72476, "GwmMmCnmFioqe9UqMv1sNNfr8dX7h8ATVwKFiYuiQrgk", "ncPvcKFUYXX6FjqYtbwVnkMkdfzjhMHbYA26bXGLXDt", "GyRnq5yXQzDh4PsRKoV4B9msomYDbm5DZvVYXKtzBGa1", "CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C"},
				{"Raydium", "XsaBXg8dU5cPM6ehmVctMkVqoiRG2ZjMo1cyBJ3AykQ", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 72476, 64639, "7HNwUP5rSUo9GfdDthJn4UCdw2Px6h2aprSedD7r7J3C", "9U6aKhHYFB3ViSPqQzKbkUPFg7hYahLk6njnzy15Smf6", "4Rq5cbWuaEHfzzwmbpuvkF4CbmWETXvo11aaaZm7dcQe", "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"},
			},
		},
		{
			// Bitget DEX aggregator split route: SOL split across two whirlpools
			// and one Raydium CLMM, all legs buying the same Token-2022 mint.
			// Before the orca windowing + Token-2022 transfer fixes both orca
			// legs were dropped entirely (T2022 plain transfers were not
			// collected), and had they been collected they would have come out
			// as duplicated mixed aggregates of the whole inner set.
			name: "Bitget Aggregator — split route, 2× Orca (Token-2022) + CLMM",
			sig:  "3JeWKtNvtKuTNLnrywbZSGQRga8v5soorf1qfRKi9wvjGg6F6uBuC8cSSBvjUQzrkrxqT6bBt6iaFC4RgrPnVJkt",
			expected: []legExpectation{
				{"Orca", "So11111111111111111111111111111111111111112", "9cRCn9rGT8V2imeM2BaKs13yhMEais3ruM3rPvTGpump", 61441250, 30670587, "FXyy7qDb75QxxnsnKpdSD2qaQSSm8CbYT3hEzMt7UBw5", "3oigEFcZRfJqJNQnAKxViRko6vVgcjwh2L9Xy5CW3WY7", "FkLcuMeqHozjRXMpsEtXhGqKHBpAmC4UZ9ybRq6T6xB4", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
				{"Orca", "So11111111111111111111111111111111111111112", "9cRCn9rGT8V2imeM2BaKs13yhMEais3ruM3rPvTGpump", 30248000, 15111367, "CNTPTpytHK9txrsPCvaEnc3PoN9ZVWDDcSnFSZZonMue", "DgKED8DK1byHPaAjhYguE3UmehPGJyvc7pn38KySWApd", "3nz2nXypJqTLi88Wce1ACQxhKkrXENbiTE7CvUc2mcXo", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
				{"Raydium", "So11111111111111111111111111111111111111112", "9cRCn9rGT8V2imeM2BaKs13yhMEais3ruM3rPvTGpump", 2835750, 1417489, "74UCGDNEXVH26DWfswmFbVDZA7VG9Dp7yYP2CG4F76DN", "A78CEMb3zhiLzt8rCgvPwViePeoWQy71L6vFAw4Fe2D4", "8MuB1MGgEfwHuxfHMw8bGz3yNPtQFn6w9zvTg4gS6jSY", "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"},
			},
		},
		{
			// OKX router two-hop whirlpool route (Xsf9mB→USDC→SOL). The OKX
			// path's per-family dedup used to collapse the second whirlpool
			// leg; windowing emits both with unique indexes.
			name: "OKX Router — 2× Orca two-hop",
			sig:  "5PwfLp1JstucUU7GpRNzFqocvbJMHGWMzyHNUdPKuos25djarMSFmPHJ3knE5VUWz8kfRpU4TXpFmDmqwavP5daW",
			expected: []legExpectation{
				{"Orca", "Xsf9mBktVB9BSU5kf4nHxPq5hCBJ2j2ui3ecFGxPRGc", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 3454316, 669666, "hSQxf9L7Tpwf1PKjWWSbnew2vyTPpGrfWwNa4gq2n2x", "DYVwasvF7Vvvnx8nL9oYTYBAX1pMBt7H5sGSwLXRZhn4", "8T1wnPCX1Wzg7x9nyPKimxn9vFHzt6VfUCMAFcRPA24C", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
				{"Orca", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "So11111111111111111111111111111111111111112", 133934, 1275531, "BSddxwYW73as8852ZTHRH13pbZEmZ96NBjayc5mSVtkZ", "BHqCWgG33QpPb17ryW5Hbfc4oP18UFcZ1GQdB5aAqu48", "F4CAXDT5v7F7XpqPqKqPbBpj2uuncGHXsXg9xD3CCejA", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
			},
		},
		{
			name: "Raydium swap (regression)",
			sig:  "3XgeS99txr7YDwyw14aVT1tewhQgEgBMzzxT6ZGAVPzusNrgx392wstsbgPrBxnKw6xJLtUfVrQpGvFFU4cQQfj5",
			expected: []legExpectation{
				{"Raydium", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 15367000, 33015387, "49iMatQtoyabsYAQc8GafVq6aeBFVDxSRH44oiatyyw6", "DyKsypuzQvhi37K8UvjCMBC43h4HtW4r6jhWoqHyrSSe", "4JEtq7NraU9U5URcCKSv6sWRRgDSuSnUjYDqpSJSWohY", "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"},
				{"Orca", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 124333000, 267254823, "6R4r93V5fcMzc13CL2enEepDSYcr4Qx3ptZBDwudTXCo", "FaHQ9Ny2U2RkcdapsKVr9pvnt4Mg7n92NdKnvyRzuibH", "5TSHEwRAgHLTYkchrUNiKUL2RvuZgdh3vExbMptWrHoX", "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"},
			},
		},
		{
			name: "Meteora DLMM (regression)",
			sig:  "qUMyimWMctuUAcTGFzJ8WZ7ncq3UJoAZTXaNh19He2NLyTnG47Y3rzFup42jnh8HtjbRyWxisgwgfS7etgzUmoA",
			expected: []legExpectation{
				{"Byreal CLMM", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 4513500, 9691643, "GjLusGo2z3mnXPmebhhNt9ocMDJgfdxrDFctVF8Ev3Kg", "14fhv98NmX6sDiyiLDsP2WrDHJuD4k5gd5SmiqaJgh7U", "H66hccvb28JaK6p7jqsNyFkyrc5vNzyDBzWvTveyJFds", "REALQqNEomY6cQGZJUGwywTBD2UmDT32rZcNnfxQ5N2"},
				{"Meteora_DLMM_Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 501501, 1077056, "8Xx3VeN92DsZksHs2nXMiJUcR5JBofqmbXqacoeUTj2g", "2tmb3wpP6XdRYUnedRWsR69odVfMawFsH2LeqBi7HZPk", "8KRNJjewxw7icuKf8LYPFaG2JtQyiuDH3dGwS2MZTtMJ", "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"},
				{"Manifest", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Xs8S1uUs1zvS2p7iwtsG3b6fkhpvmwz4GYU3gWAmWHZ", 10768699, 1506086, "HXEEdnwBrFyCbGpR1ikrGSPgW5YYajq5zQZTXWeL99xh", "7D4m5238oZJkXzZW6jHy7cfcJfXvRnQkbdSn5T4wcbMv", "EnQ51UtmoFch5n5bEbtTKTAqHBKSxNTr2KvnSYutY74y", "MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms"},
			},
		},
		{
			name: "OKX Router — Multi-leg (Xsc9 anomaly)",
			sig:  "34jX27qt8bMeufVQAhEHtofcMiZSMasp5DcZHvJj6o5QNxfvmcFAAckUMbrfs26NPqqYxe2M6dEQM28eaCoEe787",
			expected: []legExpectation{
				{"Meteora_DLMM_Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "So11111111111111111111111111111111111111112", 510946, 11581194, "FCn5zw4gAcfRpQgst5ThFuzBGXbbJ6RocVErgC4vJ9j1", "FNaEXnGP3hUrcJSBXBKzzDRJmitNnVwwQ6pQx6LhAumF", "HZgAwbRXeUSEjZL5nERN5mDeXTKiu6ZgbDMy4MoEHcd9", "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"},
				{"Meteora_DLMM_Program", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 188981, 407737, "8Xx3VeN92DsZksHs2nXMiJUcR5JBofqmbXqacoeUTj2g", "2tmb3wpP6XdRYUnedRWsR69odVfMawFsH2LeqBi7HZPk", "8KRNJjewxw7icuKf8LYPFaG2JtQyiuDH3dGwS2MZTtMJ", "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"},
				{"Pancake Swap", "So11111111111111111111111111111111111111112", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 11581194, 1102408, "DJNtGuBGEQiUCWE8F981M2C3ZghZt2XLD8f2sQdZ6rsZ", "bHHnvxhkzBebvqxpnzVaXSdQ1GdFeZFg8yi9YKgL7zE", "CRKZr6Y9t2hzVNWY4gJk5eig8uznyykB4RTuGSqJDw7n", "HpNfyc2Saw7RKkQd8nEL4khUcuPhQ7WwY1B2qjx8jxFq"},
				{"StableWeighted", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", 1510145, 5820395, "ASUXpwE84MKGmaTp2Sxd9ZE15qfENwS8oBH7UUkk1AwB", "2PkFYJpyum86qkAM46hZ7bNvUGq157RoaPKFrgTAWLub", "GvNR6b4nDjGJiwFj44uoxW8E7FPVgbLHm3HaBDRhVbT3", "swapFpHZwjELNnjvThjajtiVmkz3yPQEHjLtka2fwHW"},
			},
		},
		{
			name: "Jupiter RouteV2 multi-leg (regression)",
			sig:  "4jRhd8zs2pjhTCEuJ2argxZiw6wiQdHb4iK2u3CwgLmdFEp3tn21nASBKedV3544qEuqfqLcD6nydLNgt4kPQZwm",
			expected: []legExpectation{
				{"ZeroFi", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "So11111111111111111111111111111111111111112", 9659601, 103407029, "4otzWqybJZVrPWEXBJetK3hQEhFvGsip6n4bS24LEAgi", "2BnuubMg42zGMNcrL4DWN5SWP3A1571Mz4WCN2v8oHMR", "CdeigPnaYwWLqqmZ9u4Spxk8nPaDFyFQwjNuGzuJjJP6", "ZERor4xhbUycZ6gb9ntrhqscUcZmAbQDjEAtCf4hbZY"},
				{"Raydium", "So11111111111111111111111111111111111111112", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", 103407029, 4489017, "38rXq2BuwdNcfnwydw6S2srpNK9N4gVzf39f69suPMsF", "A5kXc8xveLnoVG16X7r5jYdsTa3M5seUHT7bsATkRTfN", "7GWA9RA7AfdYYsCsX3TK8qCxVXT2mkr93d2fuFohKnUC", "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK"},
				{"Manifest", "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 4489017, 9660288, "5oJtKpdv7Gh9yz9zXtHWz192gsZLoFxwQQYbm6Br6eTQ", "47HNDyx7mELe9KaAmTKDafie2hieJ1zcFwn71YkNSDbx", "AnjZBXg7KFoKezfN4qpmtmVy9Z5yTKJX43duHkQUaj2F", "MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms"},
			},
		},
		{
			name: "Jupiter SharedAccountsRouteV2 — SwapsEvent (3KHMZ anomaly)",
			sig:  "4Knpk5HUzdeSvE3hmiUHcPGcZzwsMUSspUvchTj5W6dDSB1dDVewbqpwdvcqf2XcrWweckKT9j89tPHzVVNnagLd",
			expected: []legExpectation{
				{"Meteora_DLMM_Program", "3KHMZhpthXuiCcgfTv7vVu9PpEz64KAEURFwi6Lopump", "So11111111111111111111111111111111111111112", 101946555003, 5594069444, "453Z9ggMEdMnBiMLc5hHthxC3P4qhmhFUoTqkRnHDhqr", "91bpAW3ksF8kUiyXGiv5kf3QdnoGkfNy7fUQ3b3iwMAT", "ABRnnnH21PisWKggE61pYXSEKyQ9hjxTPxAXY3jCZw48", "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"},
				{"Meteora_DLMM_Program", "So11111111111111111111111111111111111111112", "F5tfztTnE4sYsMhZT5KrFpWvHmYSfJZoRjCuxKPbpump", 5594069444, 353707088201, "Cy4ZW25mnXpwD1XUUio8V6bPjpVBDCXJ58fkcZaAKuoS", "6E1GqE5UhfAp5TzwYSJ7ZLoQj6iRHGKVESrrpKJktRLr", "EicNhnTaz71ywa3e2PicmXQ991UG7XYwnovmwdGdxar3", "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"},
			},
		},
		{
			name: "Pump.fun sell_v2",
			sig:  "1V9YFzcemLncSAfAAySCe2hevfo829Wrnt6zPvMYi5yPYwoMzfJBKVYtiJpwgSPPmeZjcBzRgJwavaGqCYjnyLM",
			expected: []legExpectation{
				{string(PUMP_FUN), "816srPYEjj2tKRWvrjKZ1gnSea3QCs1HrLkw8M2zpump", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 2236910234, 9874, "7egVXQix5F7ZSftWFRdBkGpWMq4qPccR5bokY2FWzt62", "8TUcFCbC6mSs4HNwZb2Xk2wG6m9Z6sZXX9Q3zq5h2HYC", "31fp8ihLuprW1XFPz9X1iUAvPqSsCAo8QVkYpZ7SNnQN", "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"},
			},
		},
		{
			name: "Pump.fun buy_v2",
			sig:  "UsNewgnk1WR3mqJuurs97xb1kTMngPVwQ39EXdUUTcQa1jvGt5ZwK3Fdn5izGzVAJCHNaR4a137EtJQKw7rDNq4",
			expected: []legExpectation{
				{string(PUMP_FUN), "So11111111111111111111111111111111111111112", "CWtjWZsgP5iKJ6qqb6FWBeDt5RWDZxdvn7WCHwhpump", 50000001, 1785357772752, "BN2VRinDQcMPLC8QyXRY3ZhwfRoMb4xcT83VbrvXVach", "Bdd7GpKt9EPoKs7yyRVLuUHhG76AZMapFCqmNoyAMNhm", "GJCQwY6WKSSMLkCHf7t8fEJwLFsAnXyFoJARBrc4tkMF", "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"},
			},
		},
		{
			name: "okx router + goonfi v2",
			sig:  "48drjMPpMcxnVGjmqcWuTYSg28gUEPa2QrUh39jeC5h3S7C6ZW6YKFZYHDrHKAKjoBs82uXGxfFLEWTRbzxLjAsF",
			expected: []legExpectation{
				{"GoonFi V2", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 2000000000, 2001931864, "EEUNhHsRoUVgJUFpkupmdF4v7uLUw1zhYLp7u9s8zFqG", "F9xyBfChZ2uCv7aQujJCe6gKVx7ydCmxfh2ZWrNwFoKr", "5dRazfLSjTq15r7XL6b5WBGTFtria1EHZQS7VZr7BD5V", "goonuddtQRrWqqn5nFyczVKaie28f3kDkHWkHtURSLE"},
			},
		},
		{
			name: "jupiter aggregator v6 + SolFi v2",
			sig:  "2Z7cJ7WdAmYFEuAHiTzp9vFVoC4dVSLSdifmpbyJLvdPsMRCekRbPord1RBpvwAevR9t9z2K1H2v9EGoEw7Q5nNg",
			expected: []legExpectation{
				{"SolFi V2", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 543112316, 542615278, "FkEB6uvyzuoaGpgs4yRtFtxC4WJxhejNFbUkj5R6wR32", "5bHD9xdEzJdkVuhs54mGPC9BZgUshqgMg4tqmTwhWggc", "ARWaajRJyF6PKQryJ4HLzLBfTWM2qmVQUQVtBjk6PgPc", "SV2EYYJyRz2YhfXwXnhNAevDEui5Q6yrfyo13WtupPF"},
			},
		},
		{
			name: "alphaq + manifest",
			sig:  "4hZFPHffsg5Y4utHjsXvvms5AdakSicftw1WgtT7wp42jjMDK7E2xrPatuwEniC9TJsY3CWHh3Tw3vqKJikGME3Q",
			expected: []legExpectation{
				{"AlphaQ", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 19428506, 19410685, "Pi9nzTjPxD8DsRfRBGfKYzmefJoJM8TcXu2jyaQjSHm", "GF8SKKobum6UJnhX2mLHePU38htg5vdr9zcY4jH8Pqs2", "F2KCaXcp7AoQtxTDvNEDCyMyWjSCAMWNzcyN9dsPfPs5", "ALPHAQmeA7bjrVuccPsYPiCvsi428SNwte66Srvs4pHA"},
				{"Manifest", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 8005138, 7997820, "8sjV1AqBFvFuADBCQHhotaRq5DFFYSjjg1jMyVWMqXvZ", "CNRQ2Q5YURFcQrATzYeKUWgKUoBDfqzkDrRWf21UXCVo", "FGQoLafigpyVb7mLa6pvsDDpDaEE3JetrzQoAggTo3n7", "MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms"},
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
				require.Equal(t, exp.pool, leg.Tx.Pool.String(), "leg %d pool mismatch", i)
				require.Equal(t, exp.poolIn, leg.Tx.PoolIn.String(), "leg %d poolIn mismatch", i)
				require.Equal(t, exp.poolOut, leg.Tx.PoolOut.String(), "leg %d poolOut mismatch", i)
				require.Equal(t, exp.amm, leg.Tx.Amm.String(), "leg %d amm mismatch", i)
			}
		})
	}
}
