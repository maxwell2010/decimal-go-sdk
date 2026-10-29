# Getting started

Requires Go 1.26.10 or newer. This is preview version `v0.1.0-preview.1`.

## Install

Run in a new directory. The nested module has its own release tag `cosmos/v0.1.0-preview.1`; Go selects it using the module path and ordinary version below.

```sh
go mod init example.com/decimal-demo
go get github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.1
# Optional independent Cosmos module:
go get github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.1
```
## Build calls offline

Save this complete program as `main.go` and run `go run .`. Both addresses are fixtures, not recommended deployments. Builders encode intent without signing, fetching a nonce or sending anything. The zero-value Client is used only for offline contract construction; use `Dial` before reads or preparation.

```go
package main

import (
    "fmt"
    "math/big"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    recipient, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
    if err != nil { panic(err) }
    tokenAddress, err := decimal.ParseAddress("0x2000000000000000000000000000000000000002")
    if err != nil { panic(err) }
    amount, err := decimal.ParseUnits("1.25", 18)
    if err != nil { panic(err) }
    native, err := decimal.NativeTransfer(recipient, amount, "invoice-123")
    if err != nil { panic(err) }

    // Offline builder only: no RPC backend is configured.
    builder := new(decimal.Client)
    token, err := builder.Token(tokenAddress)
    if err != nil { panic(err) }
    // Explicit raw token units; discover decimals separately for a real token.
    transfer, err := token.BuildTransfer(recipient, big.NewInt(125))
    if err != nil { panic(err) }
    fmt.Println(native.Value, string(native.Data))
    fmt.Printf("token=%s calldata=%x\n", transfer.To, transfer.Data)
}
```
## Read a balance

This separate `main.go` makes read-only requests to the configured official testnet RPC. It requires internet access, no signing key and no funds. `Dial` checks chain ID.

```go
package main

import (
    "context"
    "fmt"
    "time"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    client, err := decimal.Dial(ctx, decimal.Testnet())
    if err != nil { panic(err) }
    defer client.Close()
    address, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
    if err != nil { panic(err) }
    balance, err := client.Balance(ctx, address)
    if err != nil { panic(err) }
    fmt.Println(decimal.FormatUnits(balance, 18), "DEL")
}
```
## Prepare and send are separate

`client.Prepare(ctx, from, call, decimal.TxOptions{...})` performs RPC checks and gas/fee estimation and returns an unsigned transaction. It does not sign or broadcast. `client.Send(ctx, signer, call, opts)` signs and broadcasts and must be invoked explicitly. Preparing a transaction is not evidence that its business operation will execute successfully. Do not invoke Send or Cosmos Broadcast when following a read-only workflow. No signing or sending program is included here.

| TxOptions.TxType | Envelope | Limits |
|---|---|---|
| 0 (default) | Legacy gas price | No access list; uses GasPrice or RPC suggestion. |
| 1 | EIP-2930 | Explicit access list and gas price; requires network support. |
| 2 | EIP-1559 | Explicit FeeCap and TipCap; requires network support. BaseFeeEstimate is an explicit estimate input, not an automatic guarantee. |

All fee values use integer base units. Set `MaxFee` and appropriate gas-price bounds before any separately authorized submission. An estimated fee differs from the reserved maximum; inspect `Prepared.Fee`. Envelope construction and local tests do not establish live admission of every envelope on every Decimal deployment.

## Optional Cosmos codecs

Use this as a separate program after installing the nested module. It intentionally packs an empty historical message to demonstrate the schema and route classification; it is not a valid business transaction. Call `ValidateBasic` on populated messages and enforce signer order, roles and target-network admission separately.

```go
package main

import (
    "fmt"
    cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos"
)

func main() {
    message := cosmos.NewMsgSendCoin()
    packed, err := cosmos.Pack(message)
    if err != nil { panic(err) }
    fmt.Println(packed.TypeUrl)
    fmt.Println(cosmos.PinnedNativeRoute(packed.TypeUrl)) // false
}
```

The EVM package has 524 ABI method wrappers, including read methods and administration methods. This is not 524 business transaction types. Cosmos has 43 pinned message schemas; only eight are admitted as standalone native requests by the pinned current node ante. Codec availability does not imply current execution support.

Deployment ABI limitations and local verification results are recorded in [ADDITIONAL_ABI_EVIDENCE.md](ADDITIONAL_ABI_EVIDENCE.md) and [VALIDATION.md](VALIDATION.md). No real network transactions were submitted during this implementation and audit.
