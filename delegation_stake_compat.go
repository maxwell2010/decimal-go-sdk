package decimal

import (
	"bytes"
	_ "embed"
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed delegation_compat/Delegation20261030.json
var delegationUpgradeJSON string
var delegationUpgradeOnce sync.Once
var delegationUpgradeParsed abi.ABI
var delegationUpgradeError error

func delegationUpgradeABI() (abi.ABI, error) {
	delegationUpgradeOnce.Do(func() {
		delegationUpgradeParsed, delegationUpgradeError = abi.JSON(strings.NewReader(delegationUpgradeJSON))
	})
	return delegationUpgradeParsed, delegationUpgradeError
}

// canonicalDelegationUnpack rejects noncanonical padding, offsets, counts,
// truncation and trailing words instead of silently dropping a stake.
func canonicalDelegationUnpack(args abi.Arguments, b []byte) ([]any, error) {
	values, err := args.Unpack(b)
	if err != nil {
		return nil, err
	}
	encoded, err := args.Pack(values...)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(encoded, b) {
		return nil, errors.New("noncanonical delegation stake response")
	}
	return values, nil
}

func unpackDelegationRead(profile string, m abi.Method, b []byte) ([]any, error) {
	if profile != "Delegation" {
		return m.Outputs.Unpack(b)
	}
	switch m.Sig {
	case "getFrozenStake(uint256)", "getFrozenStakes(uint256[])", "getHoldStake(address,address,address,uint256)", "getHoldStakeDEL(address,address,uint256)", "getStake(address,address,address)", "getStakes(bytes32[])":
	default:
		return m.Outputs.Unpack(b)
	}
	upgraded, err := delegationUpgradeABI()
	if err != nil {
		return nil, err
	}
	next, ok := upgraded.Methods[m.RawName]
	if !ok || next.Sig != m.Sig {
		return nil, errors.New("missing upgraded delegation getter")
	}
	values, err := canonicalDelegationUnpack(next.Outputs, b)
	if err == nil {
		return values, nil
	}
	values, err = canonicalDelegationUnpack(m.Outputs, b)
	if err != nil {
		return nil, errors.New("invalid delegation stake response: expected canonical seven or eight field layout")
	}
	for i, value := range values {
		values[i] = upgradeDelegationValue(reflect.ValueOf(value), next.Outputs[i].Type.GetType()).Interface()
	}
	return values, nil
}

// Add only the missing HoldStartTime pointer. Nil means the legacy response
// supplied no start time; an explicitly returned zero remains nonnil.
func upgradeDelegationValue(src reflect.Value, target reflect.Type) reflect.Value {
	dst := reflect.New(target).Elem()
	switch target.Kind() {
	case reflect.Struct:
		for i := 0; i < target.NumField(); i++ {
			field := src.FieldByName(target.Field(i).Name)
			if field.IsValid() {
				dst.Field(i).Set(upgradeDelegationValue(field, target.Field(i).Type))
			}
		}
	case reflect.Slice:
		dst = reflect.MakeSlice(target, src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			dst.Index(i).Set(upgradeDelegationValue(src.Index(i), target.Elem()))
		}
	default:
		dst.Set(src)
	}
	return dst
}
