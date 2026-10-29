# Runnable examples

Run commands from the repository root with Go 1.26.10 or later.

- `go run ./examples/offline_transfer` builds a DEL transfer intent and prints its value and memo bytes.
- `go run ./examples/token_transfer` builds ERC20 `transfer` calldata using fixture addresses and a fixture decimal count.
- `go run ./examples/read_balance` reads a public testnet balance. Set `WALLET_ADDRESS` to a public EVM address first.

PowerShell example:

```powershell
$env:WALLET_ADDRESS = '0x1000000000000000000000000000000000000001'
go run ./examples/read_balance
```

The offline programs require no RPC. The balance program uses the official testnet endpoint, checks its chain ID and has a 15-second timeout. Transfer and ABI-envelope examples do not sign transactions. The Checks example signs only a public fixture message; it never loads user credentials or submits a transaction. Fixture token and recipient addresses do not establish any deployed contract or ownership.

[Русский](README.ru.md)

- `go run ./examples/envelopes` builds unsigned EVM types 0/1/2 using the SDK and an in-process Backend.
- `go run ./examples/transactions` calls all 260 ABI write builders in 20 profiles, entirely offline.
- `go -C cosmos run ./examples/messages` validates and encodes all 43 public schema fixtures (8 current native routes, 35 historical/schema routes).

[Install/update](../docs/INSTALL.md), [per-builder code](../docs/TRANSACTIONS.md), [envelopes/workflows](../docs/TRANSACTION_ENVELOPES.md), [Cosmos payloads](../cosmos/TRANSACTIONS.md). Signature/proof/role placeholders are encoding fixtures; no business execution is certified.

- `go run ./examples/checks_signing` signs a public fixture proof and builds calldata without RPC or submission. [Signing format](../docs/EVM_CHECKS.md).

- `go run ./examples/checks_history` decodes a saved public creation receipt offline, with no key or RPC. [History guide](../docs/CHECKS_HISTORY.md). The live CLI `go run ./cmd/checks-inspect -network testnet -blocks 2000` performs a bounded read only; empty output is window-specific.
