package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
	"testing"
)

func stakeCompatFixture(typ reflect.Type) reflect.Value {
	v := reflect.New(typ).Elem()
	if typ == reflect.TypeOf((*big.Int)(nil)) {
		v.Set(reflect.ValueOf(new(big.Int).Lsh(big.NewInt(123), 100)))
		return v
	}
	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			v.Field(i).Set(stakeCompatFixture(typ.Field(i).Type))
		}
	case reflect.Slice:
		v = reflect.MakeSlice(typ, 2, 2)
		for i := 0; i < 2; i++ {
			v.Index(i).Set(stakeCompatFixture(typ.Elem()))
		}
	case reflect.Array:
		v.Index(v.Len() - 1).SetUint(17)
	case reflect.Uint8:
		v.SetUint(2)
	}
	return v
}
func stakeCompatPayload(t *testing.T, m abi.Method) []byte {
	t.Helper()
	vals := make([]any, len(m.Outputs))
	for i, a := range m.Outputs {
		vals[i] = stakeCompatFixture(a.Type.GetType()).Interface()
	}
	b, e := m.Outputs.Pack(vals...)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func stakeCompatCheckTimes(t *testing.T, v reflect.Value, modern bool) {
	t.Helper()
	switch v.Kind() {
	case reflect.Struct:
		if s := v.FieldByName("HoldStartTime"); s.IsValid() {
			if s.IsNil() == modern {
				t.Fatal("missing versus known start confused")
			}
			a := v.FieldByName("Amount").Interface().(*big.Int)
			if a.Cmp(new(big.Int).Lsh(big.NewInt(123), 100)) != 0 {
				t.Fatal("amount changed")
			}
		}
		for i := 0; i < v.NumField(); i++ {
			stakeCompatCheckTimes(t, v.Field(i), modern)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			stakeCompatCheckTimes(t, v.Index(i), modern)
		}
	}
}
func TestDelegationStakeCompatibilityLayouts(t *testing.T) {
	old, e := loadABI("Delegation")
	if e != nil {
		t.Fatal(e)
	}
	next, e := delegationUpgradeABI()
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"getStake", "getStakes", "getHoldStake", "getHoldStakeDEL", "getFrozenStake", "getFrozenStakes"} {
		for _, modern := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/legacy", true: "/current"}[modern], func(t *testing.T) {
				m := old.Methods[name]
				wire := m
				if modern {
					wire = next.Methods[name]
				}
				b := stakeCompatPayload(t, wire)
				vals, e := unpackDelegationRead("Delegation", m, b)
				if e != nil {
					t.Fatal(e)
				}
				for _, v := range vals {
					stakeCompatCheckTimes(t, reflect.ValueOf(v), modern)
				}
				for _, bad := range [][]byte{nil, b[:len(b)-1], append(append([]byte{}, b...), make([]byte, 64)...)} {
					if _, e := unpackDelegationRead("Delegation", m, bad); e == nil {
						t.Fatal("accepted malformed response")
					}
				}
			})
		}
	}
	m := next.Methods["getStake"]
	v := stakeCompatFixture(m.Outputs[0].Type.GetType())
	v.FieldByName("HoldStartTime").Set(reflect.ValueOf(big.NewInt(0)))
	b, e := m.Outputs.Pack(v.Interface())
	if e != nil {
		t.Fatal(e)
	}
	vals, e := unpackDelegationRead("Delegation", old.Methods["getStake"], b)
	if e != nil {
		t.Fatal(e)
	}
	s := reflect.ValueOf(vals[0]).FieldByName("HoldStartTime").Interface().(*big.Int)
	if s == nil || s.Sign() != 0 {
		t.Fatal("known zero lost")
	}
}
func TestDelegationStakeCompatibilityArraysAndMalformed(t *testing.T) {
	old, _ := loadABI("Delegation")
	next, _ := delegationUpgradeABI()
	m := old.Methods["getStakes"]
	for _, wire := range []abi.Method{m, next.Methods["getStakes"]} {
		empty := reflect.MakeSlice(wire.Outputs[0].Type.GetType(), 0, 0).Interface()
		b, e := wire.Outputs.Pack(empty)
		if e != nil {
			t.Fatal(e)
		}
		vals, e := unpackDelegationRead("Delegation", m, b)
		if e != nil || reflect.ValueOf(vals[0]).Len() != 0 {
			t.Fatal("empty", e)
		}
		b = stakeCompatPayload(t, wire)
		for _, i := range []int{0, 31, 32, 63, 64, 64 + 5*32} {
			bad := append([]byte{}, b...)
			bad[i] = 255
			if _, e := unpackDelegationRead("Delegation", m, bad); e == nil {
				t.Fatalf("accepted mutation %d", i)
			}
		}
	}
}
func TestDelegationStakeCompatibilityTypedReaders(t *testing.T) {
	old, _ := loadABI("Delegation")
	next, _ := delegationUpgradeABI()
	for _, modern := range []bool{false, true} {
		wire := old.Methods["getStakes"]
		if modern {
			wire = next.Methods["getStakes"]
		}
		b := stakeCompatPayload(t, wire)
		backend := &contractReadAuditBackend{fakeRPC: newFake(), read: func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) { return b, nil }}
		client, e := NewClient(Testnet(), backend)
		if e != nil {
			t.Fatal(e)
		}
		d, e := client.Delegation(common.HexToAddress("0x1234"))
		if e != nil {
			t.Fatal(e)
		}
		result, e := d.ReadGetStakes(context.Background(), [][32]byte{{1}, {2}})
		if e != nil {
			t.Fatal(e)
		}
		if len(result.Value0) != 2 {
			t.Fatal("stake lost")
		}
		stakeCompatCheckTimes(t, reflect.ValueOf(result.Value0), modern)
	}
}

