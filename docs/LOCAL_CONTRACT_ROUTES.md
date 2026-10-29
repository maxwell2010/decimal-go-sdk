# Additional local contract routes

[Русский](LOCAL_CONTRACT_ROUTES.ru.md) | [Execution foundations](LOCAL_EXECUTION_EXTENSIONS.md) | [Validation](VALIDATION.md)

These scenarios execute genuine checked-in contract code with synthetic local accounts and balances. Normal tests do not query public RPC or submit public transactions. Local fresh-state results do not establish initialized public proxies, user permissions, Decimal consensus or settlement.

```sh
go test -run '^(TestLocalEVMCurrentChecks|TestLocalEVMWeightedSafeOwner|TestLocalEVMObservedDecimalReserveless)' .
```

## Observed current Checks routes

The [current Checks fixture](../testdata/evm/checks-current-manifest.json) pins the sampled 6318-byte implementation. Additional tests use its recipient-bound SDK signing profile, not the historical bundled artifact's check-only profile.

Token creation requires actual allowance and funding. Missing/partial allowance and insufficient token balance revert without custody, allowance, nonce or log effects. A batch with a valid first redemption and invalid second proof rolls back every transfer and status change; a subsequent valid batch succeeds and replay fails. Expired recipient proofs are rejected.

`createChecksTokenByPermit` additionally runs against the official upstream STANDARD EIP2612 token fixture. Valid creation uses the real permit and custody paths; expired or incorrectly valued authorization rolls back. This interoperability test does not claim that its upstream token is a Decimal deployment.

## Weighted Safe administration

The [Safe runtime fixture](../testdata/evm/safe-manifest.json) creates a genuine factory proxy before testing administration. A direct owner call cannot change the threshold. Signed self-calls change the threshold, add a weighted owner and remove an owner using the actual returned owner ordering. Wrong-chain signatures, outsiders and removed owners cannot authorize changes.

An impossible threshold reverts with unchanged owners, threshold, nonce and logs; the following valid operation succeeds. A current owner can approve a transaction hash without consuming the execution nonce, while a removed owner cannot. These tests cover the sampled weighted runtime and local setup, not public Safe ownership or all contract-signature/module routes.

## Observed Decimal reserveless token permit

The [token runtime manifest](../testdata/evm/token-runtime-manifest.json) records official fixed-block reserve and reserveless implementation code. The reserveless fixture is initialized through its real public ABI in empty local state. SDK CheckedPermit validates its actual domain and nonce, permit emits Approval and updates allowance/nonce, replay fails, and transferFrom spends the approved allowance. Reinitialization is rejected.

This closes the earlier upstream-only permit execution limit for the sampled reserveless Decimal code. It still does not certify public proxy state or all token routes. Reserve-token code was captured and hash-checked but is NOT executed by this suite; no reserve economics or acceptance result is inferred.
