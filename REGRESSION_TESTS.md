# Regression Test Cases

这些交易用于验证新增 router / AMM 支持以及关键 bugfix 的回归测试。运行前请确保能访问 Solana mainnet RPC（需要设置 `https_proxy` 时请在运行命令前加上）。

## 快速验证脚本

```bash
cd /path/to/repo
export TEST_TX="<tx_signature>"
# 在 main.go 中临时把 txSig 替换为 TEST_TX，然后执行：
https_proxy=http://127.0.0.1:7897 go run main.go
```

## 1. Bitget Swap — Multi-leg (PumpFun AMM + ZeroFi)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `5JtAbkDqdDqKRd5dfEpYFBAFiBP6zTDwtx6kEfUJxiyK197Vgb5yYnTw7DYxjzSdbnqTr6CknpgErLADEa2SrkQh` |
| **Router** | `2UUgGySTVXmKFatH7pGQo84ZrzdSYF5zw9iqrGwBMuuj` |
| **Purpose** | 验证 Bitget Swap router 识别 + ZeroFi inner AMM 解析 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 315168076744 | 6 | `So11111111111111111111111111111111111111112` | 274135327 | 9 |
| 2 | `ZeroFi` | `So11111111111111111111111111111111111111112` | 274135327 | 9 | `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB` | 23823600 | 6 |

### Expected ProcessSwapData

- `TokenInMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenInAmount`: 315168076744
- `TokenOutMint`: `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB`
- `TokenOutAmount`: 23823600
- `AMMs`: `["ZeroFi", "PumpFun.AMM"]`

---

## 2. Arbitrage Bot (3s1r) — Circular Arbitrage

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `44JDfDCPZub9aPtgEe7z89ot7iaLvgbMC52YagQWVDsVubmXsayFo6Jpk6aG1v3E9sg9hBKcX1CUdFjotwzKuTZz` |
| **Router** | `3s1rAymURnacreXreMy718GfqW6kygQsLNka1xDyW8pC` |
| **Purpose** | 验证 Arbitrage Bot router 识别 + Meteora DAMM V2 inner 解析 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `So11111111111111111111111111111111111111112` | 2724543 | 9 | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 3159124095 | 6 |
| 2 | `Meteora` | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 3159124095 | 6 | `So11111111111111111111111111111111111111112` | 5808 | 9 |

### Expected ProcessSwapData

- `TokenInMint`: `So11111111111111111111111111111111111111112`
- `TokenInAmount`: 2724543
- `TokenOutMint`: `So11111111111111111111111111111111111111112`
- `TokenOutAmount`: 5808
- `AMMs`: `["Meteora", "PumpFun.AMM"]`

---

## 3. Arbitrage Bot (B7qnn) — Circular Arbitrage

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `5kCsHh9W6CPjxEpv9JGyvkJjSGHmPPxPtJCTk8wJB9xX81DUa7uBkRFQYUj9HeXTxsV1zTbjGCSDhQGmPBEzczZR` |
| **Router** | `B7qnnCiZd6WfNHc4becittNreSCjxqPSrKRtWc1YEZ1R` |
| **Purpose** | 验证新 Arbitrage Bot program ID 接入 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `So11111111111111111111111111111111111111112` | 15451535 | 9 | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 19312159098 | 6 |
| 2 | `Meteora` | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 19312159098 | 6 | `So11111111111111111111111111111111111111112` | 16036463 | 9 |

### Expected ProcessSwapData

- `TokenInMint`: `So11111111111111111111111111111111111111112`
- `TokenInAmount`: 15451535
- `TokenOutMint`: `So11111111111111111111111111111111111111112`
- `TokenOutAmount`: 16036463

---

## 4. Binance Wallet — Single Leg (PumpFun AMM Sell)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `2rwTmdPNtUNysZtUafAW9FBW7Vn6y2LyhQr9ZRATgPihFxFEgCUnLvRkQTGyG6g6h9cCKdLSag96DPuqYdm4j1df` |
| **Router** | `B3111yJCeHBcA1bizdJjUFPALfhAfSRnAbJzGUtnt56A` |
| **Purpose** | 验证 Binance Wallet router 识别 + PumpFun AMM Sell via ProxySwap |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 2270000000000 | 6 | `So11111111111111111111111111111111111111112` | 1954407176 | 9 |

### Expected ProcessSwapData

- `TokenInMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenInAmount`: 2270000000000
- `TokenOutMint`: `So11111111111111111111111111111111111111112`
- `TokenOutAmount`: 1954407176

---

## 5. Axiom Trade — Single Leg (PumpFun AMM BuyExactQuoteIn)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `5yQZpKcu3gmSKKwX3zMC4EWKDB9z8UFYZ4fweModAv2cSY1b38GUzmG1FQt7a39WWqdRX3S5fcnCZ2od3rZqGyGe` |
| **Router** | `FLASHX8DrLbgeR8FcfNV1F5krxYcYMUdBkrP1EPBtxB9` |
| **Purpose** | 验证 PumpFun AMM `BuyExactQuoteIn` discriminator 支持 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `So11111111111111111111111111111111111111112` | 490108694 | 9 | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 607963442552 | 6 |

### Expected ProcessSwapData

- `TokenInMint`: `So11111111111111111111111111111111111111112`
- `TokenInAmount`: 490108694
- `TokenOutMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenOutAmount`: 607963442552

---