// Existing legacy fixtures must encode the legacy wire type, not the extended
// public result model whose extra field is intentionally absent on old nodes.
func packLegacyStakeFixture(args abi.Arguments, values ...any) ([]byte, error) {
	for i, v := range values {
		values[i] = stakeFixtureToWire(reflect.ValueOf(v), args[i].Type.GetType()).Interface()
	}
	return args.Pack(values...)
}
func stakeFixtureToWire(src reflect.Value, typ reflect.Type) reflect.Value {
	dst := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			dst.Field(i).Set(stakeFixtureToWire(src.FieldByName(typ.Field(i).Name), typ.Field(i).Type))
		}
	case reflect.Slice:
		dst = reflect.MakeSlice(typ, src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			dst.Index(i).Set(stakeFixtureToWire(src.Index(i), typ.Elem()))
		}
	default:
		dst.Set(src)
	}
	return dst
}

func TestDelegationStakeCompatibilityAllTypedReaders(t *testing.T) {
	old, _ := loadABI("Delegation")
	next, _ := delegationUpgradeABI()
	for _, name := range []string{"getStake", "getStakes", "getHoldStake", "getHoldStakeDEL", "getFrozenStake", "getFrozenStakes"} {
		for _, modern := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/legacy", true: "/current"}[modern], func(t *testing.T) {
				wire := old.Methods[name]
				if modern {
					wire = next.Methods[name]
				}
				b := stakeCompatPayload(t, wire)
				backend := &contractReadAuditBackend{fakeRPC: newFake(), read: func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) { return b, nil }}
				c, e := NewClient(Testnet(), backend)
				if e != nil {
					t.Fatal(e)
				}
				d, e := c.Delegation(testAddress)
				if e != nil {
					t.Fatal(e)
				}
				ctx := context.Background()
				var result any
				switch name {
				case "getStake":
					result, e = d.ReadGetStake(ctx, testAddress, testAddress, testAddress)
				case "getStakes":
					result, e = d.ReadGetStakes(ctx, [][32]byte{{1}, {2}})
				case "getHoldStake":
					result, e = d.ReadGetHoldStake(ctx, testAddress, testAddress, testAddress, big.NewInt(5))
				case "getHoldStakeDEL":
					result, e = d.ReadGetHoldStakeDEL(ctx, testAddress, testAddress, big.NewInt(5))
				case "getFrozenStake":
					result, e = d.ReadGetFrozenStake(ctx, big.NewInt(1))
				case "getFrozenStakes":
					result, e = d.ReadGetFrozenStakes(ctx, []*big.Int{big.NewInt(1), big.NewInt(2)})
				}
				if e != nil {
					t.Fatal(e)
				}
				stakeCompatCheckTimes(t, reflect.ValueOf(result), modern)
			})
		}
	}
}
func TestDelegationStakeCompatibilityCustomAndNFTUnchanged(t *testing.T) {
	old, _ := loadABI("Delegation")
	b := stakeCompatPayload(t, old.Methods["getStakes"])
	backend := &contractReadAuditBackend{fakeRPC: newFake(), read: func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) { return b, nil }}
	c, e := NewClient(Testnet(), backend)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := abiFiles.ReadFile("abi/Delegation.json")
	if e != nil {
		t.Fatal(e)
	}
	custom, e := c.ContractWithABI(testAddress, raw)
	if e != nil {
		t.Fatal(e)
	}
	vals, e := custom.Read(context.Background(), "getStakes(bytes32[])", [][32]byte{{1}, {2}})
	if e != nil {
		t.Fatal(e)
	}
	if reflect.ValueOf(vals[0]).Index(0).FieldByName("HoldStartTime").IsValid() {
		t.Fatal("custom ABI changed")
	}
	nftABI, _ := loadABI("DelegationNFT")
	b = stakeCompatPayload(t, nftABI.Methods["getStakes"])
	nft, e := c.DelegationNFT(testAddress)
	if e != nil {
		t.Fatal(e)
	}
	r, e := nft.ReadGetStakes(context.Background(), [][32]byte{{1}, {2}})
	if e != nil || len(r.Value0) != 2 {
		t.Fatal("NFT compatibility", e)
	}
	if reflect.ValueOf(r.Value0[0]).FieldByName("HoldStartTime").IsValid() {
		t.Fatal("unverified NFT ABI extension")
	}
}
