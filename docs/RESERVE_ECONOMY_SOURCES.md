# Reserve economy official-source evidence

## Source pins and exact fetched content

Only source reads were performed. No public wallet, signature or transaction submission was used.

- Official [go-smart-node formula.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/utils/formulas/formula.go), commit `9e6c6d718d662083c4a524376a66d2c50bd4bc77`, fetched from the official Bitbucket source API. SHA-256 `44d107dba47ea7c0a958e9e165c7363560b27df98382e4aabdafee8b1a7d011f`; bytes match the pinned local module snapshot.
- Official [dsc-js-sdk call.ts](https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/src/decimalevm/call.ts), commit `6790d35e2decb0cbb06a9149a9c476c834f99223`, fetched from the official Bitbucket source API. SHA-256 `fbe697fb2c7e9456cdb1b6ae929c944781ade5949e19b0ed3db149afd98c4fcd`. The local research copy differs from this pinned response; it is not used as exact source evidence here. This document uses the fetched official bytes.
- The [observed runtime manifest](../testdata/evm/token-runtime-manifest.json) fixes chain 202020, block 22477428 and original RPC block hash `0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c`. Reserve implementation code keccak256 is `0x67ef16857177e434b860586efd9ff7cf2c57c1adf4ba50bd9310f42f9c8c2cf1`. This is code identity, not verified Solidity source or live state.

## Native formulas are not EVM rounding specifications

The pinned node's `utils/formulas/formula.go` documents the reserve-curve relations:

- Purchase output: `S * ((1 + D/R)^(CRR/100) - 1)`.
- Sale output: `R * (1 - (1 - A/S)^(100/CRR))`.
- At CRR 100, explicit branches use integer division: purchase `S*D/R`, sale `R*A/S`. The native whole-supply sale returns all reserve.

Its nonlinear branches use 100-bit `big.Float` and convert to an integer after the power operation. `x/coin/keeper/msg_server.go` invokes these functions for native coin exchange. These source facts do not establish Solidity UD60x18 precision, rounding direction, minimums, fees, or whether an EVM route allows whole-supply sale. Reusing the native implementation as an exact reserve-token EVM oracle would invent compatibility that has not been demonstrated.

## Official JS delegates quotes to the contract

Pinned `call.ts` lines 1148-1161 forward the single-argument calculateBuyOutput, calculateBuyInput, calculateSellInput and calculateSellOutput methods to the contract. Lines 310-328 forward exact-output buys and both sale modes with caller limits and native value. They do not implement an independent exact integer quote engine.

The reviewed [versioned reserve ABI](../abi/TestnetReserveToken20261029.json) contains pure four-argument quote overloads, state-dependent one-argument views, min/max supply and CRR/reserve getters, and PRBMath UD60x18 log/exp/mulDiv custom errors. This suggests embedded fixed-point Solidity arithmetic but ABI declarations alone cannot prove absence of external calls or precompiles. Genuine local execution and call tracing must establish the tested runtime's actual dependencies.

## How local assertions should be scoped

Use the actual initializer and observable getters/events to establish local parameters. Compare state-dependent and explicit-parameter quotes at the same state; then verify executed token/native deltas, reserve and supply, recipient, return values, and rollback when limits fail. Integer/rational reference checks are appropriate only for algebraically exact selected cases whose runtime behavior is confirmed. Nonlinear quote/execution agreement is useful consistency evidence, not independent proof of the exact analytic curve or every rounding boundary.

Do not seed guessed storage, replace missing dependencies with constant-return mocks, infer reserve accounting from the contract native balance alone, or transfer legacy native fees/minimums to EVM contracts. Fresh local execution does not certify proxy constructor state, public owner rights, Decimal consensus, network fee policy or settlement. The companion test report must state which CRRs, amounts and operations were actually executed rather than claiming all economics from source formulas.
