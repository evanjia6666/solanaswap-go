---
name: diagnose-abnormal-price
description: Diagnose abnormal K-line price issues in the solanaswap-go parser. Use when a token shows an abnormally high/low price on the K-line, user reports "异常价格" or "abnormal price", or a swap tx produces wrong token amounts.
---

# Diagnose Abnormal Price

Systematic workflow for finding root causes of abnormal K-line prices caused by swap parser bugs.

## Phase 1 — Locate the transaction

### 1.1 Query the DEX API for swap txs

Use the internal DEX test API to list swap transactions for the token near the abnormal timestamp:

```bash
curl -s --location 'https://testapi.ourdex.com/solana/96c1db8008bbfe4fa8bfdac1118d86f2' \
--header 'Authorization: Bearer <TOKEN>' \
--header 'Content-Type: application/json' \
--data '{
    "id": 2,
    "method": "dex_txsQuery",
    "params": ["<TOKEN_MINT>", {"target":"token/swap","page":1,"size":200,"rawData":""}],
    "jsonrpc": "2.0"
}'
```

### 1.2 Find the abnormal tx

Filter results by `txTime` close to the abnormal timestamp (CST = UTC+8, convert accordingly). Look for tokens with `price` that is orders of magnitude off from the expected range.

### 1.3 Fetch the raw transaction

Save the full RPC response with the transaction:

```bash
curl -s -X POST '<SOLANA_RPC>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"getTransaction","params":["<SIGNATURE>", {"encoding":"json","maxSupportedTransactionVersion":0}]}' \
  > /tmp/anomaly_tx.json
```

Wrap for parser compatibility:

```python
import json
raw = json.load(open('/tmp/anomaly_tx.json'))
wrapper = {'jsonrpc':'2.0','result':raw['result'],'id':1}
json.dump(wrapper, open('/tmp/anomaly_tx_wrapped.json','w'))
```

## Phase 2 — Run the parser

### 2.1 Write a Go test

Use `anomaly_3khm_test.go` as template. Create `anomaly_<token>_test.go`:

```go
func TestAnomaly<Token>(t *testing.T) {
    data, _ := os.ReadFile("/tmp/anomaly_tx_wrapped.json")
    var rpcResp struct { /* ... */ }
    json.Unmarshal(data, &rpcResp)
    var rawResult map[string]interface{}
    json.Unmarshal(rpcResp.Result, &rawResult)
    txJSON, _ := json.Marshal(rawResult["transaction"])
    metaJSON, _ := json.Marshal(rawResult["meta"])
    var tx solana.Transaction
    json.Unmarshal(txJSON, &tx)
    var meta rpc.TransactionMeta
    json.Unmarshal(metaJSON, &meta)
    parser, _ := NewTransactionParser(&tx, &meta)
    transactionData, _ := parser.ParseTransaction()
    // Print each leg: Protocol, InputMint, InputAmount, OutputMint, OutputAmount
    // Run ProcessSwapData and print consolidated result
}
```

### 2.2 Run the test

```bash
go test -run 'TestAnomaly<Token>' -v -count=1 ./solanaswap-go/
```

### 2.3 Interpret the output

- **Leg count:** If the parser finds fewer legs than expected, events are being missed
- **Amounts:** Abnormal amounts (e.g. 247 instead of 12M) indicate partial/incomplete event parsing
- **Protocol:** "Jupiter" as protocol (instead of a specific AMM) means AMM inference failed
- **Price:** Compare the final price against the DEX API expected value

## Phase 3 — Inspect the transaction internals

### 3.1 Check instruction structure

```python
# List top-level instructions and their program IDs
# List inner instructions and their program IDs + data
# Identify which programs are AMMs vs utilities
```

### 3.2 Check for event discriminators

Key discriminators in Jupiter v6:

| Discriminator | Bytes | Hex | Meaning |
|---|---|---|---|
| AnchorSelfCPI | `228,69,165,46,81,203,154,29` | `e445a52e51cb9a1d` | Anchor self-CPI prefix |
| SwapEvent (old) | `64,198,205,232,38,8,113,226` | `40c6cde8260871e2` | Old 16-byte swap event |
| SwapsEvent (new) | `152,47,78,235,192,96,110,106` | `982f4eebc0606e6a` | Vec<SwapEventV2> event |
| SharedAccountsRouteV2 | `209,152,83,147,124,254,216,233` | `d19853937cfed8e9` | Outer instruction discriminator |

