# Read-only deployment compatibility - 2026-10-29

These observations use official Config RPC endpoints, no wallet or signing API. The inspector sends only eth_chainId, eth_blockNumber, eth_getBlockByNumber, eth_getCode, eth_getStorageAt and eth_call. All state calls use one explicit block per network; its RPC-reported hash is checked again before saving. No writes, simulation, automatic ABI substitution or initialization is performed.

## Snapshots

- [testnet evidence](compatibility/deployments-testnet.json): chain 202020, block 22477428, RPC-reported hash `0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c`. 17 entries, including unresolved registry slugs.
- [mainnet evidence](compatibility/deployments-mainnet.json): chain 75, block 34336444, RPC-reported hash `0xd71b1667291098376553306e483d817232ef821e5e648ec4d7391cf79d871ab7`. 14 entries, including unresolved registry slugs.

## Findings

- Testnet system proxy runtime hashes and the implementation hashes present in the three historical deployment manifests still match. The sampled token beacon implementations also match their recorded code hashes. Token implementation runtime hashes are compared with implementations, never mislabeled as proxy runtime hashes.
- Mainnet is observed independently. No testnet versioned ABI profile or code hash is imported as a mainnet default.
- Checks on testnet is a 170-byte proxy at 0xb6161CAA8Bd7167C2fa67b93b041FF9a94d6a735. EIP1967 implementation storage and getImpl() agree on 0xBACA5A733002F5323aBE4c83C91B87212E7A3ca3; its runtime hash is 0x78da094992fb84bf17517277dcfc9c67ab36ba535f8879b5fd5b5614ac397bf4. owner() is nonzero, nonces() returns zero at this block.
- The configured mainnet Checks address 0x9c3326594b49AFa24db9b988Cda1258d0dcb4e40 has 6318 runtime bytes with exactly that implementation hash. Its EIP1967 implementation/beacon slots, getImpl(), owner() and nonces() are all zero. This is evidence of a different deployment shape with uninitialized-looking getters; it does not establish an initialized, funded or usable Checks proxy. Configuration was not changed and no initialization was attempted.
- Safe nonce() and getWeightThreshold() returned values on both configured addresses. getOwnersWithWeights() failed on both. This does not certify owner initialization or prove the ABI is absent; these static addresses must not be treated as accepted user Safes.
- DelegationNFT has runtime and an EIP1967 implementation on both networks. Its owner()/getImpl() probes fail on testnet but return values on mainnet. Reverts are not evidence of contract absence or complete compatibility.
- ContractCenter gas-center, bridge and bridge-v2 slugs return zero/invalid addresses on both networks. The separately established official-TheGraph testnet Bridge address still has runtime and EIP1967 implementation 0x06D7AC3874adC1f773c5165E91077c8037B789ff, while owner()/getImpl() probes fail. Missing registry entries do not prove absence.

## Reproduce

From the SDK root, choose new output paths (existing snapshots are refused):

```sh
go run ./cmd/compatibility-inspect -network testnet -block 22477428 -out docs/compatibility/deployments-testnet-new.json
go run ./cmd/compatibility-inspect -network mainnet -block 34336444 -out docs/compatibility/deployments-mainnet-new.json
go test ./cmd/compatibility-inspect
```

A zero/default -block selects a current snapshot. Network is restricted to testnet/mainnet; endpoints come from SDK Config. Historical comparisons read docs/*deployment-sources.json. Error responses are classified conservatively as read failures, not guessed contract semantics. Raw return bytes remain in the evidence; success means the RPC call returned data, not that every business rule was checked.

Beacon implementation getters are observational zero-caller reads. They do not provide the stronger caller-independent binding guaranteed by the SDK beacon loader. Top-level eth_call cannot prove nested execution behavior, future identity, stateful authorization or settlement. Block-number pinning with before/after RPC hash equality is not an EIP1898 block-hash selector or a cryptographic state proof. Decimal RPC hashes are recorded directly, not recomputed with Ethereum header hashing.

The runtime Client.BlockIdentity(ctx, number) API now reads the RPC-reported hash directly through ethclient raw RPC, or an explicitly implemented custom Backend adapter. It validates hash/height and never falls back to Header.Hash(). Header remains available for its Ethereum-shaped fields. This prevents snapshot code from substituting an Ethereum-recomputed hash for Decimal canonical identity.

Getter probes omit the From field. No-argument nonces() returning zero is a response in that default caller context, not proof of a global nonce or a particular creator nonce. Caller-dependent reads require explicit caller context and deployment review; no nonce interpretation is inferred.
