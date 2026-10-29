# Cosmos descriptor and source notices

The independent Go implementation is MIT licensed (LICENSE). Decimal node-derived definitions and source compatibility predicates retain the node MIT notice in THIRD_PARTY_NODE_LICENSE.

Embedded protobuf interfaces are compiled from the official inputs recorded with per-file hashes in schema/sources.json. The source repositories and revisions are reproducible through tools/compile.go.

- Decimal go-smart-node 9e6c6d718d662083c4a524376a66d2c50bd4bc77: BIT TEAM GROUP LTD, MIT.
- Cosmos SDK v0.46.16, Cosmos protobuf definitions and Tendermint interfaces: Apache-2.0; accompanying text in LICENSES/Cosmos-SDK-Apache-2.0.txt.
- Decimal Ethermint v0.22.13 ethsecp256k1 key interface: LGPL-3.0; accompanying text in LICENSES/Ethermint-LGPL-3.0.txt. Original interface source: https://github.com/decimalteam/ethermint/blob/v0.22.13/proto/ethermint/crypto/v1/ethsecp256k1/keys.proto .
- GoGo protobuf option schema: Copyright 2013 The GoGo Authors, BSD; notice in LICENSES/GoGo-Protobuf-BSD.txt.
- Google API annotations: Apache-2.0. Standard Protocol Buffers definitions and Go protobuf compiler support retain their upstream BSD notices; the Go protobuf license is in LICENSES/Go-Protobuf-BSD.txt.

Descriptors are interface data, not redistributed node executable binaries. Runtime dependencies are installed separately under their own licenses; the node/Ethermint reference implementations remain isolated offline tools. No production wallet material is included.
