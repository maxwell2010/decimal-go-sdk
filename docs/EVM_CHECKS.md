# EVM Checks redemption signatures

[Русский](EVM_CHECKS.ru.md) | [Official JS evidence](CHECKS_JS_EVIDENCE.md) | [Node evidence](CHECKS_NODE_EVIDENCE.md)

**Signing profiles are versioned.** The historical 6258-byte Checks artifact bundled with the pinned JS SDK rejects this helper's recipient-bound proof. The separately captured official 6318-byte implementation runtime accepts it in fresh local state, rejects a different caller and rejects replay; ECRECOVER traces match the SDK digest. This confirms the sampled code's tested signing path, not public proxy initialization, funding or settlement. See [the execution evidence](LOCAL_EXECUTION_EXTENSIONS.md).

This helper implements the reviewed official JS redemption-signing format. It is separate from historical Cosmos/RLP checks, ERC20 permits and Safe signing. No helper invokes RPC, signs a blockchain transaction or submits anything.

## Confirmed format

The official JS derives the check signer as `keccak256(UTF8(password))`, with an empty-password guard. The SDK exposes `EVMCheckSignerFromPassword`, returning a reusable `KeySigner` rather than raw key bytes. Invalid UTF-8 is rejected; Unicode is not normalized. This is direct password hashing, not a password-hardening KDF. Public example passwords are fixtures and must not control funds.

For redemption, the JS packs exactly the opaque 32-byte check hash and the 20-byte redeemer address, hashes those 52 bytes with Keccak256, then uses `Wallet.signMessage` on the 32 raw hash bytes. Therefore:

```text
inner  = keccak256(checkHash[32] || redeemer[20])
digest = keccak256("\x19Ethereum Signed Message:\n32" || inner[32])
```

`EVMCheckRedemptionHash` returns inner; `EVMCheckRedemptionDigest` returns digest. The signer callback receives the completed digest: do not personal-sign it a second time or sign its hexadecimal text. Returned EOA signatures have 65 bytes, canonical low-S and v=27/28. Callback signatures with v=0/1 are normalized after validation.

The helper adds no chain ID, contract address, nonce, amount or deadline to that payload. Whether the opaque check hash internally binds any of those values is not established by this helper. Do not invent a check-details hash from ABI fields. The node confirms a separate stored identity `keccak256(checkDetailsHash || signer)`, but does not provide the original checkDetailsHash construction.

## Run the complete offline example

```sh
go run ./examples/checks_signing
go test ./examples/checks_signing
```

[Complete Go source](../examples/checks_signing/main.go) derives a signer from a public fixture, signs for a fixture redeemer, verifies recovery and builds `redeemChecks(bytes[],bytes32[])`. Its check hash was not obtained from an issued check. The program prints only the fixture hash, signer address and calldata length. No deployment, existing balance or redemption acceptance is implied.

Core API usage inside an application function:

```go
signer, err := decimal.EVMCheckSignerFromPassword(password)
if err != nil { return err }
defer signer.Close()
signature, err := decimal.SignEVMCheckRedemption(ctx, signer, checkHash, redeemer)
if err != nil { return err }
if err = decimal.VerifyEVMCheckRedemption(checkHash, redeemer, signer.Address(), signature); err != nil { return err }
checks, err := new(decimal.Client).Checks(checksAddress) // offline construction only
if err != nil { return err }
call, err := checks.BuildSignedRedemption(redeemer, []decimal.EVMCheckRedemptionProof{
    {CheckHash: checkHash, Signer: signer.Address(), Signature: signature},
})
if err != nil { return err }
_ = call // no Send or Broadcast
```

Import `decimal "github.com/maxwell2010/decimal-go-sdk"`. Variables ctx, password, checkHash, redeemer and checksAddress are explicit application inputs. `SignEVMCheckRedemptionWith` supports an external digest-signing callback plus an expected signer address. Wrong-key results, invalid signature values and cancellation are rejected. Hash inputs are values; returned signature buffers are independent snapshots.

## Creation and redemption roles

- Creation builders receive the password-derived **check signer** in signers[]. The creator funds the check; the check signer and creator are not assumed to be the same account. DEL creation value covers amount multiplied by signer count, as corroborated by the node comments. Token permit remains a separate ERC20 authorization.
- The pinned JS ABI exposes nonces() without an address argument. In contrast, the node migration describes per-address nonce storage. These are different source/generation observations, not interchangeable getters or proof that every deployment has a global nonce. No signing helper reads or guesses a nonce, due block, amount, token or deployment; confirm the selected deployment separately.
- Obtain actual check identities from the reviewed deployment's creation return/simulation or emitted CreatedCheck events. Validate the target ABI and state separately. The helper cannot manufacture an issued check.
- The **redeemer** is the direct-call account used by the official JS profile and its payout recipient. A direct transaction must use that same sender. Contract-mediated routes require separate review of the immediate caller. Building calldata cannot enforce From.
- BuildSignedRedemption verifies every proof against that redeemer and declared check signer, rejects duplicate check hashes and preserves order. It does not establish issuance, expiry, availability, token allowance or stateful permissions.

Pure Hash/Digest functions reproduce all fixed-width inputs. Signing and proof verification additionally reject zero redeemer/expected-signer addresses as an SDK usage policy. This policy is not asserted as an independently proven contract rule.

The SDK tests use independently executed official ethers vectors for ASCII, Unicode and embedded-NUL public passwords, including both recovery IDs. See [vectors](checks-js-vectors.json). The signing format is the reviewed JS profile; it is not advertised as a universal interface for every node contract generation.

Current deployed acceptance and complete Solidity source remain separate evidence boundaries. No network transaction was sent for this implementation.

For a direct Checks transaction, its sender must be the redeemer used in the proof. Contract-mediated calls (Safe, relayers or other wrappers) require separate review of the immediate caller and are not certified by this helper. A Call does not encode a From restriction.

A creation callStatic result is a prediction, not issuance or funding. Confirm the actual creation event/state before treating its hash as an existing redeemable check.
