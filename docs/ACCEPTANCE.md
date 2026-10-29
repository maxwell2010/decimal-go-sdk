# Network acceptance resources

Implementation tests and transaction acceptance are separate results. The local
suite exercises every generated ABI wrapper, signing vectors, fee/nonce logic,
RPC wire requests and failure paths. It does not demonstrate successful execution
of every deployed business operation.

No mainnet or testnet write is part of the validation recorded so far. The
following resources are needed to run a bounded testnet write campaign:

| Family | Required resources | Acceptance evidence |
|---|---|---|
| DEL | Funded disposable sender, recipient, explicit fee/value budget | Balance delta, memo/calldata, receipt and actual fee |
| Tokens | Reserve creation funds, reservable and reserveless deployments, creator rights | Supply/reserve/CRR changes; exact-input/output limits; approvals, permits, cap and failure checks |
| NFT | Both ERC721/1155 generations, collection creator, tokens with/without reserves | Creation tuple generation, IDs/events, ownership/balances, mint/reserve/burn and batch failure behavior |
| Staking | Disposable validator target, DEL/token/NFT stake, known hold/frozen IDs | Regular/held/frozen state at pinned blocks, reset/transfer/withdraw and actual completion; maturity is not completion |
| Validator controls | Disposable validator/operator and required admin roles | Authorized controls succeed; unauthorized controls revert; no production validator changes |
| Checks | Deployment ABI, test funds and signers, expiration boundary | Nonce, issuance/redeem event hashes, invalid/repeated/expired signature failures |
| Safe | Weighted-owner deployment, owners, threshold and nonce | Domain/hash, sorted EOA signatures, threshold/weight failures and execution receipt; contract signatures remain separate |
| Bridge | Exact bridge-v2 deployment ABI, source/target test networks, fee model, valid proof producer | Source receipt plus target settlement; invalid proof/replay/wrong-chain failures |
| Contract administration | Disposable owner-controlled deployments matching each tested version | Upgrade/ownership/pause permissions and state transitions; production contracts are not acceptance fixtures |
| Legacy Cosmos | Compatible test network, account number/sequence, fee parameters and appropriate module permissions | Protobuf bytes/type URL, signer algorithm, node simulation and receipt/code |

The caller must provide the signer locally or fund a fresh disposable testnet
wallet. Secrets should not be pasted into chat. A funded wallet alone does not
grant validator, factory, bridge, oracle or owner permissions. Missing deployment
ABI, proof producers or roles must remain explicit unexecuted cases, not passing
tests or fabricated substitutes.

The EVM operation-level requirements are in `operation-matrix.json`; the legacy
inventory is documented in `LEGACY_SCOPE.md`. All fee caps and destructive test
amounts must be bounded for the selected test network before submission.
