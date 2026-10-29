package decimal

import (
	"math/big"
	"testing"
)

func TestVersionedTokenReferenceRouting(t *testing.T) {
	c := &Client{}
	reserve, _ := c.TestnetReserveToken20261029(testAddress)
	reserveless, _ := c.TestnetReservelessToken20261029(testAddress)
	for _, contract := range []*Contract{reserve.Contract, reserveless.Contract} {
		call, e := contract.BuildReferenceOperation("transferToken", nil, testAddress, big.NewInt(7))
		if e != nil || call.Signature != "transfer(address,uint256)" {
			t.Fatal("ERC20 route", e)
		}
		if _, e = contract.BuildReferenceOperationSignature("approveToken", "approve(address,uint256)", nil, testAddress, big.NewInt(7)); e != nil {
			t.Fatal(e)
		}
	}
	call, e := reserveless.BuildReferenceOperation("mintTokenReserveless", nil, testAddress, big.NewInt(7))
	if e != nil || call.Signature != "mint(address,uint256)" {
		t.Fatal("reserveless mint route", e)
	}
	if _, e = reserveless.BuildReferenceOperationSignature("mintTokenReserveless", "mint(address,uint256)", nil, testAddress, big.NewInt(7)); e != nil {
		t.Fatal(e)
	}
	if _, e = reserve.BuildReferenceOperation("mintTokenReserveless", nil, testAddress, big.NewInt(7)); e == nil {
		t.Fatal("reserve token accepted reserveless mint")
	}
}
func TestVersionedReserveUpdateSelectorIsExplicit(t *testing.T) {
	c := &Client{}
	reserve, _ := c.TestnetReserveToken20261029(testAddress)
	pinned, _ := c.Token(testAddress)
	call, e := reserve.BuildReferenceOperation("updateTokenIdentity", nil, "identity")
	if e != nil || call.Signature != "updateIdentity(string)" {
		t.Fatal("versioned identity selector", e)
	}
	if _, e = reserve.BuildReferenceOperationSignature("updateTokenIdentity", "updateIdentity(string)", nil, "identity"); e != nil {
		t.Fatal(e)
	}
	if _, e = reserve.BuildReferenceOperationSignature("updateTokenIdentity", "transfer(address,uint256)", nil, testAddress, big.NewInt(1)); e == nil {
		t.Fatal("identity operation accepted unrelated transfer")
	}
	if _, e = pinned.BuildReferenceOperation("updateTokenIdentity", nil, "identity"); e == nil {
		t.Fatal("versioned identity alias leaked to pinned Token")
	}
	if _, e = pinned.BuildReferenceOperation("updateTokenMinTotalSupply", nil, big.NewInt(1)); e == nil {
		t.Fatal("versioned min supply update leaked into pinned Token")
	}
	if _, e = reserve.BuildReferenceOperation("updateTokenMinTotalSupply", nil, big.NewInt(1)); e != nil {
		t.Fatal("observed versioned min supply update", e)
	}
	if _, e = reserve.BuildReferenceOperation("updateTokenMaxTotalSupply", nil, big.NewInt(100)); e != nil {
		t.Fatal("max supply update", e)
	}
}
