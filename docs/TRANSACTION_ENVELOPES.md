# Transaction envelopes and manual workflows

[Русский](TRANSACTION_ENVELOPES.ru.md)

## Runnable unsigned envelopes

From the repository root:

```sh
go run ./examples/envelopes
go test ./examples/envelopes
```

[Example source](../examples/envelopes/main.go) calls the real `Client.Prepare` with an in-process `Backend`. There is no network transport or signer. Backend submission and receipt methods panic if accidentally called. All three cases prepare the same DEL intent independently using pending nonce 7. `Prepare` does not reserve that nonce; these are alternative serialization examples, not three transactions to submit together.

| Type | Explicit options | Fixture result |
|---|---|---|
| 0 Legacy | `TxType: types.LegacyTxType`, suggested price 10 wei | Gas 23100, estimated fee 210000 wei, reserved fee 231000 wei |
| 1 EIP-2930 | `TxType: types.AccessListTxType`, `AccessList`, `GasPrice: 10` | Same fixture fee; typed chain ID 202020 and access list retained |
| 2 EIP-1559 | `TxType: types.DynamicFeeTxType`, `FeeCap: 20`, `TipCap: 2`, `BaseFeeEstimate: 5` | Effective estimate 7 wei, estimated fee 147000 wei, reserve 462000 wei |

The backend returns estimated gas 21000 for every case. The SDK's default 11000 basis-point multiplier gives gas limit 23100. A real access list changes intrinsic gas; the fixture does not model execution. Dynamic reserve uses the cap, while estimated fee uses `min(cap, base estimate + tip)`. Omitting `BaseFeeEstimate` leaves dynamic estimated fee unknown. All values are integer base units.

Tests check type, nonce, unsigned signature fields, gas, fee arithmetic, chain ID, access list and binary round trips. Constructing these envelopes does not establish that a particular Decimal deployment accepts typed transactions. Type 0 is the default; types 1 and 2 require explicit opt-in. Other EVM envelope types are rejected by `Prepare`.

## Manual workflow API map

The following are exact public helpers, not automatic submission workflows. Supply reviewed addresses and deployment ABI versions. Offline builders do not establish roles, balances or successful execution. [All ABI builders](API.md), [versioned catalogs](token-deployment-catalog.json), [acceptance boundaries](ACCEPTANCE.md).

| Workflow | Public API and required inputs | Boundary |
|---|---|---|
| DEL transfer | `NativeTransfer(to, value, memo)` | `value` is wei; memo becomes data. Runnable [offline transfer](../examples/offline_transfer/main.go). |
| DEL burn | `NativeTransfer(common.Address{}, value, "")` | Zero-address transfer expresses the burn intent; the builder does not send it. |
| DEL multisend | `client.BuildMultisendDEL([]decimal.Recipient{...})` | One recipient uses native transfer; multiple recipients use configured Multicall and total DEL value. Generic ERC20 allowance to Multicall is prohibited. |
| ERC20 transfer | `token.BuildTransfer(recipient, amount)` after `client.Token(address)` | Verify actual token decimals. Runnable [token transfer](../examples/token_transfer/main.go). |
| Reset approval | `client.PlanERC20Allowance(ctx, token, owner, spender, amount, action, true)` | Reads current allowance. `AllowancePlan.Approvals` may contain zero-reset then target approval. Each needs its own successful receipt before proceeding; `Action` remains separate. |
| ERC20 permit signing | `PermitTypedData(name, version, chainID, token, permit)` then `key.SignTypedData(ctx, typed)` and `SplitSignature(bytes)` | Raw signing only. `client.CheckedPermit(ctx, key, token, name, version, permit)` additionally reads and checks nonce/domain. Owner, spender, nonce and deadline must match the intended deployment. |
| Safe signing | `NewSafeTransaction(call, nonce)`, `tx.Hash(chainID, safe)`, `tx.Sign(ctx, key, chainID, safe)` | Signature alone does not prove authorization. `ReadBigSafePolicy` and `PackBigWeightedSafeSignatures` check explicit owner weights/threshold/nonce; `BuildExecution(contract, signatures)` requires the reviewed execution ABI. |
| EVM Checks | `client.Checks(address)`, `BuildCreateChecksDEL`, `BuildCreateChecksToken`, `BuildCreateChecksTokenByPermit`, `BuildRedeemChecks` | Creation/redeem builders consume ABI inputs. There is no universal EVM check-message signing helper: do not invent a digest or reuse Safe/ERC20 permit signatures. Reviewed deployment-specific signing payloads are required. |
| Historical Cosmos check signing | Nested module `cosmos.IssueCheck(ctx, chainID, denom, amount, nonce, dueBlock, passphrase, issuerKey, sign)` | Separate historical protocol, not an EVM Checks signature. Use its deterministic fixtures and callback contract; never production credentials in examples. |

For allowance ordering, weighted/contract Safe signatures and runtime-bound restricted multisend, see [Safe workflows](SAFE_WORKFLOWS.md). For protobuf constructors and historical routes, see [Cosmos README](../cosmos/README.md). No example here calls `Send` or `Broadcast`, loads a user key, or writes to a network.
