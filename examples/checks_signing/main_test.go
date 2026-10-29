package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	decimal "github.com/maxwell2010/decimal-go-sdk"
)

func TestExampleBuildsVerifiedFixtureWithoutRPC(t *testing.T) {
	call, hash, signer, err := buildExample()
	if err != nil {
		t.Fatal(err)
	}
	if signer != common.HexToAddress("0x35c3b2d1c8AE237d0E072CC8Fb739a167146F36a") || call.To != common.HexToAddress("0x3000000000000000000000000000000000000003") || call.Signature != "redeemChecks(bytes[],bytes32[])" || call.Value.Sign() != 0 {
		t.Fatal("unexpected fixture or call")
	}
	raw, err := decimal.ABIJSON("Checks")
	if err != nil {
		t.Fatal(err)
	}
	a, err := abi.JSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	m, err := a.MethodById(call.Data[:4])
	if err != nil {
		t.Fatal(err)
	}
	values, err := m.Inputs.Unpack(call.Data[4:])
	if err != nil {
		t.Fatal(err)
	}
	signatures := values[0].([][]byte)
	hashes := values[1].([][32]byte)
	if len(signatures) != 1 || len(hashes) != 1 || hashes[0] != [32]byte(hash) || len(signatures[0]) != 65 || signatures[0][64] != 28 {
		t.Fatal("redemption arguments changed")
	}
	if common.BytesToHash(hashes[0][:]) != common.HexToHash("0x"+strings.Repeat("01", 32)) {
		t.Fatal("invented issued check identity")
	}
	if err := decimal.VerifyEVMCheckRedemption(hash, common.HexToAddress("0x2000000000000000000000000000000000000002"), signer, signatures[0]); err != nil {
		t.Fatal(err)
	}
}
