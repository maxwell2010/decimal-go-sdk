package decimal

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestRetiredMainnetNFTDepositsFailOffline(t *testing.T) {
	a, err := loadABI("DelegationNFT")
	if err != nil {
		t.Fatal(err)
	}
	retired := common.HexToAddress("0xe45adfcc739a0d10ce9462b58866c9a1a06035e2")
	custom := Mainnet()
	custom.ChainID = big.NewInt(777)
	for name, cfg := range map[string]Config{"mainnet": Mainnet(), "testnet": Testnet(), "custom": custom} {
		client, err := NewClient(cfg, newFake())
		if err != nil {
			t.Fatal(err)
		}
		for _, address := range []common.Address{retired, testAddress} {
			d, err := client.DelegationNFT(address)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range a.Methods {
				if m.IsConstant() {
					continue
				}
				args := make([]any, len(m.Inputs))
				for i, arg := range m.Inputs {
					args[i] = stakeCompatFixture(arg.Type.GetType()).Interface()
				}
				_, err := d.Build(m.Sig, nil, args...)
				blocked := name == "mainnet" && address == retired && (strings.HasPrefix(m.RawName, "delegate") || m.RawName == "hold")
				if blocked != (err != nil) {
					t.Fatalf("unexpected route result %s %s %s: %v", name, address, m.Sig, err)
				}
			}
		}
		if client.cfg.ChainID.Cmp(cfg.ChainID) != 0 || client.cfg.ContractCenter != cfg.ContractCenter {
			t.Fatal("global client configuration changed")
		}
	}
}

func TestDelegationReceiptPreservesRepeatedExitPairing(t *testing.T) {
	for _, profile := range []string{"Delegation", "DelegationNFT", "UnifiedNFTDelegation20261030"} {
		t.Run(profile, func(t *testing.T) { testDelegationReceiptPairing(t, profile) })
	}
}

func testDelegationReceiptPairing(t *testing.T, profile string) {
	old, _ := loadABI("Delegation")
	current, _ := delegationUpgradeABI()
	c := &Contract{Address: testAddress, Profile: profile}
	if profile == "DelegationNFT" {
		old, _ = loadABI("DelegationNFT")
	} else if profile == "UnifiedNFTDelegation20261030" {
		c.override = &current
	}
	r := &types.Receipt{}
	for i, source := range []struct{ current bool }{{false}, {true}} {
		a := old
		if (source.current && profile != "DelegationNFT") || profile == "UnifiedNFTDelegation20261030" {
			a = current
		}
		e := a.Events["WithdrawRequest"]
		var args []any
		for _, arg := range e.Inputs.NonIndexed() {
			value := delegationEventFixtureValue(arg.Type)
			if arg.Name == "stakeIndex" {
				value = big.NewInt(int64(7 + i))
			}
			args = append(args, value)
		}
		data, err := e.Inputs.NonIndexed().Pack(args...)
		if err != nil {
			t.Fatal(err)
		}
		r.Logs = append(r.Logs, &types.Log{Address: testAddress, Topics: []common.Hash{e.ID}, Data: data, Index: uint(i)})
	}
	foreign := *r.Logs[0]
	foreign.Address = common.HexToAddress("0x9876")
	foreign.Data = []byte{1}
	r.Logs = append(r.Logs, &foreign, &types.Log{Address: testAddress, Topics: []common.Hash{common.HexToHash("0x1234")}, Data: []byte{1}})
	before, err := json.Marshal(r.Logs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.DecodeDelegationReceipt(r)
	if err != nil || len(got) != 2 {
		t.Fatal("repeated exits lost or foreign event decoded", len(got), err)
	}
	if got[0].Values["stakeId"] != got[1].Values["stakeId"] {
		t.Fatal("fixture must share one stake ID")
	}
	for i, event := range got {
		if event.Values["stakeIndex"].(*big.Int).Int64() != int64(7+i) || event.Log.Index != uint(i) {
			t.Fatal("stake ID/index order changed")
		}
	}
	after, _ := json.Marshal(r.Logs)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("source receipt mutated")
	}
	r.Logs = append(r.Logs, &types.Log{Address: testAddress, Topics: append([]common.Hash{}, r.Logs[0].Topics...), Data: []byte{1}})
	if partial, err := c.DecodeDelegationReceipt(r); err == nil || partial != nil {
		t.Fatal("malformed known event became a partial result")
	}
	if _, err := c.DecodeDelegationReceipt(nil); err == nil {
		t.Fatal("nil receipt accepted")
	}
}
