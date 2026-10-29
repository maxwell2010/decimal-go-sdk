package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Exercise the generated Go types against ABI wire encoding, including every
// overload and nested tuple. This catches generator/runtime integration defects;
// it does not assert deployed business rules or on-chain authorization.
func TestAllGeneratedABIWrappers(t *testing.T) {
	raw, e := os.ReadFile("docs/contract-catalog.json")
	if e != nil {
		t.Fatal(e)
	}
	var catalog []struct{ Profile, Signature, Method string }
	if e = json.Unmarshal(raw, &catalog); e != nil {
		t.Fatal(e)
	}
	if len(catalog) != 300 {
		t.Fatalf("unexpected pinned catalog size %d", len(catalog))
	}
	var deployment []struct{ Profile, Signature, Method string }
	catalogs, e := filepath.Glob("docs/*deployment-catalog.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range catalogs {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		var rows []struct{ Profile, Signature, Method string }
		if e = json.Unmarshal(raw, &rows); e != nil {
			t.Fatal(e)
		}
		deployment = append(deployment, rows...)
	}
	if len(deployment) == 0 {
		t.Fatal("missing deployment catalog")
	}
	var manifest struct{ Profiles []struct{ Profile string } }
	manifests, e := filepath.Glob("docs/*deployment-sources.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range manifests {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		var rows struct{ Profiles []struct{ Profile string } }
		if e = json.Unmarshal(raw, &rows); e != nil {
			t.Fatal(e)
		}
		manifest.Profiles = append(manifest.Profiles, rows.Profiles...)
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, row := range deployment {
		key := row.Profile + row.Signature
		if seen[key] {
			t.Fatal("duplicate deployment selector")
		}
		seen[key] = true
		counts[row.Profile]++
	}
	for _, profile := range manifest.Profiles {
		a, e := loadABI(profile.Profile)
		if e != nil {
			t.Fatal(e)
		}
		if counts[profile.Profile] != len(a.Methods) {
			t.Fatalf("deployment catalog incomplete for %s", profile.Profile)
		}
		delete(counts, profile.Profile)
	}
	if len(counts) != 0 {
		t.Fatal("deployment catalog profiles lack provenance")
	}
	catalog = append(catalog, deployment...)
	f := newFake()
	client := testClient(t, f)
	wrappers := map[string]any{}
	wrappers["ContractCenter"], _ = client.ContractCenter(testAddress)
	wrappers["Delegation"], _ = client.Delegation(testAddress)
	wrappers["DelegationNFT"], _ = client.DelegationNFT(testAddress)
	wrappers["ERC721"], _ = client.ERC721(testAddress)
	wrappers["ERC1155"], _ = client.ERC1155(testAddress)
	wrappers["NFTCenter"], _ = client.NFTCenter(testAddress)
	wrappers["Token"], _ = client.Token(testAddress)
	wrappers["TokenCenter"], _ = client.TokenCenter(testAddress)
	wrappers["MasterValidator"], _ = client.MasterValidator(testAddress)
	wrappers["Checks"], _ = client.Checks(testAddress)
	wrappers["Multicall"], _ = client.Multicall(testAddress)
	for _, entry := range catalog {
		t.Run(entry.Profile+"/"+entry.Method, func(t *testing.T) {
			if wrappers[entry.Profile] == nil {
				factory := reflect.ValueOf(client).MethodByName(entry.Profile)
				if !factory.IsValid() {
					t.Fatal("missing profile constructor")
				}
				built := factory.Call([]reflect.Value{reflect.ValueOf(testAddress)})
				if !built[1].IsNil() {
					t.Fatal(built[1].Interface())
				}
				wrappers[entry.Profile] = built[0].Interface()
			}
			a, e := loadABI(entry.Profile)
			if e != nil {
				t.Fatal(e)
			}
			_, m, e := method(a, entry.Signature)
			if e != nil {
				t.Fatal(e)
			}
			wrapper := reflect.ValueOf(wrappers[entry.Profile]).MethodByName(entry.Method)
			if !wrapper.IsValid() {
				t.Fatal("missing generated wrapper")
			}
			arguments := make([]reflect.Value, wrapper.Type().NumIn())
			for i := range arguments {
				if wrapper.Type().In(i) == reflect.TypeOf((*context.Context)(nil)).Elem() {
					arguments[i] = reflect.ValueOf(context.Background())
				} else {
					arguments[i] = abiFixture(wrapper.Type().In(i))
				}
			}
			f.call = func(msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
				if len(msg.Data) < 4 || string(msg.Data[:4]) != string(m.ID) {
					t.Error("reader selector mismatch")
				}
				if _, e := m.Inputs.Unpack(msg.Data[4:]); e != nil {
					t.Error(e)
				}
				outputs := make([]any, len(m.Outputs))
				for i, arg := range m.Outputs {
					outputs[i] = abiFixture(arg.Type.GetType()).Interface()
				}
				return m.Outputs.Pack(outputs...)
			}
			results := wrapper.Call(arguments)
			if !results[1].IsNil() {
				t.Fatal(results[1].Interface())
			}
			if !m.IsConstant() {
				call := results[0].Interface().(Call)
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
		t.Fatal("ABI wrappers broadcast")
	}
}

func abiFixture(typ reflect.Type) reflect.Value {
	if typ == reflect.TypeOf((*big.Int)(nil)) {
		return reflect.ValueOf(big.NewInt(7))
	}
	v := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			v.Field(i).Set(abiFixture(typ.Field(i).Type))
		}
	case reflect.Slice:
		v = reflect.MakeSlice(typ, 2, 2)
		for i := 0; i < 2; i++ {
			v.Index(i).Set(abiFixture(typ.Elem()))
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			v.Index(i).Set(abiFixture(typ.Elem()))
		}
	case reflect.String:
		v.SetString("fixture")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	}
	return v
}

func TestEmbeddedABIProvenance(t *testing.T) {
	raw, e := os.ReadFile("docs/sources.json")
	if e != nil {
		t.Fatal(e)
	}
	var manifest struct {
		ABIs []struct {
			Profile string
			Hash    string `json:"abi_sha256"`
		}
	}
	if e = json.Unmarshal(raw, &manifest); e != nil {
		t.Fatal(e)
	}
	for _, source := range manifest.ABIs {
		b, e := ABIJSON(source.Profile)
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != source.Hash {
			t.Fatalf("ABI artifact changed: %s", source.Profile)
		}
	}
}

func TestVersionedABIProvenance(t *testing.T) {
	paths, e := filepath.Glob("docs/*deployment-sources.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		var source struct {
			Profiles []struct {
				Profile string
				Hash    string `json:"normalized_abi_sha256"`
			}
		}
		if e = json.Unmarshal(raw, &source); e != nil {
			t.Fatal(e)
		}
		for _, profile := range source.Profiles {
			b, e := ABIJSON(profile.Profile)
			if e != nil {
				t.Fatal(e)
			}
			digest := sha256.Sum256(b)
			if profile.Hash == "" || hex.EncodeToString(digest[:]) != profile.Hash {
				t.Fatalf("versioned ABI drift: %s", profile.Profile)
			}
		}
	}
}