### 3.3 Decode SwapsEvent data

When a Jupiter inner instruction data starts with `e445a52e51cb9a1d` + `982f4eebc0606e6a`:

```
[8B AnchorSelfCPI] [8B SwapsEventDisc] [4B vec_len] [entries...]
```

Each SwapEventV2 entry (80 bytes):
```
[32B input_mint] [8B input_amount LE] [32B output_mint] [8B output_amount LE]
```

### 3.4 Check log messages

Look for `Program data:` logs. The base64 data after decoding should start with event discriminators. These are the events emitted by Jupiter via Anchor's event mechanism.

## Phase 4 — Identify root cause

### Common failure modes

1. **Jupiter SwapsEvent not detected** — parser only checks for `isSharedAccountsV2` gate but the tx is RouteV2. Events in inner instructions go unprocessed.
2. **Unknown AMM program** — a Meteora DLMM fork (e.g. `REALQq...`) not recognized by `isKnownAMM`. processRouterSwaps skips it, capturing only partial amounts.
3. **AMM inference mismatch** — `collectAMMsFromInnerInstructions` returns fewer AMMs than SwapsEvent legs, causing wrong protocol mapping.
4. **Duplicate leg with wrong amounts** — fallback path processes a partial swap from an AMM that isn't the full route.

### Decode event amounts to verify

```python
import base64, struct
data = base64.b64decode(event_log)
# Skip discriminator (8B), parse: amm(32), in_mint(32), in_amt(8), out_mint(32), out_amt(8)
```

## Phase 5 — Fix and verify

### 5.1 Code changes

| File | Purpose |
|---|---|
| `event_jupiter.go` | SwapsEvent detection gate, AMM inference, parseJupiterEvents aggregation |
| `checks.go` | Event discriminator checks |
| `consts.go` | New program ID constants |
| `parse_transfer.go` | Add missing programs to `isKnownAMM` |
| `parser.go` | Add AMM variants to processRouterSwaps / ParseTransaction switches |

### 5.2 Key coding patterns

- **SwapsEvent gate:** `isSharedAccountsV2 || hasUnknownAMMsInInnerInstructions` — trigger SwapsEvent parsing when processRouterSwaps can't handle all AMMs
- **AMM collection:** Filter out Jupiter, DFlow, utility programs. Collect remaining as AMMs.
- **AMM inference:** Map `inferredAMMs[ei]` to SwapEventV2 entry `ei` by order of appearance

### 5.3 Verification

```bash
# Run the anomaly test
go test -run 'TestAnomaly<Token>' -v -count=1 ./solanaswap-go/

# Run ALL regression tests (must pass before considering work done)
SOLANA_RPC_URL='<RPC>' go test -run 'TestRegression' -v -count=1 -timeout 300s ./solanaswap-go/
```

### 5.4 Acceptance criteria

- [ ] Anomaly test shows correct leg count and amounts matching decoded events
- [ ] ProcessSwapData output price matches DEX API expected price (~$X/token)
- [ ] All existing regression tests pass
- [ ] No new "Jupiter" protocol for legs that should have specific AMM names

## Reference: Key program IDs

| Program | Address | Type |
|---|---|---|
| Jupiter v6 | `JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4` | Aggregator |
| Meteora DLMM (LBUZK) | `LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo` | AMM |
| Meteora DLMM (King7ki) | `King7ki4SKMBPb3iupnQwTyjsq294jaXsgLmJo8cb7T` | AMM |
| Meteora DLMM (REALQq) | `REALQqNEomY6cQGZJUGwywTBD2UmDT32rZcNnfxQ5N2` | AMM |
| Orca Whirlpool | `whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc` | AMM |
| Pancake Swap | `HpNfyc2Saw7RKkQd8nEL4khUcuPhQ7WwY1B2qjx8jxFq` | AMM (Orca fork) |
| Raydium V4 | `675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8` | AMM |
| DFlow Aggregator v4 | `DF1ow4tspfHX9JwWJsAb9epbkA8hmpSEAtxXy1V27QBH` | Aggregator (Jupiter-compatible) |
| Pool fee helper | `pfeeUxB6jkeY1Hxd7CsFCAjcbHA9rWtchMGdZ6VojVZ` | Utility (non-AMM) |
