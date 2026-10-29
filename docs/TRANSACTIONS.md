# Transaction builder examples

These generated examples execute every catalogued ABI write builder without RPC, keys, signing or broadcasting. Run `go run ./examples/transactions` from the SDK checkout. Only a summary is printed. Run `python tools/generate_transaction_examples.py --check` to verify regeneration and `go test ./examples/transactions` for catalog completeness and selector/ABI checks.

Fixtures demonstrate ABI encoding only. Addresses, signatures, permits, proofs, amounts and tuple fields are placeholders, not executable business transactions or evidence of authorization. Values are raw integer base units. A payable builder's first argument is native value; it is not an extra ABI argument. Array fixtures have one element unless a fixed length requires more. Never send these fixtures.

The 524 ABI methods include readers and administration methods. This page covers only write builders; method coverage is not a count of distinct business transaction types. Versioned profiles are explicit interfaces and do not replace pinned profiles. Missing current authoritative ABI remains documented in [ADDITIONAL_ABI_EVIDENCE.md](ADDITIONAL_ABI_EVIDENCE.md).

## Shared setup

Snippets below run within a function with the imports and variables below; each profile section also constructs `contract`. Complete compiled programs are in [examples/transactions](../examples/transactions).

```go
import (
 decimal "github.com/maxwell2010/decimal-go-sdk"
 "github.com/ethereum/go-ethereum/common"
 "math/big"
)
offline := new(decimal.Client)
fixtureAddress := common.HexToAddress("0x1000000000000000000000000000000000000001")
```

## Native transfer and burn

These native calls are separate from ABI builder coverage. `NativeTransfer` accepts integer DEL base units and a memo. A zero-address transfer represents the SDK burn workflow; it still requires separately authorized preparation and execution before it has any chain effect.

```go
transfer, err := decimal.NativeTransfer(fixtureAddress, big.NewInt(1), "fixture")
if err != nil { panic(err) }
burn, err := decimal.NativeTransfer(common.Address{}, big.NewInt(1), "")
if err != nil { panic(err) }
_, _ = transfer, burn
```

## Checks

[Go source](../checks_generated.go), [ABI](../abi/Checks.json).

```go
contract, err := offline.Checks(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildCreateChecksDEL`

`createChecksDEL(address[],uint256,uint256,uint256)`

```go
call, err := contract.BuildCreateChecksDEL(big.NewInt(1), []common.Address{fixtureAddress}, big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildCreateChecksToken`

`createChecksToken(address[],uint256,uint256,uint256,address)`

```go
call, err := contract.BuildCreateChecksToken([]common.Address{fixtureAddress}, big.NewInt(1), big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildCreateChecksTokenByPermit`

`createChecksTokenByPermit(address[],uint256,uint256,uint256,address,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildCreateChecksTokenByPermit([]common.Address{fixtureAddress}, big.NewInt(1), big.NewInt(1), big.NewInt(1), fixtureAddress, big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildRedeemChecks`

`redeemChecks(bytes[],bytes32[])`

