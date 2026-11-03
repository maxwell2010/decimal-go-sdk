// This example only builds calldata. It does not use a transport or signer.
package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"math/big"
)

func buildExamples() ([]decimal.Call, error) {
	client := &decimal.Client{}
	unified, err := client.UnifiedNFTDelegation(common.HexToAddress("0x1001"))
	if err != nil {
		return nil, err
	}
	validator := common.HexToAddress("0x1002")
	collection := common.HexToAddress("0x1003")
	newValidator := common.HexToAddress("0x1004")
	tokenID, amount := big.NewInt(42), big.NewInt(3)
	hold, deadline := big.NewInt(2000000000), big.NewInt(1900000000)
	// Encoding placeholders only, not a generated or usable permit proof.
	var r, s [32]byte
	r[31], s[31] = 1, 2
	builders := []func() (decimal.Call, error){
		func() (decimal.Call, error) { return unified.BuildDelegateNFT(validator, collection, tokenID) },
		func() (decimal.Call, error) {
			return unified.BuildDelegateNFTHold(validator, collection, tokenID, hold)
		},
		func() (decimal.Call, error) {
			return unified.BuildDelegateNFTByPermit(validator, collection, tokenID, deadline, 27, r, s)
		},
		func() (decimal.Call, error) {
			return unified.BuildDelegateNFT1155(validator, collection, tokenID, amount)
		},
		func() (decimal.Call, error) {
			return unified.BuildDelegateNFT1155Hold(validator, collection, tokenID, amount, hold)
		},
		func() (decimal.Call, error) { return unified.BuildWithdrawNFT(validator, collection, tokenID) },
		func() (decimal.Call, error) {
			return unified.BuildWithdrawNFTHold(validator, collection, tokenID, hold)
		},
		func() (decimal.Call, error) { return unified.BuildWithdrawNFT1155(validator, collection, tokenID) },
		func() (decimal.Call, error) {
			return unified.BuildWithdrawNFT1155Hold(validator, collection, tokenID, hold)
		},
		func() (decimal.Call, error) {
			return unified.BuildTransferNFT(validator, collection, tokenID, newValidator)
		},
		func() (decimal.Call, error) {
			return unified.BuildTransferNFTHold(validator, collection, tokenID, hold, newValidator)
		},
		func() (decimal.Call, error) {
			return unified.BuildTransferNFT1155(validator, collection, tokenID, newValidator)
		},
		func() (decimal.Call, error) {
			return unified.BuildTransferNFT1155Hold(validator, collection, tokenID, hold, newValidator)
		},
		func() (decimal.Call, error) { return unified.BuildERC721Approval(collection, tokenID) },
		// Collection-wide grant calldata, kept separate for review. Nothing is approved.
		func() (decimal.Call, error) { return unified.BuildERC1155ApprovalForAll(collection) },
	}
	calls := make([]decimal.Call, 0, len(builders))
	for _, build := range builders {
		call, err := build()
		if err != nil {
			return nil, err
		}
		calls = append(calls, call)
	}
	return calls, nil
}

func main() {
	calls, err := buildExamples()
	if err != nil {
		panic(err)
	}
	for _, call := range calls {
		fmt.Println(call.Signature, call.To.Hex(), len(call.Data))
	}
}
