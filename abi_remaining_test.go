package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"math/big"
	"os"
	"reflect"
	"testing"
)

func TestAdditionalWeightedSafeABIProfiles(t *testing.T) {
	raw, e := os.ReadFile("docs/additional-deployment-catalog.json")
	if e != nil {
		t.Fatal(e)
	}
	var catalog []struct{ Profile, Signature, Method string }
	if e = json.Unmarshal(raw, &catalog); e != nil {
		t.Fatal(e)
	}
	if len(catalog) != 39 {
		t.Fatalf("unexpected additional catalog %d", len(catalog))
	}
	f := newFake()
	client := testClient(t, f)
	for _, row := range catalog {
		t.Run(row.Profile+"/"+row.Method, func(t *testing.T) {
			constructor := reflect.ValueOf(client).MethodByName(row.Profile)
			if !constructor.IsValid() {
				t.Fatal("missing profile")
			}
			created := constructor.Call([]reflect.Value{reflect.ValueOf(testAddress)})
			if !created[1].IsNil() {
				t.Fatal(created[1].Interface())
			}
			methodGo := created[0].MethodByName(row.Method)
			a, e := loadABI(row.Profile)
			if e != nil {
				t.Fatal(e)
			}
			_, m, e := method(a, row.Signature)
			if e != nil {
				t.Fatal(e)
			}
			args := make([]reflect.Value, methodGo.Type().NumIn())
			for i := range args {
				if methodGo.Type().In(i) == reflect.TypeOf((*context.Context)(nil)).Elem() {
					args[i] = reflect.ValueOf(context.Background())
				} else {
					args[i] = abiFixture(methodGo.Type().In(i))
				}
			}
			f.call = func(msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
				if string(msg.Data[:4]) != string(m.ID) {
					t.Error("selector mismatch")
				}
				values := make([]any, len(m.Outputs))
				for i, x := range m.Outputs {
					values[i] = abiFixture(x.Type.GetType()).Interface()
				}
				return m.Outputs.Pack(values...)
			}
			result := methodGo.Call(args)
			if !result[1].IsNil() {
				t.Fatal(result[1].Interface())
			}
			if !m.IsConstant() {
				call := result[0].Interface().(Call)
				if string(call.Data[:4]) != string(m.ID) {
					t.Fatal("builder selector mismatch")
				}
				if _, e = m.Inputs.Unpack(call.Data[4:]); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
	if len(f.sent) != 0 {
		t.Fatal("ABI test broadcast")
	}
}
func TestAdditionalABIProvenance(t *testing.T) {
	b, e := os.ReadFile("docs/additional-deployment-sources.json")
	if e != nil {
		t.Fatal(e)
	}
	var manifest struct {
		Profiles []struct {
			Profile string
			Hash    string `json:"normalized_abi_sha256"`
		}
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		t.Fatal(e)
	}
	for _, row := range manifest.Profiles {
		b, e := ABIJSON(row.Profile)
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != row.Hash {
			t.Fatal("ABI provenance mismatch")
		}
	}
}
