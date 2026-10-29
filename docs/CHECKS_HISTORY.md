# Read existing EVM Checks creation events

[Русский](CHECKS_HISTORY.ru.md) | [Signing](EVM_CHECKS.md)

No signer, password or transaction submission is needed. The decoder follows the pinned official JS Checks ABI: CreatedCheck has one event-signature topic and 128 data bytes; CreatedCheckDetails has one topic and 192 bytes. Neither event has indexed payload fields. Exact ABI unpack/repack rejects noncanonical padding and trailing bytes. Unknown uint8 status/type values are preserved, not interpreted as spendability.

Run a bounded official-RPC query:

```sh
go run ./cmd/checks-inspect -network testnet -blocks 2000
go run ./cmd/checks-inspect -network mainnet -blocks 2000
```

The command resolves the upper block once (or accepts `-to BLOCK`), reads its raw RPC hash before/after the log query, and rejects a changing snapshot. Decimal raw block.hash is deliberately used instead of recomputing types.Header.Hash(). `-address` selects an explicit deployment; the network default is configuration evidence, not runtime compatibility certification. A query covers at most 2000 blocks and accepts at most 10000 matching events. It never scans the whole chain.

Library API:

```go
rows, err := client.ChecksCreationHistory(ctx, checksAddress, fromBlock, toBlock)
// Or read one previously known transaction receipt:
rows, err = client.ChecksCreationReceipt(ctx, checksAddress, creationTxHash)
```

`ChecksCreationsFromReceipt` also accepts an already fetched receipt. It requires success, positive block height, nonzero transaction/block hashes and matching event transaction/block/index identities. It ignores other event types/contracts. `DecodeChecksCreationLog` and `DecodeChecksCreationDetailsLog` decode individual mined logs. Returned data/topics and integer values are owned snapshots. Creation rows are ordered by block, transaction and log position; duplicate log positions and inconsistent block hashes fail. Repeated check hashes at distinct historical positions are preserved because ABI evidence alone does not prove an immutable check lifetime.

Fresh observations on 2026-10-29:

- Testnet chain 202020, configured Checks address, blocks 22475438..22477437: six CreatedCheck events. [Window evidence](compatibility/checks-testnet.json).
- Mainnet chain 75, blocks 34334454..34336453: zero matching events. This is absence in that window only. [Evidence](compatibility/checks-mainnet.json).
- One existing testnet transaction `0xd670470a9c382456cd9102e7b9a99ecf5a42de07bfa1ad8388615b912c649518` was fetched independently with eth_getTransactionReceipt. Its successful receipt and raw block hash matched the official node. It contains check hash `0x58d605b116ba5b5833da55ad0b89b2b20564f982d4afe2e586bfc8256fca1d4c`. [Receipt evidence](compatibility/checks-testnet-receipts.json).

Replay this saved public receipt entirely offline:

```sh
go run ./examples/checks_history
go test -run 'Checks(Creation|Receipt|History|Saved)' .
```

A CreatedCheck status describes event time. It does not show current redemption/revocation, balance, expiry, rights or finality. A successful historical receipt does not grant control of its signer or make the check spendable now. No detailsHash or checkHash is reconstructed from guessed issuance fields, and no Node-generation hash formula is imposed on a different JS generation. Current-state and deployment acceptance remain separate checks. These observations sent no transactions and loaded no user secrets.

## Complete existing-receipt reader

This only reads the official testnet RPC. The transaction hash comes from the public evidence above. No key, signing or submission is involved. Matching the RPC block hash confirms response consistency, not finality or current spendability. Install the SDK in your application, save this as main.go and run go run .

```go
package main

import (
    "context"
    "fmt"
    "math/big"
    "time"
    "github.com/ethereum/go-ethereum/common"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()
    cfg := decimal.Testnet()
    client, err := decimal.Dial(ctx, cfg)
    if err != nil { panic(err) }
    defer client.Close()
    tx := common.HexToHash("0xd670470a9c382456cd9102e7b9a99ecf5a42de07bfa1ad8388615b912c649518")
    rows, err := client.ChecksCreationReceipt(ctx, cfg.Checks, tx)
    if err != nil { panic(err) }
    for _, row := range rows {
        block, err := client.BlockIdentity(ctx, new(big.Int).SetUint64(row.Log.BlockNumber))
        if err != nil { panic(err) }
        if block.Hash != row.Log.BlockHash { panic("receipt/RPC block identity mismatch") }
        fmt.Println(row.CheckHash.Hex(), row.Signer.Hex(), block.Number, block.Hash.Hex())
    }
}
```
