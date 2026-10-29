# Cosmos signed-size commission convergence

`cosmos.ConvergeFee` implements bounded fixed-point commission calculation. Its primary reference is [go-smart-node/sdk/tx/tx.go, lines 28-80](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/sdk/tx/tx.go#lines-28): the node's SDK constructs and signs a transaction at the candidate fee, measures the actual signed bytes, calls the commission calculator and repeats until the fee is unchanged.

The implementation does not invent a DEL price, commission rate, currency conversion or gas formula. A caller supplies the commission callback backed by verified node parameters and the oracle price. The callback receives the signed bytes, exact signed length, candidate fee and pinned snapshot identifier. Fees are nonnegative integers bounded to uint256 and to the caller's required `MaxFee`.

## API

```go
result, err := cosmos.ConvergeFee(ctx, cosmos.FeeConvergenceOptions{
    MaxFee: approvedFeeLimit,
    Snapshot: pinnedParameterAndPriceSnapshot,
    MaxIterations: 16,
}, buildSignedTransactionAtFee, calculateCommission)
```

`FeeTxBuilder` rebuilds AND signs the exact transaction for each candidate. Its callback must preserve messages, memo, chain ID, signer order, account numbers and sequences; only the fee changes. It returns the final `TxRaw` bytes, not a sign document or an unsigned estimate. For `DirectTx`, set a fresh `[]Coin{{Denom: denom, Amount: fee.String()}}`, call `SignDirectTx`, and return its `Raw` field. The denomination and all non-fee transaction data belong to the caller's immutable snapshot. This function never broadcasts.

`CommissionQuoter` returns `CommissionQuote{Fee, Snapshot}`. Snapshot must equal the requested identifier on every iteration. It should identify both the parameter and price state, with any applicable quote expiry enforced by the callback. An arbitrary matching label is not proof of authenticity or freshness; RPC/indexer trust remains an application concern. Callback errors abort rather than silently reusing an earlier quote or delegating zero-fee calculation to the chain.

The initial fee defaults to one base unit, matching upstream. An explicit zero candidate is allowed; zero is never assumed merely because a quote fails. Convergence succeeds only when the quote equals the fee used to sign those exact returned bytes. `ConvergedFee` contains independent copies of `Fee` and `SignedBytes`, plus iteration count and snapshot identifier.

## Bounds and failures

- `MaxFee` is mandatory. Overflow, negative quotes and fee-cap violations return `ErrFeeLimit` before another signing iteration.
- Default iteration limit is 16; configurable range is 1..1024. Exhaustion returns `ErrFeeIterations`.
- Repeated candidate fees before convergence indicate oscillation and return `ErrFeeCycle`. Fees are not clamped to conceal a cycle.
- Default signed size cap is 1 MiB, configurable up to 64 MiB. Empty or oversized signed transactions are rejected.
- Snapshot mismatch returns `ErrFeeSnapshot`; a quote/build error returns the corresponding sanitized sentinel without including backend/signer error text.
- Cancellation is checked before callbacks and immediately afterwards. Callbacks must also honor their context; the SDK cannot forcibly terminate an arbitrary blocking callback.
- Every error returns an empty result. A previous iteration's transaction is never returned as a usable fallback.

Input amounts, callback fee arguments, request bytes and successful results are copied so ordinary callback mutation cannot change the iteration's reference state. As with all Go callbacks, callers must not concurrently mutate buffers during return/copy; the builder and quoter must follow their contracts.

Tests cover signed-length convergence across a fee digit boundary, bounded oscillation, uint256 overflow, fee caps, stale snapshot/quote errors, empty partial results, iteration exhaustion, size bounds, cancellation and mutation isolation. No network request, real wallet or broadcast is used.
