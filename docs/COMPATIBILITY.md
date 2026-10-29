# Read-only compatibility matrix

This is an observation matrix, not a readiness percentage. Runtime code, getter data and successful eth_call do not certify a whole ABI, user authorization, signed-transaction admission or settlement. Live inspectors did not load keys, sign or submit public-network transactions. Separate offline tests use synthetic signing vectors.

[JSON matrix](compatibility/matrix.json) | [Deployments](DEPLOYMENT_COMPATIBILITY.md) | [Simulations](SIMULATION_COMPATIBILITY.md) | [Checks history](CHECKS_HISTORY.md)

524 ABI methods / 260 builders / 264 readers / 20 profiles. 93 reference JS operations. Local wire tests: `TestAllGeneratedABIWrappers`, recorded pass with a source fingerprint in [local-validation.json](compatibility/local-validation.json). [Validation](VALIDATION.md).

| Network | Block | Code addresses | Getter responses / failures | Successful limited eth_call | Checks events in bounded window |
|---|---|---|---|---|---|
| testnet | 22477428 | 14 | 26 / 6 | 2 | 6 |
| mainnet | 34336444 | 11 | 19 / 1 | 1 | 0 |

**Mainnet Checks needs deployment review:** the configured address has the testnet implementation runtime and zero slots/getImpl/owner. An initialized usable proxy is not established. Configuration was not changed.

Successful simulations cover self-transfer only: testnet reserve/reserveless token 1 raw unit and mainnet native DEL 1 wei. They do not certify external recipients, burns, allowance, permits, roles or bridges. Testnet native self-transfer was not tested because no suitable EOA was found in 32 selected blocks. Control of public From accounts is not claimed.

Six testnet CreatedCheck events and one separately verified receipt were observed. Zero mainnet events means only absence in the selected 2000-block window. Historical creation is not current spendability. History and deployment snapshots have distinct explicit ranges and RPC hashes.

| Profile | Builders / readers | Testnet targets / getter responses / simulated signatures | Mainnet targets / getter responses / simulated signatures |
|---|---|---|---|
| Checks | 7 / 3 | 1 / 3 / 0 | 1 / 3 / 0 |
| ContractCenter | 10 / 9 | 1 / 2 / 0 | 1 / 2 / 0 |
| Delegation | 34 / 18 | 1 / 2 / 0 | 1 / 2 / 0 |
| DelegationNFT | 23 / 14 | 1 / 0 / 0 | 1 / 2 / 0 |
| ERC1155 | 14 / 20 | 0 / 0 / 0 | 0 / 0 / 0 |
| ERC721 | 16 / 17 | 0 / 0 / 0 | 0 / 0 / 0 |
| MasterValidator | 18 / 8 | 1 / 2 / 0 | 1 / 2 / 0 |
| Multicall | 1 / 0 | 1 / 0 / 0 | 1 / 0 / 0 |
| NFTCenter | 8 / 8 | 1 / 1 / 0 | 1 / 1 / 0 |
| TestnetContractCenter20261029 | 11 / 13 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetMasterValidator20261029 | 20 / 9 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetNFTCenter20261029 | 15 / 14 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetReserveToken20261029 | 15 / 34 | 1 / 4 / 1 | 0 / 0 / 0 |
| TestnetReservelessToken20261029 | 10 / 17 | 1 / 5 / 1 | 0 / 0 / 0 |
| TestnetSafe20261029 | 15 / 18 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetSafeFactory20261029 | 3 / 2 | 1 / 0 / 0 | 0 / 0 / 0 |
| TestnetSafeMultiSend20261029 | 1 / 0 | 1 / 0 / 0 | 0 / 0 / 0 |
| TestnetTokenCenter20261029 | 13 / 14 | 1 / 4 / 0 | 0 / 0 / 0 |
| Token | 13 / 32 | 2 / 4 / 0 | 0 / 0 / 0 |
| TokenCenter | 13 / 14 | 1 / 4 / 0 | 1 / 4 / 0 |

Targets count observed addresses, not whole-ABI binding. Versioned testnet profiles are not adopted on mainnet. Unsampled ERC721/1155 instances are not replaced by NFTCenter observations. Getter responses are raw RPC bytes, and a zero owner is not initialization. Per-method records and all 93 operation statuses are in JSON.

Remaining work: authoritative interfaces for missing generations; valid state/proof/role inputs for other simulations; separately authorized funded-testnet acceptance. Use Client.BlockIdentity rather than types.Header.Hash() for the RPC block identity. Before/after equality is not a cryptographic state proof or finality guarantee.


Deployment getter probes omit From. Their results do not establish global nonce semantics or a particular creator account state. Use an explicit caller and ReadFromAt for reviewed caller-dependent getters. Existing creation hashes do not supply an authorized redemption proof.
