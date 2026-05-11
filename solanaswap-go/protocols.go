package solanaswapgo

import "github.com/gagliardetto/solana-go"

var AMMProtocolNames = map[solana.PublicKey]string{
	RAYDIUM_V4_PROGRAM_ID:                     "Raydium",
	RAYDIUM_CPMM_PROGRAM_ID:                   "Raydium",
	RAYDIUM_CONCENTRATED_LIQUIDITY_PROGRAM_ID: "Raydium Concentrated Liquidity",
	RAYDIUM_LAUNCHLAB_PROGRAM_ID:              "Raydium",
	ORCA_PROGRAM_ID:                           "Orca",
	METEORA_PROGRAM_ID:                        "Meteora_DLMM_Program",
	METEORA_POOLS_PROGRAM_ID:                  "Meteora Pools Program",
	METEORA_DLMM_PROGRAM_ID:                   "Meteora_DLMM_Program",
	BYREAL_CLMM_PROGRAM_ID:                    "Byreal CLMM",
	METEORA_DAMM_V2:                           "Meteora_DAMM_V2",
	Meteora_Dynamic_Bonding_Curve_Program:     "Meteora Dynamic Bonding Curve",
	PUMPFUN_AMM_PROGRAM_ID:                    "PumpFun.AMM",
	PUMP_FUN_PROGRAM_ID:                       "PumpFun.AMM",
	ZEROFI:                                    "ZeroFi",
	HUMIDIDI_PROGRAM_ID:                       "HumidiFi",
	PANCAKE_SWAP_PROGRAM_ID:                   "PancakeSwap",
	MANIFEST_PROGRAM_ID:                       "Manifest",
	PHOENIX_PROGRAM_ID:                        "Phoenix",
}

func ProtocolName(amm solana.PublicKey) string {
	if name, ok := AMMProtocolNames[amm]; ok {
		return name
	}
	return amm.String()
}
