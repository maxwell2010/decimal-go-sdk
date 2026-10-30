package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// These are supplied saved read observations, not fresh network acceptance.
func TestDelegationSavedMainnetObservations(t *testing.T) {
	raw, err := os.ReadFile("docs/DELEGATION_UPGRADE_SOURCES.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Fixtures []struct{ Path, SHA256 string } `json:"saved_rpc_fixtures"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil || len(manifest.Fixtures) != 3 {
		t.Fatal("missing RPC fixture provenance", err)
	}
	for _, f := range manifest.Fixtures {
		b, err := os.ReadFile(f.Path)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != f.SHA256 {
			t.Fatal("saved observation bytes changed", f.Path)
		}
	}
	raw, err = os.ReadFile("testdata/delegation/mainnet_stakes_post_recovery.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Block, Address, ReturnData string
		InputStakeIDs              []common.Hash
		VerifiedFields             int
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Block != "0x20c2060" || len(fixture.InputStakeIDs) != 8 || fixture.VerifiedFields != 8 {
		t.Fatal("unexpected saved snapshot shape")
	}
	data := common.FromHex(fixture.ReturnData)
	if len(data) != 64+8*8*32 {
		t.Fatal("unexpected raw tuple count or width")
	}
	backend := &contractReadAuditBackend{fakeRPC: newFake(), read: func(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		if msg.To == nil || *msg.To != common.HexToAddress(fixture.Address) || block == nil || block.Uint64() != 34349152 {
			t.Fatal("saved snapshot request identity lost")
		}
		return data, nil
	}}
	backend.chain = big.NewInt(75)
	client, err := NewClient(Mainnet(), backend)
	if err != nil {
		t.Fatal(err)
	}
	d, err := client.Delegation(common.HexToAddress(fixture.Address))
	if err != nil {
		t.Fatal(err)
	}
	ids := make([][32]byte, len(fixture.InputStakeIDs))
	for i, id := range fixture.InputStakeIDs {
		ids[i] = id
	}
	values, err := d.ReadAt(context.Background(), new(big.Int).SetUint64(34349152), "getStakes(bytes32[])", ids)
	if err != nil {
		t.Fatal(err)
	}
	stakes := reflect.ValueOf(values[0])
	if stakes.Len() != len(ids) {
		t.Fatal("stake missing")
	}
	for i := range ids {
		for name, index := range map[string]int{"Amount": 3, "HoldTimestamp": 6, "HoldStartTime": 7} {
			word := data[64+i*256+index*32 : 64+i*256+(index+1)*32]
			got := stakes.Index(i).FieldByName(name).Interface().(*big.Int)
			if got == nil || got.Cmp(new(big.Int).SetBytes(word)) != 0 {
				t.Fatalf("raw value changed for stake %d field %s", i, name)
			}
		}
	}
	raw, err = os.ReadFile("testdata/delegation/mainnet_delegation_post_recovery.json")
	if err != nil {
		t.Fatal(err)
	}
	var logs []types.Log
	if err := json.Unmarshal(raw, &logs); err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int)
	seen := make(map[[32]byte]bool)
	for _, log := range logs {
		event, err := d.DecodeLog(log)
		if err != nil {
			t.Fatal(err)
		}
		counts[event.Name]++
		if event.Name != "StakeUpdated" {
			continue
		}
		id := event.Values["stakeId"].([32]byte)
		if seen[id] {
			t.Fatal("duplicate update in saved sample")
		}
		seen[id] = true
		stake := reflect.ValueOf(event.Values["stake"])
		if stake.NumField() != 8 || stake.FieldByName("HoldStartTime").IsNil() {
			t.Fatal("new event field lost")
		}
		matched := false
		for i, expected := range ids {
			if id == expected {
				matched = true
				if stake.FieldByName("Amount").Interface().(*big.Int).Cmp(stakes.Index(i).FieldByName("Amount").Interface().(*big.Int)) != 0 {
					t.Fatal("event versus saved getter amount mismatch")
				}
			}
		}
		if !matched {
			t.Fatal("event not matched to requested stake")
		}
	}
	if len(logs) != 16 || counts["StakeUpdated"] != 8 || counts["StakeAmountUpdated"] != 8 || len(seen) != 8 {
		t.Fatal("incomplete saved event sample", counts)
	}
}
