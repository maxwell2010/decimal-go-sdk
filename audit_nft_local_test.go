package decimal

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestUnifiedNFTDelegationRejectsMissingTargetWithoutPanic(t *testing.T) {
	valid, err := (&Client{}).UnifiedNFTDelegation(testAddress)
	if err != nil {
		t.Fatal(err)
	}
	valid.Address = common.Address{} // Address is exported and can be changed after construction.
	for _, d := range []*UnifiedNFTDelegation{nil, {}, valid} {
		if d.ApprovalOperator() != (common.Address{}) {
			t.Fatal("missing target has an operator")
		}
		if _, err := d.BuildDelegateNFT(testAddress, testAddress, big.NewInt(1)); err == nil {
			t.Fatal("missing action destination accepted")
		}
		if _, err := d.BuildERC721Approval(testAddress, big.NewInt(1)); err == nil {
			t.Fatal("missing operator became token approval revocation")
		}
		if _, err := d.BuildERC1155ApprovalForAll(testAddress); err == nil {
			t.Fatal("zero operator grant accepted")
		}
		if _, err := d.ReadNFTStakeAt(context.Background(), big.NewInt(1), testAddress, big.NewInt(1), testAddress, testAddress, big.NewInt(0)); err == nil {
			t.Fatal("missing read target accepted")
		}
		if _, err := d.ReadNFTStakeIDAt(context.Background(), big.NewInt(1), testAddress, big.NewInt(1), testAddress, testAddress, big.NewInt(0)); err == nil {
			t.Fatal("missing ID read target accepted")
		}
	}
}
