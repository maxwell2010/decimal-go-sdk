# Official JS EVM Checks evidence - 2026-10-29

The published compatibility profile is based on official dsc-js-sdk pinned commit `6790d35e2decb0cbb06a9149a9c476c834f99223` and current master `c7a72278551c517d3c0a37e170f844e8176e385f`. Their Checks signing implementations are identical. Source trees were cloned read-only and Git blob contents inspected; no SDK constructor, provider or network transaction was executed.

## Authoritative implementation

- [Pinned index.ts, lines 1862-1900](https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/src/decimalevm/index.ts#lines-1862): password derivation, creation signer addresses and redemption signatures.
- [Current index.ts, lines 1988-2026](https://bitbucket.org/decimalteam/dsc-js-sdk/src/c7a72278551c517d3c0a37e170f844e8176e385f/src/decimalevm/index.ts#lines-1988): identical algorithm.
- [Current call.ts](https://bitbucket.org/decimalteam/dsc-js-sdk/src/c7a72278551c517d3c0a37e170f844e8176e385f/src/decimalevm/call.ts): creation reads global `nonces()`, DEL value equals amount times signer count, creation hashes are returned by callStatic. Token permit is separate and uses MaxUint256 deadline in this wrapper. These RPC/submission branches were read, not invoked.

The exact algorithm is:

1. Reject an empty password. Compute private scalar bytes as Keccak256 of UTF-8 password bytes (`ethers.utils.id`). There is no Unicode normalization or slow password KDF. The scalar must be valid for secp256k1; ethers.Wallet enforces this.
2. Derive the check signer EVM address from that key for issuance.
3. Treat the creation-returned check hash as an opaque 32-byte value. Pack `bytes32 checkHash || address redeemer` into 52bytes and Keccak256 hash it.
4. Sign those 32 hash bytes using personal-sign: the final digest is Keccak256 of `0x19 || "Ethereum Signed Message:\n32" || messageHash`. This is neither EIP712 nor a personal-sign of the hex text.
5. The ethers signature is 65-byte `r || s || v`, canonical low-s, with v 27/28. Submit matching ordered signature/hash arrays to redeemChecks only through a separately authorized transaction workflow.

The signature payload has no separate chain ID, contract, amount, dueBlock or nonce field. Do not add such fields to this compatibility algorithm. They may be committed by an opaque contract-created check hash, but the JS wrapper does not establish its derivation. Binding/check existence/expiry/spend state and deployed verifier behavior require independently verified contract state/source. This evidence does not establish arbitrary deployment acceptance or a universal Checks standard. Cosmos historical checks are a different protocol.

## Executed independent vectors

[checks-js-vectors.json](checks-js-vectors.json) was produced by actually executing [checks-js-reference.cjs](checks-js-reference.cjs) with ethers 5.7.2. ASCII, Cyrillic/checkmark and embedded-NUL passwords are public deterministic fixtures. v 28 and v 27 are both represented. Private scalar strings are deliberately public test data and must never hold funds.

The ethers version and integrity match the official package-lock.json: `sha512-wswUsmWo1aOK8rR7DIKiWSw9DbLWe6x98Jrn8wcTflTVvaXhAMaB5zGAXy0GYQEQp9iO1iSHWVyARQm11zUtyg==`. All 45 installed dependency artifact versions and integrity hashes were compared with the official lock. Identical packages may be nested at different paths by npm hoisting. Overrides in the isolated reference package select the exact upstream versions; initial default resolution differences were corrected and the reference re-executed with byte-identical results. npm installation used --ignore-scripts --no-audit --no-fund. Only utilities and an offline Wallet were used, never the full SDK or provider.

Reproduce in a disposable directory: copy [checks-js-reference-package.json](checks-js-reference-package.json) to package.json and [checks-js-reference-lock.json](checks-js-reference-lock.json) to package-lock.json; run npm ci --ignore-scripts --no-audit --no-fund, then `node /absolute/path/checks-js-reference.cjs /absolute/path/node_modules/ethers`. Compare parsed JSON with the fixture. Runtime Go SDK users do not need Node/npm.

## Git blob SHA-256

| Source | Pinned | Current master |
|---|---|---|
| src/decimalevm/index.ts | 0daa45cda32cb8100a628d7c442e190a1defbe46ea935fdb377d780f68036289 | 6bfb9e128f179350fe5c451cb8ba6a8c0fd1950556079519f1ec1c97046c3dd9 |
| src/decimalevm/call.ts | fbe697fb2c7e9456cdb1b6ae929c944781ade5949e19b0ed3db149afd98c4fcd | 8e2f6d548288e4ca6744774b6d5d78f8277da52a1682363138342cf98d76d858 |
| src/decimalevm/abi/Checks.json | 9504852077e385f2163005fa18e4dc58b4a49005cf68f9b5d5582ad1415bc33b | same |
| package-lock.json | 89ad7630217294e951b9ade3a1f4e505f8cdee768e39e4e6e3a524a3cf90298b | same |

The raw Checks artifact is wrapped. After the SDK normalizer is applied, its ABI equals gosdk/abi/Checks.json exactly for both revisions. No default ABI was changed and no incorrect API-provided Token ABI was imported.
