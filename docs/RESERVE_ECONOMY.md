# Local reserve-token economics

[Русский](RESERVE_ECONOMY.ru.md) | [Official sources](RESERVE_ECONOMY_SOURCES.md) | [Validation](VALIDATION.md)

The suite executes the exact reserve implementation runtime recorded in [the token manifest](../testdata/evm/token-runtime-manifest.json): chain 202020, block 22477428, code keccak256 `0x67ef16857177e434b860586efd9ff7cf2c57c1adf4ba50bd9310f42f9c8c2cf1`. The real payable initializer creates fresh local state. No live proxy storage or guessed slots are copied. Normal runs do not call public RPC or submit public transactions.

```sh
go test -run '^(TestLocalEVMReserve|TestLocalReserve)' .
```

## Execution and accounting

`TestLocalEVMReserveEconomyLinearLifecycle` covers buy, buyTokenForExactDEL, buyExactTokenForDEL, sell, sellExactTokensForDEL, sellTokensForExactDEL and burn at CRR 100. Assertions compare total supply, reserve(), actual native backing, token balances, recipient proceeds, buyer debit/refund and genuine Transfer/ReserveUpdated events. Burning reduces supply without a native payout in the tested path.

`TestLocalEVMReserveEconomyRollbackAndLimits` rejects insufficient output, inadequate exact-output budgets, over-balance sale, minimum supply/reserve violations, zero recipient, unauthorized cap changes and buying beyond the configured cap. Failed calls preserve supply, backing, balances and logs. These assertions concern the selected initialized fixture parameters, not every possible configuration.

`TestLocalEVMReserveNonlinearQuoteExecutionConcordance` and `TestLocalEVMReserveExactOutputNonlinearConcordance` execute selected buy/sell paths at CRR 10, 50 and 100. They compare current-state quotes with actual state transitions, including exact-output refunds and required burns. This is quote/execution concordance, not independent proof of the nonlinear formula or its entire rounding domain.

## Independent arithmetic checks

`TestLocalReserveOracleCRR100IntegerRounding` uses Go big.Int rational arithmetic for 96 deterministic inputs across four pure quote methods, 384 comparisons total. It checks the observed floor result with both lower and strict upper rational bounds. Selected zero and dust inputs are included. Inverse input estimators also floor in this sampled code; do not replace them with an assumed ceiling.

`TestLocalReserveOracleCRR50PerfectSquareVectors` adds four analytically exact, 18-decimal-scaled vectors with supply and reserve both 100: buy input/output 100/300 and sale output/input 50/75. These narrow independent checks supplement, rather than mathematically certify, nonlinear quote/execution agreement. The legacy node's 100-bit big.Float formulas are not used as an exact Solidity oracle.

## Event timing is not final settlement

ReserveUpdated reports the balance at the point of emission. In the tested exact-output buy, a 15-unit attached budget buys 10 tokens: the event reports reserve 115 before the 5-unit refund, while the final reserve()/native balance is 110. A sale event reports reserve 100 before paying 10 to the recipient; final backing is 90. Supply assertions follow the actual event payload, but an event's reserve must not be blindly treated as the final settled balance.

## Dust behavior is contract behavior

`TestLocalEVMReserveDustExactOutputRejectsZeroInput` checks that a zero offered budget/input is rejected and all effects roll back. A zero inverse quote alone is not evidence that an arbitrary write succeeds.

`TestLocalEVMReserveDustPositiveBudgetFollowsFloorQuote` separately preserves observed positive-budget behavior. For the selected CRR-100 ratios, offering one raw unit can mint one raw token while refunding the offered native budget, leaving net native charge zero. The selected sell counterpart pays one wei while burning zero raw tokens and reducing reserve by one wei. The test checks balances, supply, backing and logs. These are actual sampled-runtime outcomes, not SDK fixes, recommended trading strategies or proofs that repeated public execution is possible.

In-process CREATE/CALL tests charge no transaction gas. A zero net economic charge there does not mean a free public transaction.

## Signed-chain fees and limits

`TestLocalReserveSignedChainEconomyAndActualFees` uses an isolated Ethereum dev chain, chain ID 1337. It signs/mines initialize, exact-output buy with refund, sale and burn through SDK Send/WaitReceipt. Sender balance must equal initial balance minus net economic cost and actual accumulated gas fees. An impossible slippage preparation does not consume a nonce or charge a fee. The test uses an explicit ABI on chain 1337 and confirms that the network-bound testnet profile is not silently adopted there.

## Initializer, administration and quote contexts

The 18 selected cases in `TestLocalEVMReserveInitializationBoundaries` and `TestLocalEVMReserveAdministrationBoundaries` now assert outcomes and state, rather than classifying every unusual input as invalid.

For the standalone initializer, the baseline succeeds. CRR 101, minimum supply above initial mint and maximum supply one raw unit above the exposed 10^30 constant also succeed in the sampled code; initialized getters and funding are checked. CRR zero fails with Solidity division-by-zero panic 0x12. Reserve, initial mint or configured minimum below the selected one-token/one-native-unit lower bound, and maximum supply below initial mint, are rejected with rollback. These observations are not claims about factory admission policy.

For owner administration, ordinary min/max updates, minimum above current supply, maximum below current supply and maximum above the exposed constant succeed in the selected fixtures. A minimum below the selected lower bound, outsider min/max updates and reinitialization fail. Tests verify events/getters for accepted updates and preserve backing, supply, sender balance and settings on rejection. They do not infer unobserved relationships between min/max or future trade validity from setter success.

`TestLocalEVMReserveCurrentPriceRationalVectors` compares selected price reads at CRR 10/50/100, before and after a real burn, with the integer floor of `nativeBacking * 100 * 10^18 / (totalSupply * CRR)`. `TestLocalEVMReserveQuoteOverloadsMatchState` compares all four current-state quote getters with the explicit supply/reserve/CRR overloads before and after burn.

`TestLocalEVMReserveBeforeQuotesCreditedBalanceContext` explicitly credits local native balance to represent incoming value, then verifies that the Before getters remove the supplied incoming context from their reserve calculation. This credit does not invoke a deployed receive/transfer route and does not prove that sending funds to a public token has no side effects. Quotes leave the credited balance and supply unchanged. These selected-context tests do not certify all arithmetic extremes.

These local results do not certify all CRRs, all uint256 inputs, formal PRBMath correctness, reserve-token public proxy initialization, user roles, Decimal consensus, public fee policy or settlement. Available official ABI and code are used as recorded; missing dependencies must not be replaced with constant-return mocks.
