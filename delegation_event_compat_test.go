package decimal

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func delegationEventFixtureValue(typ abi.Type) any {
	switch typ.T {
	case abi.TupleTy:
		v := reflect.New(typ.GetType()).Elem()
		for i, child := range typ.TupleElems {
			v.Field(i).Set(reflect.ValueOf(delegationEventFixtureValue(*child)))
		}
		return v.Interface()
	case abi.AddressTy:
		return common.HexToAddress("0x1234")
	case abi.BoolTy:
		return true
	case abi.UintTy:
		if typ.Size <= 64 {
			v := reflect.New(typ.GetType()).Elem()
			v.SetUint(2)
			return v.Interface()
		}
		return new(big.Int).Lsh(big.NewInt(17), 80)
	case abi.IntTy:
		if typ.Size <= 64 {
			v := reflect.New(typ.GetType()).Elem()
			v.SetInt(-2)
			return v.Interface()
		}
		return big.NewInt(-17)
	case abi.FixedBytesTy:
		v := reflect.New(typ.GetType()).Elem()
		v.Index(v.Len() - 1).SetUint(9)
		return v.Interface()
	default:
		panic("unexpected delegation event fixture type")
	}
}

func TestDelegationUpgradeEventsKeepOriginalLayouts(t *testing.T) {
	legacy, err := loadABI("Delegation")
	if err != nil {
		t.Fatal(err)
	}
	current, err := delegationUpgradeABI()
	if err != nil {
		t.Fatal(err)
	}
	contract := &Contract{Address: testAddress, Profile: "Delegation"}
	for version, source := range map[string]abi.ABI{"legacy": legacy, "current": current} {
		for name, event := range source.Events {
			t.Run(version+"/"+name, func(t *testing.T) {
				var args []any
				topics := []common.Hash{event.ID}
				for _, arg := range event.Inputs {
					value := delegationEventFixtureValue(arg.Type)
					if arg.Indexed {
						encoded, err := (abi.Arguments{{Type: arg.Type}}).Pack(value)
						if err != nil {
							t.Fatal(err)
						}
						topics = append(topics, common.BytesToHash(encoded))
					} else {
						args = append(args, value)
					}
				}
				data, err := event.Inputs.NonIndexed().Pack(args...)
				if err != nil {
					t.Fatal(err)
				}
				log := types.Log{Address: testAddress, Topics: topics, Data: data, BlockNumber: 34342701}
				got, err := contract.DecodeLog(log)
				if err != nil || got.Signature != event.Sig || got.Name != name || !reflect.DeepEqual(got.Log, log) {
					t.Fatalf("original event identity lost: %v %v", got, err)
				}
				for i, arg := range event.Inputs.NonIndexed() {
					if !reflect.DeepEqual(got.Values[arg.Name], args[i]) {
						t.Fatalf("field %s changed", arg.Name)
					}
				}
				log.Data = append(append([]byte{}, data...), make([]byte, 32)...)
				if _, err := contract.DecodeLog(log); err == nil {
					t.Fatal("trailing event data accepted")
				}
				log.Data = data
				log.Topics = append(append([]common.Hash{}, topics...), common.Hash{})
				if _, err := contract.DecodeLog(log); err == nil {
					t.Fatal("extra topic accepted")
				}
			})
		}
	}
}

func TestDelegationUpgradeEventsRejectBadPaddingAndWrongProfile(t *testing.T) {
	a, err := delegationUpgradeABI()
	if err != nil {
		t.Fatal(err)
	}
	event := a.Events["NFTDelegated"]
	var args []any
	for _, arg := range event.Inputs.NonIndexed() {
		args = append(args, delegationEventFixtureValue(arg.Type))
	}
	data, err := event.Inputs.NonIndexed().Pack(args...)
	if err != nil {
		t.Fatal(err)
	}
	log := types.Log{Address: testAddress, Topics: []common.Hash{event.ID, {}, {}, common.HexToHash("0x1234")}, Data: data}
	c := &Contract{Address: testAddress, Profile: "Delegation"}
	if _, err := c.DecodeLog(log); err != nil {
		t.Fatal(err)
	}
	log.Topics[3][0] = 1
	if _, err := c.DecodeLog(log); err == nil {
		t.Fatal("noncanonical indexed address accepted")
	}
	log.Topics[3][0] = 0
	log.Data[0] = 1
	if _, err := c.DecodeLog(log); err == nil {
		t.Fatal("noncanonical address in data accepted")
	}
	log.Data = data
	c.Profile = "DelegationNFT"
	if _, err := c.DecodeLog(log); err == nil {
		t.Fatal("unified event incorrectly decoded by legacy NFT profile")
	}
}
