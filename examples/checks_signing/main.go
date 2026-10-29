// Command checks_signing signs a public fixture and builds calldata offline.
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	decimal "github.com/maxwell2010/decimal-go-sdk"
)

// All inputs are public fixtures. This hash was not obtained from an issued check.
func buildExample() (decimal.Call, common.Hash, common.Address, error) {
	hash := common.HexToHash("0x" + strings.Repeat("01", 32))
	redeemer := common.HexToAddress("0x2000000000000000000000000000000000000002")
	signer, err := decimal.EVMCheckSignerFromPassword("decimal-check-public-fixture")
	if err != nil {
		return decimal.Call{}, hash, common.Address{}, err
	}
	defer signer.Close()
	signature, err := decimal.SignEVMCheckRedemption(context.Background(), signer, hash, redeemer)
	if err != nil {
		return decimal.Call{}, hash, signer.Address(), err
	}
	if err = decimal.VerifyEVMCheckRedemption(hash, redeemer, signer.Address(), signature); err != nil {
		return decimal.Call{}, hash, signer.Address(), err
	}
	// A zero client is used only for offline ABI construction, never for RPC.
	checks, err := new(decimal.Client).Checks(common.HexToAddress("0x3000000000000000000000000000000000000003"))
	if err != nil {
		return decimal.Call{}, hash, signer.Address(), err
	}
	call, err := checks.BuildSignedRedemption(redeemer, []decimal.EVMCheckRedemptionProof{{
		CheckHash: hash, Signer: signer.Address(), Signature: signature,
	}})
	return call, hash, signer.Address(), err
}

func main() {
	call, hash, signer, err := buildExample()
	if err != nil {
		panic(err)
	}
	fmt.Printf("fixture_check=%s signer=%s calldata_bytes=%d\n", hash.Hex(), signer.Hex(), len(call.Data))
}