## 6. OKX Labs 2 — Single Leg (PumpFun AMM BuyExactQuoteIn)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `2QzkwCkLd3mP2TSPQ5M7eL3qGMwZmyWzPShdUC7TrTzL5TmNRdsvKMH5GNG7L8VMfFAoGBRGt1ohSCpJqsF9rhTM` |
| **Router** | `proVF4pMXVaYqmy4NjniPh4pqKNfMmsihgd4wdkCX3u` |
| **Purpose** | 验证 OKX Labs 2 router 识别 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `So11111111111111111111111111111111111111112` | 196340513 | 9 | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 245048222036 | 6 |

### Expected ProcessSwapData

- `TokenInMint`: `So11111111111111111111111111111111111111112`
- `TokenInAmount`: 196340513
- `TokenOutMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenOutAmount`: 245048222036

---

## 6.1. Bitget DEX Aggregator — Single Leg (PumpFun AMM BuyExactQuoteIn)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `52ctk8ybpLmqfJvjPBj59d7tZsZxasREVCBEkdxbueu6ZuhD6WvrJjD4ouMEV5TTeAMSeK5yB43VUnQ3pynkHQ9k` |
| **Router** | `s7SunwrPG5SbViEKiViaDThPRJxkkTrNx2iRPN3exNC` |
| **Purpose** | 验证 Bitget DEX Aggregator router 识别 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `So11111111111111111111111111111111111111112` | 1970335967 | 9 | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 2032255254700 | 6 |

### Expected ProcessSwapData

- `TokenInMint`: `So11111111111111111111111111111111111111112`
- `TokenInAmount`: 1970335967
- `TokenOutMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenOutAmount`: 2032255254700

---

## 7. Jupiter Aggregator v6 — Single Leg (Event-Based)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `5LFEcHCGdXRn9FdmZvm1neT1jLm5T3dJToY1utcpsyHMifc7UNNJdeS6bQhXXtAwmQveR58ELyZfmi83wJinXYQH` |
| **Router** | `JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4` |
| **Purpose** | 验证 Jupiter 事件驱动解析（有 JupiterRouteEvent） |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `Jupiter` | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 114521290827 | 6 | `So11111111111111111111111111111111111111112` | 100409303 | 9 |

> 注意：此交易有 JupiterRouteEvent，所以 `Data` 字段不为 nil，Type 为 `Jupiter`。

---

## 8. Jupiter Aggregator v6 — Multi-leg (RouteV2 Fallback)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `4iaTjbw7nJ3aqeavwCyrMZvF69u8mU9zbJQMQaxifjyEayvVDUwcyHHQkPNbWhU4EAqkXSCQERR3y6DZ41jTxF1T` |
| **Router** | `JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4` |
| **Purpose** | 验证 Jupiter RouteV2 无事件时 fallback 到 inner-instruction 扫描 + Raydium CL 解析 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `PumpFun.AMM` | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 32143500000 | 6 | `So11111111111111111111111111111111111111112` | 25657769 | 9 |
| 2 | `Raydium` | `So11111111111111111111111111111111111111112` | 25657769 | 9 | `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v` | 2241421 | 6 |

### Expected ProcessSwapData

- `TokenInMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenInAmount`: 32143500000
- `TokenOutMint`: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`
- `TokenOutAmount`: 2241421

> 关键回归点：Raydium CL 的 output 必须是 **2,241,421**（USDC），而不是被后面的 fee transfer 覆盖为 33,621。

---

## 9. HumidiFi — Multi-leg (Custom AMM in Router)

| 字段 | 值 |
|------|-----|
| **Tx Signature** | `3aqonRWReqZUoZiRJuq9KX2uMUUVTKL64dTg2krM8enEwv6zw3tjm8c27HasVde3CdNofxwgpCHupSUHcSuGsubA` |
| **Router** | `B3111yJCeHBcA1bizdJjUFPALfhAfSRnAbJzGUtnt56A` (Binance Wallet) |
| **Purpose** | 验证 HumidiFi AMM 独立解析 + input/output 顺序修正 |

### Expected Swap Legs

| # | Type | Input Mint | Input Amount | Decimals | Output Mint | Output Amount | Decimals |
|---|------|-----------|-------------|----------|------------|--------------|----------|
| 1 | `HumidiFi` | `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v` | 100986821 | 6 | `So11111111111111111111111111111111111111112` | 1154784782 | 9 |
| 2 | `PumpFun.AMM` | `So11111111111111111111111111111111111111112` | 1143373865 | 9 | `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump` | 1402190646613 | 6 |

### Expected ProcessSwapData

- `TokenInMint`: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`
- `TokenInAmount`: 100986821
- `TokenOutMint`: `E95sJahssFKUk6jcWYbyfmjtcCsr4Z226HD9Qbjupump`
- `TokenOutAmount`: 1402190646613

> 关键回归点：HumidiFi 的 `InputMint` 必须是 **USDC**（因为 inner transfer 顺序是 output 先，input 后，parser 已做交换修正）。

---

## 通用回归验证清单

运行以上任意交易后，请额外检查：

1. **PumpFun AMM decimals**: E95s... 的 decimals 必须是 **6**，不能是 9（这是 `splDecimalsMap` 修复后的关键回归点）。
2. **Swap leg 顺序**: 多 leg 交易中，leg 顺序必须与交易实际执行顺序一致（`ProcessSwapData` 的 `txSwaps` 构建逻辑）。
3. **Router program ID**: `Tx.Router` 字段必须指向外层 router（如 Binance Wallet、Bitget、Axiom 等），而不是 inner AMM。
4. **No empty swap data**: 所有列出的交易必须至少解析出 1 条 swap leg，不能返回 "no swap data provided"。
5. **processRouterSwaps isolation**: 当 router 内包含多个不同 AMM（如 PumpFun + Raydium）时，各 AMM 的 transfer 不能互相污染（通过 `innerIdx` 参数隔离）。