```go
call, err := contract.BuildRedeemChecks([][]byte{[]byte{1}}, [][32]byte{[32]byte{1}})
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

## ContractCenter

[Go source](../contractcenter_generated.go), [ABI](../abi/ContractCenter.json).

```go
contract, err := offline.ContractCenter(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildMarkReserve`

`markReserve((address,uint256))`

```go
call, err := contract.BuildMarkReserve(decimal.ContractCenterMarkReserveArg0{ContractAddress: fixtureAddress, Amount: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildMarkReserves`

`markReserves((address,uint256)[])`

```go
call, err := contract.BuildMarkReserves([]decimal.ContractCenterMarkReservesArg0{decimal.ContractCenterMarkReservesArg0{ContractAddress: fixtureAddress, Amount: big.NewInt(1)}})
if err != nil { panic(err) }
_ = call
```

### `BuildMigrateReserves`

`migrateReserves()`

```go
call, err := contract.BuildMigrateReserves(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildMigrateReservesPage`

`migrateReservesPage(uint256,uint256)`

```go
call, err := contract.BuildMigrateReservesPage(big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetAddress`

`setAddress(string,address)`

```go
call, err := contract.BuildSetAddress("fixture", fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractAddress`

`setContractAddress(string,address)`

```go
call, err := contract.BuildSetContractAddress("fixture", fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetMigrating`

`setMigrating(bool)`

```go
call, err := contract.BuildSetMigrating(true)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

## Delegation

[Go source](../delegation_generated.go), [ABI](../abi/Delegation.json).

```go
contract, err := offline.Delegation(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildApplyPenaltiesToStakes`

`applyPenaltiesToStakes(bytes32[])`

```go
call, err := contract.BuildApplyPenaltiesToStakes([][32]byte{[32]byte{1}})
if err != nil { panic(err) }
_ = call
```

### `BuildApplyPenaltiesToValidator`

`applyPenaltiesToValidator((address,address,uint256)[])`

```go
call, err := contract.BuildApplyPenaltiesToValidator([]decimal.DelegationApplyPenaltiesToValidatorArg0{decimal.DelegationApplyPenaltiesToValidatorArg0{Validator: fixtureAddress, Token: fixtureAddress, TokenId: big.NewInt(1)}})
if err != nil { panic(err) }
_ = call
```

### `BuildAutoUnbondEnqueue`

`autoUnbondEnqueue(address,address,uint256,address,uint256)`

```go
call, err := contract.BuildAutoUnbondEnqueue(fixtureAddress, fixtureAddress, big.NewInt(1), fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildChangeStake`

`changeStake(address[],address[],uint256[],address[])`

```go
call, err := contract.BuildChangeStake([]common.Address{fixtureAddress}, []common.Address{fixtureAddress}, []*big.Int{big.NewInt(1)}, []common.Address{fixtureAddress})
if err != nil { panic(err) }
_ = call
```

### `BuildComplete`

`complete(uint256[])`

```go
call, err := contract.BuildComplete([]*big.Int{big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegate`

`delegate(address,address,uint256)`

```go
call, err := contract.BuildDelegate(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateByPermit`

`delegateByPermit(address,address,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildDelegateByPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateDEL`

`delegateDEL(address)`

```go
call, err := contract.BuildDelegateDEL(big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateDELTo`

`delegateDELTo(address,address)`

```go
call, err := contract.BuildDelegateDELTo(big.NewInt(1), fixtureAddress, fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHold`

`delegateHold(address,address,uint256,uint256)`

```go
call, err := contract.BuildDelegateHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHoldByPermit`

`delegateHoldByPermit(address,address,uint256,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildDelegateHoldByPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHoldDEL`

`delegateHoldDEL(address,uint256)`

```go
call, err := contract.BuildDelegateHoldDEL(big.NewInt(1), fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateTo`

`delegateTo(address,address,address,uint256)`

```go
call, err := contract.BuildDelegateTo(fixtureAddress, fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildForcedWithdrawal`

`forcedWithdrawal(address,address,uint256,address)`

```go
call, err := contract.BuildForcedWithdrawal(fixtureAddress, fixtureAddress, big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildForcedWithdrawalBatch`

`forcedWithdrawalBatch(address[],address[],uint256[],address[])`

```go
call, err := contract.BuildForcedWithdrawalBatch([]common.Address{fixtureAddress}, []common.Address{fixtureAddress}, []*big.Int{big.NewInt(1)}, []common.Address{fixtureAddress})
if err != nil { panic(err) }
_ = call
```

### `BuildHold`

`hold(address,address,uint256,uint256,uint256)`

```go
call, err := contract.BuildHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildHoldWithReset`

`holdWithReset(address,address,uint256,uint256,uint256[])`

```go
call, err := contract.BuildHoldWithReset(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), []*big.Int{big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildMigrateStakes`

`migrateStakes(address[],address[],address[],address[])`

```go
call, err := contract.BuildMigrateStakes([]common.Address{fixtureAddress}, []common.Address{fixtureAddress}, []common.Address{fixtureAddress}, []common.Address{fixtureAddress})
if err != nil { panic(err) }
_ = call
```

### `BuildPause`

`pause()`

```go
call, err := contract.BuildPause()
if err != nil { panic(err) }
_ = call
```

### `BuildProcessAutoUnbond`

`processAutoUnbond(uint256)`

```go
call, err := contract.BuildProcessAutoUnbond(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildResetHold`

`resetHold(address,address,address,uint256)`

```go
call, err := contract.BuildResetHold(fixtureAddress, fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildResetHolds`

`resetHolds(address,address,address,uint256[])`

```go
call, err := contract.BuildResetHolds(fixtureAddress, fixtureAddress, fixtureAddress, []*big.Int{big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractCenter`

`setContractCenter(address)`

```go
call, err := contract.BuildSetContractCenter(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetFreezeTime`

`setFreezeTime(uint256,uint256)`

```go
call, err := contract.BuildSetFreezeTime(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransfer`

`transfer(address,address,uint256,address)`

```go
call, err := contract.BuildTransfer(fixtureAddress, fixtureAddress, big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferHold`

`transferHold(address,address,uint256,uint256,address)`

```go
call, err := contract.BuildTransferHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferWithReset`

`transferWithReset(address,address,uint256,address,uint256[])`

```go
call, err := contract.BuildTransferWithReset(fixtureAddress, fixtureAddress, big.NewInt(1), fixtureAddress, []*big.Int{big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildUnpause`

`unpause()`

```go
call, err := contract.BuildUnpause()
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildWithdraw`

`withdraw(address,address,uint256)`

```go
call, err := contract.BuildWithdraw(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildWithdrawHold`

`withdrawHold(address,address,uint256,uint256)`

```go
call, err := contract.BuildWithdrawHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildWithdrawWithReset`

`withdrawWithReset(address,address,uint256,uint256[])`

```go
call, err := contract.BuildWithdrawWithReset(fixtureAddress, fixtureAddress, big.NewInt(1), []*big.Int{big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

## DelegationNFT

[Go source](../delegationnft_generated.go), [ABI](../abi/DelegationNFT.json).

```go
contract, err := offline.DelegationNFT(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildApplyPenaltiesToValidator`

`applyPenaltiesToValidator((address,address,uint256)[])`

```go
call, err := contract.BuildApplyPenaltiesToValidator([]decimal.DelegationNFTApplyPenaltiesToValidatorArg0{decimal.DelegationNFTApplyPenaltiesToValidatorArg0{Validator: fixtureAddress, Token: fixtureAddress, TokenId: big.NewInt(1)}})
if err != nil { panic(err) }
_ = call
```

### `BuildComplete`

`complete(uint256[])`

```go
call, err := contract.BuildComplete([]*big.Int{big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateDRC1155`

`delegateDRC1155(address,address,uint256,uint256)`

```go
call, err := contract.BuildDelegateDRC1155(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateDRC1155ByPermit`

`delegateDRC1155ByPermit(address,address,uint256,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildDelegateDRC1155ByPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateDRC721`

`delegateDRC721(address,address,uint256)`

```go
call, err := contract.BuildDelegateDRC721(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateDRC721ByPermit`

`delegateDRC721ByPermit(address,address,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildDelegateDRC721ByPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHoldDRC1155`

`delegateHoldDRC1155(address,address,uint256,uint256,uint256)`

```go
call, err := contract.BuildDelegateHoldDRC1155(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHoldDRC1155ByPermit`

`delegateHoldDRC1155ByPermit(address,address,uint256,uint256,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildDelegateHoldDRC1155ByPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHoldDRC721`

`delegateHoldDRC721(address,address,uint256,uint256)`

```go
call, err := contract.BuildDelegateHoldDRC721(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDelegateHoldDRC721ByPermit`

`delegateHoldDRC721ByPermit(address,address,uint256,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildDelegateHoldDRC721ByPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildHold`

`hold(address,address,uint256,uint256,uint256,uint256)`

```go
call, err := contract.BuildHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildOnERC1155BatchReceived`

`onERC1155BatchReceived(address,address,uint256[],uint256[],bytes)`

```go
call, err := contract.BuildOnERC1155BatchReceived(fixtureAddress, fixtureAddress, []*big.Int{big.NewInt(1)}, []*big.Int{big.NewInt(1)}, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildOnERC1155Received`

`onERC1155Received(address,address,uint256,uint256,bytes)`

```go
call, err := contract.BuildOnERC1155Received(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildOnERC721Received`

`onERC721Received(address,address,uint256,bytes)`

```go
call, err := contract.BuildOnERC721Received(fixtureAddress, fixtureAddress, big.NewInt(1), []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractCenter`

`setContractCenter(address)`

```go
call, err := contract.BuildSetContractCenter(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetFreezeTime`

`setFreezeTime(uint256,uint256)`

```go
call, err := contract.BuildSetFreezeTime(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransfer`

`transfer(address,address,uint256,uint256,address)`

```go
call, err := contract.BuildTransfer(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferHold`

`transferHold(address,address,uint256,uint256,uint256,address)`

```go
call, err := contract.BuildTransferHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildWithdraw`

`withdraw(address,address,uint256,uint256)`

```go
call, err := contract.BuildWithdraw(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildWithdrawHold`

`withdrawHold(address,address,uint256,uint256,uint256)`

```go
call, err := contract.BuildWithdrawHold(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

## ERC1155

[Go source](../erc1155_generated.go), [ABI](../abi/ERC1155.json).

```go
contract, err := offline.ERC1155(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildAddReserve`

`addReserve(uint256,uint256)`

```go
call, err := contract.BuildAddReserve(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildAddReserveByETH`

`addReserveByETH(uint256)`

```go
call, err := contract.BuildAddReserveByETH(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildAddReserveByPermit`

`addReserveByPermit(uint256,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildAddReserveByPermit(big.NewInt(1), big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildBurn`

`burn(uint256,uint256)`

```go
call, err := contract.BuildBurn(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDisableMint`

`disableMint()`

```go
call, err := contract.BuildDisableMint()
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize(string,string,string,address,bool)`

```go
call, err := contract.BuildInitialize("fixture", "fixture", "fixture", fixtureAddress, true)
if err != nil { panic(err) }
_ = call
```

### `BuildMint`

`mint(address,uint256,uint256,string,uint256,address)`

```go
call, err := contract.BuildMint(fixtureAddress, big.NewInt(1), big.NewInt(1), "fixture", big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildMintByETH`

`mintByETH(address,uint256,uint256,string)`

```go
call, err := contract.BuildMintByETH(big.NewInt(1), fixtureAddress, big.NewInt(1), big.NewInt(1), "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildMintByPermit`

`mintByPermit(address,uint256,uint256,string,uint256,address,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildMintByPermit(fixtureAddress, big.NewInt(1), big.NewInt(1), "fixture", big.NewInt(1), fixtureAddress, big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildPenalty`

`penalty(uint256,uint256,uint16)`

```go
call, err := contract.BuildPenalty(big.NewInt(1), big.NewInt(1), uint16(1))
if err != nil { panic(err) }
_ = call
```

### `BuildPermit`

`permit(address,address,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildPermit(fixtureAddress, fixtureAddress, big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSafeBatchTransferFrom`

`safeBatchTransferFrom(address,address,uint256[],uint256[],bytes)`

```go
call, err := contract.BuildSafeBatchTransferFrom(fixtureAddress, fixtureAddress, []*big.Int{big.NewInt(1)}, []*big.Int{big.NewInt(1)}, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSafeTransferFrom`

`safeTransferFrom(address,address,uint256,uint256,bytes)`

```go
call, err := contract.BuildSafeTransferFrom(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSetApprovalForAll`

`setApprovalForAll(address,bool)`

```go
call, err := contract.BuildSetApprovalForAll(fixtureAddress, true)
if err != nil { panic(err) }
_ = call
```

## ERC721

[Go source](../erc721_generated.go), [ABI](../abi/ERC721.json).

```go
contract, err := offline.ERC721(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildAddReserve`

`addReserve(uint256,uint256)`

```go
call, err := contract.BuildAddReserve(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildAddReserveByETH`

`addReserveByETH(uint256)`

```go
call, err := contract.BuildAddReserveByETH(big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildAddReserveByPermit`

`addReserveByPermit(uint256,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildAddReserveByPermit(big.NewInt(1), big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildApprove`

`approve(address,uint256)`

```go
call, err := contract.BuildApprove(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBurn`

`burn(uint256)`

```go
call, err := contract.BuildBurn(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDisableMint`

`disableMint()`

```go
call, err := contract.BuildDisableMint()
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize(string,string,string,address,bool)`

```go
call, err := contract.BuildInitialize("fixture", "fixture", "fixture", fixtureAddress, true)
if err != nil { panic(err) }
_ = call
```

### `BuildMint`

`mint(address,string,uint256,address)`

```go
call, err := contract.BuildMint(fixtureAddress, "fixture", big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildMintByETH`

`mintByETH(address,string)`

```go
call, err := contract.BuildMintByETH(big.NewInt(1), fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildMintByPermit`

`mintByPermit(address,string,uint256,address,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildMintByPermit(fixtureAddress, "fixture", big.NewInt(1), fixtureAddress, big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildPenalty`

`penalty(uint256,uint16)`

```go
call, err := contract.BuildPenalty(big.NewInt(1), uint16(1))
if err != nil { panic(err) }
_ = call
```

### `BuildPermit`

`permit(address,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildPermit(fixtureAddress, big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSafeTransferFrom`

`safeTransferFrom(address,address,uint256)`

```go
call, err := contract.BuildSafeTransferFrom(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildSafeTransferFrom1`

`safeTransferFrom(address,address,uint256,bytes)`

```go
call, err := contract.BuildSafeTransferFrom1(fixtureAddress, fixtureAddress, big.NewInt(1), []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSetApprovalForAll`

`setApprovalForAll(address,bool)`

```go
call, err := contract.BuildSetApprovalForAll(fixtureAddress, true)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferFrom`

`transferFrom(address,address,uint256)`

```go
call, err := contract.BuildTransferFrom(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

## MasterValidator

[Go source](../mastervalidator_generated.go), [ABI](../abi/MasterValidator.json).

```go
contract, err := offline.MasterValidator(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildAddCandidate`

`addCandidate(address,string,(address,uint256))`

```go
call, err := contract.BuildAddCandidate(fixtureAddress, "fixture", decimal.MasterValidatorAddCandidateArg2{Token: fixtureAddress, Amount: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildAddCandidateDEL`

`addCandidateDEL(address,string)`

```go
call, err := contract.BuildAddCandidateDEL(big.NewInt(1), fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildAddPenalty`

`addPenalty(address,uint16)`

```go
call, err := contract.BuildAddPenalty(fixtureAddress, uint16(1))
if err != nil { panic(err) }
_ = call
```

### `BuildAddValidator`

`addValidator(address,string,(address,uint256))`

```go
call, err := contract.BuildAddValidator(fixtureAddress, "fixture", decimal.MasterValidatorAddValidatorArg2{Token: fixtureAddress, Amount: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildAddValidatorDEL`

`addValidatorDEL(address,string)`

```go
call, err := contract.BuildAddValidatorDEL(big.NewInt(1), fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildAddValidators`

`addValidators((address,string)[])`

```go
call, err := contract.BuildAddValidators([]decimal.MasterValidatorAddValidatorsArg0{decimal.MasterValidatorAddValidatorsArg0{Validator: fixtureAddress, Meta: "fixture"}})
if err != nil { panic(err) }
_ = call
```

### `BuildApproveCandidates`

`approveCandidates(address[])`

```go
call, err := contract.BuildApproveCandidates([]common.Address{fixtureAddress})
if err != nil { panic(err) }
_ = call
```

### `BuildClearRewardPerBlock`

`clearRewardPerBlock()`

```go
call, err := contract.BuildClearRewardPerBlock()
if err != nil { panic(err) }
_ = call
```

### `BuildPauseSelf`

`pauseSelf()`

```go
call, err := contract.BuildPauseSelf()
if err != nil { panic(err) }
_ = call
```

### `BuildPauseValidator`

`pauseValidator(address)`

```go
call, err := contract.BuildPauseValidator(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildRemoveValidator`

`removeValidator(address)`

```go
call, err := contract.BuildRemoveValidator(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetRewardPerBlock`

`setRewardPerBlock(uint256)`

```go
call, err := contract.BuildSetRewardPerBlock(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUnpauseSelf`

`unpauseSelf()`

```go
call, err := contract.BuildUnpauseSelf()
if err != nil { panic(err) }
_ = call
```

### `BuildUnpauseValidator`

`unpauseValidator(address)`

```go
call, err := contract.BuildUnpauseValidator(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpdateValidatorMeta`

`updateValidatorMeta(address,string)`

```go
call, err := contract.BuildUpdateValidatorMeta(fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

## Multicall

[Go source](../multicall_generated.go), [ABI](../abi/Multicall.json).

```go
contract, err := offline.Multicall(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildAggregate`

`aggregate((address,uint256,bytes)[])`

```go
call, err := contract.BuildAggregate(big.NewInt(1), []decimal.MulticallAggregateArg0{decimal.MulticallAggregateArg0{Target: fixtureAddress, Value: big.NewInt(1), CallData: []byte{1}}})
if err != nil { panic(err) }
_ = call
```

## NFTCenter

[Go source](../nftcenter_generated.go), [ABI](../abi/NFTCenter.json).

```go
contract, err := offline.NFTCenter(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildCreateERC1155`

`createERC1155((address,string,string,string,bool))`

```go
call, err := contract.BuildCreateERC1155(decimal.NFTCenterCreateERC1155Arg0{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateERC721`

`createERC721((address,string,string,string,bool))`

```go
call, err := contract.BuildCreateERC721(decimal.NFTCenterCreateERC721Arg0{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true})
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize()`

```go
call, err := contract.BuildInitialize()
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractCenter`

`setContractCenter(address)`

```go
call, err := contract.BuildSetContractCenter(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildUpgradeNFT`

`upgradeNFT(address,uint8)`

```go
call, err := contract.BuildUpgradeNFT(fixtureAddress, uint8(1))
if err != nil { panic(err) }
_ = call
```

## TestnetContractCenter20261029

[Go source](../testnetcontractcenter20261029_generated.go), [ABI](../abi/TestnetContractCenter20261029.json).

```go
contract, err := offline.TestnetContractCenter20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildMarkReserve`

`markReserve((address,uint256))`

```go
call, err := contract.BuildMarkReserve(decimal.TestnetContractCenter20261029MarkReserveArg0{ContractAddress: fixtureAddress, Amount: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildMarkReserves`

`markReserves((address,uint256)[])`

```go
call, err := contract.BuildMarkReserves([]decimal.TestnetContractCenter20261029MarkReservesArg0{decimal.TestnetContractCenter20261029MarkReservesArg0{ContractAddress: fixtureAddress, Amount: big.NewInt(1)}})
if err != nil { panic(err) }
_ = call
```

### `BuildMigrateReserves`

`migrateReserves()`

```go
call, err := contract.BuildMigrateReserves(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildMigrateReservesPage`

`migrateReservesPage(uint256,uint256)`

```go
call, err := contract.BuildMigrateReservesPage(big.NewInt(1), big.NewInt(1), big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildMigrateReservesPageWithSkipping`

`migrateReservesPageWithSkipping(uint256,uint256,address[])`

```go
call, err := contract.BuildMigrateReservesPageWithSkipping(big.NewInt(1), big.NewInt(1), big.NewInt(1), []common.Address{fixtureAddress})
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetAddress`

`setAddress(string,address)`

```go
call, err := contract.BuildSetAddress("fixture", fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractAddress`

`setContractAddress(string,address)`

```go
call, err := contract.BuildSetContractAddress("fixture", fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetMigrating`

`setMigrating(bool)`

```go
call, err := contract.BuildSetMigrating(true)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

## TestnetMasterValidator20261029

[Go source](../testnetmastervalidator20261029_generated.go), [ABI](../abi/TestnetMasterValidator20261029.json).

```go
contract, err := offline.TestnetMasterValidator20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildAddCandidate`

`addCandidate(address,string,(address,uint256))`

```go
call, err := contract.BuildAddCandidate(fixtureAddress, "fixture", decimal.TestnetMasterValidator20261029AddCandidateArg2{Token: fixtureAddress, Amount: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildAddCandidateDEL`

`addCandidateDEL(address,string)`

```go
call, err := contract.BuildAddCandidateDEL(big.NewInt(1), fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildAddPenalty`

`addPenalty(address,uint16)`

```go
call, err := contract.BuildAddPenalty(fixtureAddress, uint16(1))
if err != nil { panic(err) }
_ = call
```

### `BuildAddValidator`

`addValidator(address,string,(address,uint256))`

```go
call, err := contract.BuildAddValidator(fixtureAddress, "fixture", decimal.TestnetMasterValidator20261029AddValidatorArg2{Token: fixtureAddress, Amount: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildAddValidatorDEL`

`addValidatorDEL(address,string)`

```go
call, err := contract.BuildAddValidatorDEL(big.NewInt(1), fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildAddValidators`

`addValidators((address,string)[])`

```go
call, err := contract.BuildAddValidators([]decimal.TestnetMasterValidator20261029AddValidatorsArg0{decimal.TestnetMasterValidator20261029AddValidatorsArg0{Validator: fixtureAddress, Meta: "fixture"}})
if err != nil { panic(err) }
_ = call
```

### `BuildApproveCandidates`

`approveCandidates(address[])`

```go
call, err := contract.BuildApproveCandidates([]common.Address{fixtureAddress})
if err != nil { panic(err) }
_ = call
```

### `BuildClearRewardPerBlock`

`clearRewardPerBlock()`

```go
call, err := contract.BuildClearRewardPerBlock()
if err != nil { panic(err) }
_ = call
```

### `BuildPause`

`pause()`

```go
call, err := contract.BuildPause()
if err != nil { panic(err) }
_ = call
```

### `BuildPauseSelf`

`pauseSelf()`

```go
call, err := contract.BuildPauseSelf()
if err != nil { panic(err) }
_ = call
```

### `BuildPauseValidator`

`pauseValidator(address)`

```go
call, err := contract.BuildPauseValidator(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildRemoveValidator`

`removeValidator(address)`

```go
call, err := contract.BuildRemoveValidator(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetRewardPerBlock`

`setRewardPerBlock(uint256)`

```go
call, err := contract.BuildSetRewardPerBlock(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUnpause`

`unpause()`

```go
call, err := contract.BuildUnpause()
if err != nil { panic(err) }
_ = call
```

### `BuildUnpauseSelf`

`unpauseSelf()`

```go
call, err := contract.BuildUnpauseSelf()
if err != nil { panic(err) }
_ = call
```

### `BuildUnpauseValidator`

`unpauseValidator(address)`

```go
call, err := contract.BuildUnpauseValidator(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpdateValidatorMeta`

`updateValidatorMeta(address,string)`

```go
call, err := contract.BuildUpdateValidatorMeta(fixtureAddress, "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

## TestnetNFTCenter20261029

[Go source](../testnetnftcenter20261029_generated.go), [ABI](../abi/TestnetNFTCenter20261029.json).

```go
contract, err := offline.TestnetNFTCenter20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildCreateAndMigrateBatch`

`createAndMigrateBatch(((address,string,string,string,bool,bool),(string,uint256,string,bool),bytes,uint8,(address,uint256)[])[])`

```go
call, err := contract.BuildCreateAndMigrateBatch([]decimal.TestnetNFTCenter20261029CreateAndMigrateBatchArg0{decimal.TestnetNFTCenter20261029CreateAndMigrateBatchArg0{Meta: decimal.TestnetNFTCenter20261029CreateAndMigrateBatchArg0Meta{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true}, MigrationMeta: decimal.TestnetNFTCenter20261029CreateAndMigrateBatchArg0MigrationMeta{ReserveSymbol: "fixture", ReserveAmount: big.NewInt(1), TokenURI: "fixture", AllowMint: true}, Salt: []byte{1}, NftType: uint8(1), Balances: []decimal.TestnetNFTCenter20261029CreateAndMigrateBatchArg0Balances{decimal.TestnetNFTCenter20261029CreateAndMigrateBatchArg0Balances{Account: fixtureAddress, Balance: big.NewInt(1)}}}})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateCollectionDRC1155Migration`

`createCollectionDRC1155Migration(((address,string,string,string,bool,bool),(string,uint256,string,bool),bytes,uint8,(address,uint256)[]))`

```go
call, err := contract.BuildCreateCollectionDRC1155Migration(decimal.TestnetNFTCenter20261029CreateCollectionDRC1155MigrationArg0{Meta: decimal.TestnetNFTCenter20261029CreateCollectionDRC1155MigrationArg0Meta{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true}, MigrationMeta: decimal.TestnetNFTCenter20261029CreateCollectionDRC1155MigrationArg0MigrationMeta{ReserveSymbol: "fixture", ReserveAmount: big.NewInt(1), TokenURI: "fixture", AllowMint: true}, Salt: []byte{1}, NftType: uint8(1), Balances: []decimal.TestnetNFTCenter20261029CreateCollectionDRC1155MigrationArg0Balances{decimal.TestnetNFTCenter20261029CreateCollectionDRC1155MigrationArg0Balances{Account: fixtureAddress, Balance: big.NewInt(1)}}})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateCollectionDRC721Migration`

`createCollectionDRC721Migration(((address,string,string,string,bool,bool),(string,uint256,string,bool),bytes,uint8,(address,uint256)[]))`

```go
call, err := contract.BuildCreateCollectionDRC721Migration(decimal.TestnetNFTCenter20261029CreateCollectionDRC721MigrationArg0{Meta: decimal.TestnetNFTCenter20261029CreateCollectionDRC721MigrationArg0Meta{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true}, MigrationMeta: decimal.TestnetNFTCenter20261029CreateCollectionDRC721MigrationArg0MigrationMeta{ReserveSymbol: "fixture", ReserveAmount: big.NewInt(1), TokenURI: "fixture", AllowMint: true}, Salt: []byte{1}, NftType: uint8(1), Balances: []decimal.TestnetNFTCenter20261029CreateCollectionDRC721MigrationArg0Balances{decimal.TestnetNFTCenter20261029CreateCollectionDRC721MigrationArg0Balances{Account: fixtureAddress, Balance: big.NewInt(1)}}})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateDRC1155`

`createDRC1155((address,string,string,string,bool,bool))`

```go
call, err := contract.BuildCreateDRC1155(decimal.TestnetNFTCenter20261029CreateDRC1155Arg0{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateDRC1155Reserveless`

`createDRC1155Reserveless((address,string,string,string,bool,bool))`

```go
call, err := contract.BuildCreateDRC1155Reserveless(decimal.TestnetNFTCenter20261029CreateDRC1155ReservelessArg0{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateDRC721`

`createDRC721((address,string,string,string,bool,bool))`

```go
call, err := contract.BuildCreateDRC721(decimal.TestnetNFTCenter20261029CreateDRC721Arg0{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateDRC721Reserveless`

`createDRC721Reserveless((address,string,string,string,bool,bool))`

```go
call, err := contract.BuildCreateDRC721Reserveless(decimal.TestnetNFTCenter20261029CreateDRC721ReservelessArg0{Creator: fixtureAddress, Symbol: "fixture", Name: "fixture", ContractURI: "fixture", Refundable: true, Burnable: true})
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize()`

```go
call, err := contract.BuildInitialize()
if err != nil { panic(err) }
_ = call
```

### `BuildPause`

`pause()`

```go
call, err := contract.BuildPause()
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractCenter`

`setContractCenter(address)`

```go
call, err := contract.BuildSetContractCenter(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUnpause`

`unpause()`

```go
call, err := contract.BuildUnpause()
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildUpgradeNFT`

`upgradeNFT(address,uint8,bool)`

```go
call, err := contract.BuildUpgradeNFT(fixtureAddress, uint8(1), true)
if err != nil { panic(err) }
_ = call
```

## TestnetReserveToken20261029

[Go source](../testnetreservetoken20261029_generated.go), [ABI](../abi/TestnetReserveToken20261029.json).

```go
contract, err := offline.TestnetReserveToken20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildApprove`

`approve(address,uint256)`

```go
call, err := contract.BuildApprove(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBurn`

`burn(uint256)`

```go
call, err := contract.BuildBurn(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBuy`

`buy(uint256,address)`

```go
call, err := contract.BuildBuy(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildBuyExactTokenForDEL`

`buyExactTokenForDEL(uint256,address)`

```go
call, err := contract.BuildBuyExactTokenForDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildBuyTokenForExactDEL`

`buyTokenForExactDEL(uint256,address)`

```go
call, err := contract.BuildBuyTokenForExactDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize(string,string,address,uint8,uint256,uint256,uint256,string)`

```go
call, err := contract.BuildInitialize(big.NewInt(1), "fixture", "fixture", fixtureAddress, uint8(1), big.NewInt(1), big.NewInt(1), big.NewInt(1), "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildPermit`

`permit(address,address,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSell`

`sell(uint256,uint256,address)`

```go
call, err := contract.BuildSell(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSellExactTokensForDEL`

`sellExactTokensForDEL(uint256,uint256,address)`

```go
call, err := contract.BuildSellExactTokensForDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSellTokensForExactDEL`

`sellTokensForExactDEL(uint256,uint256,address)`

```go
call, err := contract.BuildSellTokensForExactDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransfer`

`transfer(address,uint256)`

```go
call, err := contract.BuildTransfer(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransferFrom`

`transferFrom(address,address,uint256)`

```go
call, err := contract.BuildTransferFrom(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildUpdateIdentity`

`updateIdentity(string)`

```go
call, err := contract.BuildUpdateIdentity("fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildUpdateMaxTotalSupply`

`updateMaxTotalSupply(uint256)`

```go
call, err := contract.BuildUpdateMaxTotalSupply(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildUpdateMinTotalSupply`

`updateMinTotalSupply(uint256)`

```go
call, err := contract.BuildUpdateMinTotalSupply(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

## TestnetReservelessToken20261029

[Go source](../testnetreservelesstoken20261029_generated.go), [ABI](../abi/TestnetReservelessToken20261029.json).

```go
contract, err := offline.TestnetReservelessToken20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildApprove`

`approve(address,uint256)`

```go
call, err := contract.BuildApprove(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBurn`

`burn(uint256)`

```go
call, err := contract.BuildBurn(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBurnFrom`

`burnFrom(address,uint256)`

```go
call, err := contract.BuildBurnFrom(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize(string,string,address,bool,bool,uint256,uint256,string)`

```go
call, err := contract.BuildInitialize(big.NewInt(1), "fixture", "fixture", fixtureAddress, true, true, big.NewInt(1), big.NewInt(1), "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildMint`

`mint(address,uint256)`

```go
call, err := contract.BuildMint(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildPermit`

`permit(address,address,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildTransfer`

`transfer(address,uint256)`

```go
call, err := contract.BuildTransfer(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransferFrom`

`transferFrom(address,address,uint256)`

```go
call, err := contract.BuildTransferFrom(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

## TestnetSafe20261029

[Go source](../testnetsafe20261029_generated.go), [ABI](../abi/TestnetSafe20261029.json).

```go
contract, err := offline.TestnetSafe20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildAddOwnerWithThreshold`

`addOwnerWithThreshold((address,uint256),uint256)`

```go
call, err := contract.BuildAddOwnerWithThreshold(decimal.TestnetSafe20261029AddOwnerWithThresholdArg0{Owner: fixtureAddress, Weight: big.NewInt(1)}, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildApproveHash`

`approveHash((address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,uint256))`

```go
call, err := contract.BuildApproveHash(decimal.TestnetSafe20261029ApproveHashArg0{To: fixtureAddress, Value: big.NewInt(1), Data: []byte{1}, Operation: uint8(1), SafeTxGas: big.NewInt(1), BaseGas: big.NewInt(1), GasPrice: big.NewInt(1), GasToken: fixtureAddress, RefundReceiver: fixtureAddress, Nonce: big.NewInt(1)})
if err != nil { panic(err) }
_ = call
```

### `BuildChangeWeightThreshold`

`changeWeightThreshold(uint256)`

```go
call, err := contract.BuildChangeWeightThreshold(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildDisableModule`

`disableModule(address,address)`

```go
call, err := contract.BuildDisableModule(fixtureAddress, fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildEnableModule`

`enableModule(address)`

```go
call, err := contract.BuildEnableModule(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildExecTransaction`

`execTransaction(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,bytes)`

```go
call, err := contract.BuildExecTransaction(big.NewInt(1), fixtureAddress, big.NewInt(1), []byte{1}, uint8(1), big.NewInt(1), big.NewInt(1), big.NewInt(1), fixtureAddress, fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildExecTransactionFromModule`

`execTransactionFromModule(address,uint256,bytes,uint8)`

```go
call, err := contract.BuildExecTransactionFromModule(fixtureAddress, big.NewInt(1), []byte{1}, uint8(1))
if err != nil { panic(err) }
_ = call
```

### `BuildExecTransactionFromModuleReturnData`

`execTransactionFromModuleReturnData(address,uint256,bytes,uint8)`

```go
call, err := contract.BuildExecTransactionFromModuleReturnData(fixtureAddress, big.NewInt(1), []byte{1}, uint8(1))
if err != nil { panic(err) }
_ = call
```

### `BuildRemoveOwner`

`removeOwner(address,address,uint256)`

```go
call, err := contract.BuildRemoveOwner(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildSetFallbackHandler`

`setFallbackHandler(address)`

```go
call, err := contract.BuildSetFallbackHandler(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetGuard`

`setGuard(address)`

```go
call, err := contract.BuildSetGuard(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetModuleGuard`

`setModuleGuard(address)`

```go
call, err := contract.BuildSetModuleGuard(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSetup`

`setup((address,uint256)[],uint256,address,bytes,address,address,uint256,address)`

```go
call, err := contract.BuildSetup([]decimal.TestnetSafe20261029SetupArg0{decimal.TestnetSafe20261029SetupArg0{Owner: fixtureAddress, Weight: big.NewInt(1)}}, big.NewInt(1), fixtureAddress, []byte{1}, fixtureAddress, fixtureAddress, big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSimulateAndRevert`

`simulateAndRevert(address,bytes)`

```go
call, err := contract.BuildSimulateAndRevert(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSwapOwner`

`swapOwner(address,address,address)`

```go
call, err := contract.BuildSwapOwner(fixtureAddress, fixtureAddress, fixtureAddress)
if err != nil { panic(err) }
_ = call
```

## TestnetSafeFactory20261029

[Go source](../testnetsafefactory20261029_generated.go), [ABI](../abi/TestnetSafeFactory20261029.json).

```go
contract, err := offline.TestnetSafeFactory20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildCreateChainSpecificProxyWithNonce`

`createChainSpecificProxyWithNonce(address,bytes,uint256)`

```go
call, err := contract.BuildCreateChainSpecificProxyWithNonce(fixtureAddress, []byte{1}, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildCreateProxyWithCallback`

`createProxyWithCallback(address,bytes,uint256,address)`

```go
call, err := contract.BuildCreateProxyWithCallback(fixtureAddress, []byte{1}, big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildCreateProxyWithNonce`

`createProxyWithNonce(address,bytes,uint256)`

```go
call, err := contract.BuildCreateProxyWithNonce(fixtureAddress, []byte{1}, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

## TestnetSafeMultiSend20261029

[Go source](../testnetsafemultisend20261029_generated.go), [ABI](../abi/TestnetSafeMultiSend20261029.json).

```go
contract, err := offline.TestnetSafeMultiSend20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildMultiSend`

`multiSend(bytes)`

```go
call, err := contract.BuildMultiSend(big.NewInt(1), []byte{1})
if err != nil { panic(err) }
_ = call
```

## TestnetTokenCenter20261029

[Go source](../testnettokencenter20261029_generated.go), [ABI](../abi/TestnetTokenCenter20261029.json).

```go
contract, err := offline.TestnetTokenCenter20261029(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildConvert`

`convert(address,address,uint256,uint256,address)`

```go
call, err := contract.BuildConvert(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildConvert1`

`convert(address,address,uint256,uint256,address,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildConvert1(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), fixtureAddress, big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateToken`

`createToken((uint256,uint256,uint256,address,uint8,string,string,string))`

```go
call, err := contract.BuildCreateToken(big.NewInt(1), decimal.TestnetTokenCenter20261029CreateTokenArg0{InitialMint: big.NewInt(1), MinTotalSupply: big.NewInt(1), MaxTotalSupply: big.NewInt(1), Creator: fixtureAddress, Crr: uint8(1), Identity: "fixture", Symbol: "fixture", Name: "fixture"})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateTokenReserveless`

`createTokenReserveless(string,string,bool,bool,uint256,uint256,string)`

```go
call, err := contract.BuildCreateTokenReserveless("fixture", "fixture", true, true, big.NewInt(1), big.NewInt(1), "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize()`

```go
call, err := contract.BuildInitialize()
if err != nil { panic(err) }
_ = call
```

### `BuildPause`

`pause()`

```go
call, err := contract.BuildPause()
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractCenter`

`setContractCenter(address)`

```go
call, err := contract.BuildSetContractCenter(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUnpause`

`unpause()`

```go
call, err := contract.BuildUnpause()
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildUpgradeToken`

`upgradeToken(address)`

```go
call, err := contract.BuildUpgradeToken(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgradeTokenReserveless`

`upgradeTokenReserveless(address)`

```go
call, err := contract.BuildUpgradeTokenReserveless(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

## Token

[Go source](../token_generated.go), [ABI](../abi/Token.json).

```go
contract, err := offline.Token(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildApprove`

`approve(address,uint256)`

```go
call, err := contract.BuildApprove(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBurn`

`burn(uint256)`

```go
call, err := contract.BuildBurn(big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildBuy`

`buy(uint256,address)`

```go
call, err := contract.BuildBuy(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildBuyExactTokenForDEL`

`buyExactTokenForDEL(uint256,address)`

```go
call, err := contract.BuildBuyExactTokenForDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildBuyTokenForExactDEL`

`buyTokenForExactDEL(uint256,address)`

```go
call, err := contract.BuildBuyTokenForExactDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize(string,string,address,uint8,uint256,uint256,uint256,string)`

```go
call, err := contract.BuildInitialize(big.NewInt(1), "fixture", "fixture", fixtureAddress, uint8(1), big.NewInt(1), big.NewInt(1), big.NewInt(1), "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildPermit`

`permit(address,address,uint256,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildPermit(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildSell`

`sell(uint256,uint256,address)`

```go
call, err := contract.BuildSell(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSellExactTokensForDEL`

`sellExactTokensForDEL(uint256,uint256,address)`

```go
call, err := contract.BuildSellExactTokensForDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildSellTokensForExactDEL`

`sellTokensForExactDEL(uint256,uint256,address)`

```go
call, err := contract.BuildSellTokensForExactDEL(big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransfer`

`transfer(address,uint256)`

```go
call, err := contract.BuildTransfer(fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildTransferFrom`

`transferFrom(address,address,uint256)`

```go
call, err := contract.BuildTransferFrom(fixtureAddress, fixtureAddress, big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

### `BuildUpdateDetails`

`updateDetails(string,uint256)`

```go
call, err := contract.BuildUpdateDetails("fixture", big.NewInt(1))
if err != nil { panic(err) }
_ = call
```

## TokenCenter

[Go source](../tokencenter_generated.go), [ABI](../abi/TokenCenter.json).

```go
contract, err := offline.TokenCenter(fixtureAddress)
if err != nil { panic(err) }
```

### `BuildConvert`

`convert(address,address,uint256,uint256,address)`

```go
call, err := contract.BuildConvert(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildConvert1`

`convert(address,address,uint256,uint256,address,uint256,uint8,bytes32,bytes32)`

```go
call, err := contract.BuildConvert1(fixtureAddress, fixtureAddress, big.NewInt(1), big.NewInt(1), fixtureAddress, big.NewInt(1), uint8(1), [32]byte{1}, [32]byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateToken`

`createToken((uint256,uint256,uint256,address,uint8,string,string,string))`

```go
call, err := contract.BuildCreateToken(big.NewInt(1), decimal.TokenCenterCreateTokenArg0{InitialMint: big.NewInt(1), MinTotalSupply: big.NewInt(1), MaxTotalSupply: big.NewInt(1), Creator: fixtureAddress, Crr: uint8(1), Identity: "fixture", Symbol: "fixture", Name: "fixture"})
if err != nil { panic(err) }
_ = call
```

### `BuildCreateTokenReserveless`

`createTokenReserveless(string,string,bool,bool,uint256,uint256,string)`

```go
call, err := contract.BuildCreateTokenReserveless("fixture", "fixture", true, true, big.NewInt(1), big.NewInt(1), "fixture")
if err != nil { panic(err) }
_ = call
```

### `BuildInitialize`

`initialize()`

```go
call, err := contract.BuildInitialize()
if err != nil { panic(err) }
_ = call
```

### `BuildPause`

`pause()`

```go
call, err := contract.BuildPause()
if err != nil { panic(err) }
_ = call
```

### `BuildRenounceOwnership`

`renounceOwnership()`

```go
call, err := contract.BuildRenounceOwnership()
if err != nil { panic(err) }
_ = call
```

### `BuildSetContractCenter`

`setContractCenter(address)`

```go
call, err := contract.BuildSetContractCenter(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildTransferOwnership`

`transferOwnership(address)`

```go
call, err := contract.BuildTransferOwnership(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUnpause`

`unpause()`

```go
call, err := contract.BuildUnpause()
if err != nil { panic(err) }
_ = call
```

### `BuildUpgrade`

`upgrade(address,bytes)`

```go
call, err := contract.BuildUpgrade(fixtureAddress, []byte{1})
if err != nil { panic(err) }
_ = call
```

### `BuildUpgradeToken`

`upgradeToken(address)`

```go
call, err := contract.BuildUpgradeToken(fixtureAddress)
if err != nil { panic(err) }
_ = call
```

### `BuildUpgradeTokenReserveless`

`upgradeTokenReserveless(address)`

```go
call, err := contract.BuildUpgradeTokenReserveless(fixtureAddress)
if err != nil { panic(err) }
_ = call
```
