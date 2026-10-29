package decimal

import (
	"encoding/json"
	"math/big"
	"testing"
)

func referenceSpec(t *testing.T, name string) ReferenceOperation {
	t.Helper()
	for _, r := range ReferenceOperations() {
		if r.Operation == name {
			return r
		}
	}
	t.Fatalf("missing operation %s", name)
	return ReferenceOperation{}
}

func TestReferenceUsesSubmissionTargetForNFTReset(t *testing.T) {
	c := &Client{}
	wrong, _ := c.Delegation(testAddress)
	for _, name := range []string{"withdrawNFTWithReset", "transferNFTWithReset", "holdNFTWithReset"} {
		r := referenceSpec(t, name)
		if r.Status != "explicit-deployment-ABI-required" || len(r.Profiles) != 1 || r.Profiles[0] != "DelegationNFT" {
			t.Fatalf("%s incorrectly inherits estimateGas target: %+v", name, r)
		}
		// A coin-withdraw signature must never be routed as an NFT reset merely
		// because the upstream estimate branch happens to target that contract.
		if _, e := wrong.BuildReferenceOperation(name, nil, testAddress, testAddress, big.NewInt(1), []*big.Int{}); e == nil {
			t.Fatalf("%s accepted coin delegation target", name)
		}
	}
}

func TestReferencePublicMethodDoesNotCapturePrivateMultisig(t *testing.T) {
	r := referenceSpec(t, "updateValidatorMeta")
	if len(r.Methods) != 1 || r.Methods[0] != "updateValidatorMeta" || len(r.Profiles) != 1 || r.Profiles[0] != "MasterValidator" {
		t.Fatalf("private method leaked into public operation: %+v", r)
	}
	for _, r := range ReferenceOperations() {
		if r.Operation == "mintNFT" || r.Operation == "passwordToPrivateKey" || r.Operation == "getAccountAndBridge" {
			t.Fatalf("non-public-write operation in inventory: %s", r.Operation)
		}
	}
}

func TestReferenceCollectionKindNeverSwitchesByArgumentShape(t *testing.T) {
	c := &Client{}
	contract, e := c.ContractWithABI(testAddress, json.RawMessage(`[{"type":"function","name":"createDRC721","stateMutability":"nonpayable","inputs":[{"name":"label","type":"string"}],"outputs":[]},{"type":"function","name":"createDRC1155","stateMutability":"nonpayable","inputs":[{"name":"label","type":"string"}],"outputs":[]}]`))
	if e != nil {
		t.Fatal(e)
	}
	a, e := contract.BuildReferenceOperation("createCollectionDRC721", nil, "fixture")
	if e != nil || a.Signature != "createDRC721(string)" {
		t.Fatal("721 routed incorrectly", e)
	}
	b, e := contract.BuildReferenceOperation("createCollectionDRC1155", nil, "fixture")
	if e != nil || b.Signature != "createDRC1155(string)" {
		t.Fatal("1155 routed incorrectly", e)
	}
}

func TestReferenceExplicitSignaturePreservesOperation(t *testing.T) {
	c := &Client{}
	token, _ := c.Token(testAddress)
	call, e := token.BuildReferenceOperationSignature("transferToken", "transfer(address,uint256)", nil, testAddress, big.NewInt(3))
	if e != nil || call.Signature != "transfer(address,uint256)" {
		t.Fatal(e)
	}
	if _, e = token.BuildReferenceOperationSignature("transferToken", "approve(address,uint256)", nil, testAddress, big.NewInt(3)); e == nil {
		t.Fatal("approve substituted for transfer")
	}
	if _, e = token.BuildReferenceOperationSignature("transferToken", "transfer", nil); e == nil {
		t.Fatal("short selector accepted")
	}
	if _, e = token.BuildReferenceOperationSignature("multiSendToken", "approve(address,uint256)", nil, testAddress, big.NewInt(3)); e == nil {
		t.Fatal("disabled spender route reopened")
	}
}

func TestReferenceOperationMetadataIsAnIsolatedCopy(t *testing.T) {
	first := ReferenceOperations()
	if len(first) != 93 {
		t.Fatal(len(first))
	}
	first[0].Methods[0] = "corrupt"
	if ReferenceOperations()[0].Methods[0] == "corrupt" {
		t.Fatal("caller mutated global routing table")
	}
	var c *Contract
	if _, e := c.BuildReferenceOperation("transferToken", nil); e == nil {
		t.Fatal("nil contract")
	}
	if _, e := c.BuildReferenceOperationSignature("transferToken", "transfer(address,uint256)", nil); e == nil {
		t.Fatal("nil contract")
	}
}

func TestVersionedTestnetNFTBuilderKeepsProxyAndTupleGeneration(t *testing.T) {
	c := &Client{}
	profile, e := c.TestnetNFTCenter20261029(testAddress)
	if e != nil {
		t.Fatal(e)
	}
	meta := TestnetNFTCenter20261029CreateDRC721Arg0{Creator: testAddress, Symbol: "FIXTURE", Name: "fixture", ContractURI: "ipfs://fixture", Refundable: true, Burnable: true}
	call, e := profile.BuildCreateDRC721(meta)
	if e != nil {
		t.Fatal(e)
	}
	if call.To != testAddress || call.Profile != "TestnetNFTCenter20261029" || call.Signature != "createDRC721((address,string,string,string,bool,bool))" {
		t.Fatal("wrong deployment target or tuple generation")
	}
	ref, e := profile.BuildReferenceOperation("createCollectionDRC721", nil, meta)
	if e != nil || string(ref.Data) != string(call.Data) {
		t.Fatal("versioned reference routing", e)
	}
}
