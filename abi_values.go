package decimal

import (
	"errors"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// normalizeArguments makes integer widths explicit and rejects truncation before
// go-ethereum's ABI encoder (which otherwise accepts some oversized big.Ints).
func normalizeArguments(types abi.Arguments, args []any) ([]any, error) {
	if len(types) != len(args) {
		return nil, errors.New("ABI argument count mismatch")
	}
	out := make([]any, len(args))
	for i, arg := range args {
		v, e := normalizeArgument(types[i].Type, reflect.ValueOf(arg))
		if e != nil {
			return nil, e
		}
		out[i] = v.Interface()
	}
	return out, nil
}
func normalizeArgument(t abi.Type, v reflect.Value) (reflect.Value, error) {
	bad := func() (reflect.Value, error) {
		return reflect.Value{}, errors.New("invalid ABI argument type or range")
	}
	if !v.IsValid() || !v.CanInterface() {
		return bad()
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return bad()
		}
		v = v.Elem()
	}
	if t.T == abi.UintTy || t.T == abi.IntTy {
		var n *big.Int
		if v.Type() == reflect.TypeOf((*big.Int)(nil)) {
			if v.IsNil() {
				return bad()
			}
			n = new(big.Int).Set(v.Interface().(*big.Int))
		} else {
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n = big.NewInt(v.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				n = new(big.Int).SetUint64(v.Uint())
			default:
				return bad()
			}
		}
		if t.T == abi.UintTy {
			if n.Sign() < 0 || n.BitLen() > t.Size {
				return bad()
			}
		} else {
			bound := new(big.Int).Lsh(big.NewInt(1), uint(t.Size-1))
			if n.Cmp(new(big.Int).Neg(bound)) < 0 || n.Cmp(bound) >= 0 {
				return bad()
			}
		}
		if t.GetType().Kind() == reflect.Pointer {
			return reflect.ValueOf(n), nil
		}
		out := reflect.New(t.GetType()).Elem()
		if t.T == abi.UintTy {
			out.SetUint(n.Uint64())
		} else {
			out.SetInt(n.Int64())
		}
		return out, nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return bad()
		}
		v = v.Elem()
	}
	switch t.T {
	case abi.TupleTy:
		if v.Kind() != reflect.Struct {
			return bad()
		}
		out := reflect.New(t.GetType()).Elem()
		for i, element := range t.TupleElems {
			field := v.FieldByName(t.GetType().Field(i).Name)
			for k := 0; k < v.NumField(); k++ {
				if v.Type().Field(k).Tag.Get("abi") == t.TupleRawNames[i] && t.TupleRawNames[i] != "" {
					field = v.Field(k)
					break
				}
			}
			if !field.IsValid() {
				return bad()
			}
			converted, e := normalizeArgument(*element, field)
			if e != nil {
				return reflect.Value{}, e
			}
			out.Field(i).Set(converted)
		}
		return out, nil
	case abi.SliceTy, abi.ArrayTy:
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return bad()
		}
		if t.T == abi.ArrayTy && v.Len() != t.Size {
			return bad()
		}
		var out reflect.Value
		if t.T == abi.SliceTy {
			out = reflect.MakeSlice(t.GetType(), v.Len(), v.Len())
		} else {
			out = reflect.New(t.GetType()).Elem()
		}
		for i := 0; i < v.Len(); i++ {
			converted, e := normalizeArgument(*t.Elem, v.Index(i))
			if e != nil {
				return reflect.Value{}, e
			}
			out.Index(i).Set(converted)
		}
		return out, nil
	default:
		if v.Type() != t.GetType() {
			return bad()
		}
		return v, nil
	}
}
