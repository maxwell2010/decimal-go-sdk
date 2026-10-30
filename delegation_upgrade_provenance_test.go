package decimal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestDelegationUpgradeABIProvenance(t *testing.T) {
	raw, err := os.ReadFile("docs/DELEGATION_UPGRADE_SOURCES.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Commit string
		ABI    struct {
			Path   string
			SHA256 string
		} `json:"compatibility_abi"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(delegationUpgradeJSON))
	if source.Commit != "a46cb48d7965c1a64a495cebb22e735dfe4b9da3" || source.ABI.Path != "delegation_compat/Delegation20261030.json" || source.ABI.SHA256 != hex.EncodeToString(hash[:]) {
		t.Fatal("official upgrade ABI provenance mismatch")
	}
	a, err := delegationUpgradeABI()
	if err != nil {
		t.Fatal(err)
	}
	m := a.Methods["getStakes"]
	if hex.EncodeToString(m.ID) != "226f6ea2" || len(m.Outputs[0].Type.Elem.TupleElems) != 8 || m.Outputs[0].Type.Elem.TupleRawNames[7] != "holdStartTime" {
		t.Fatal("unexpected upgraded Stake layout or selector")
	}
	legacy, err := loadABI("DelegationNFT")
	if err != nil || len(legacy.Methods["getStakes"].Outputs[0].Type.Elem.TupleElems) != 7 {
		t.Fatal("legacy NFT format changed", err)
	}
}
