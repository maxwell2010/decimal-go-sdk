package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestMainnetProfilesKeepNetworkGuards(t *testing.T) {
	cases := []struct{ profile, method string }{
		{"MainnetNFTCenter20261103", "createDRC721"},
		{"MainnetReserveToken20261103", "updateMinTotalSupply"},
	}
	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			backend := newFake()
			wrong, err := NewClient(Testnet(), backend)
			if err != nil {
				t.Fatal(err)
			}
			construct := func(c *Client) []reflect.Value {
				f := reflect.ValueOf(c).MethodByName(tc.profile)
				if !f.IsValid() {
					t.Fatal("missing constructor", tc.profile)
				}
				return f.Call([]reflect.Value{reflect.ValueOf(testAddress)})
			}
			if r := construct(wrong); r[1].IsNil() || backend.calls != 0 {
				t.Fatal("cross-network constructor reached RPC")
			}
			backend.chain = big.NewInt(75)
			correct, err := NewClient(Mainnet(), backend)
			if err != nil {
				t.Fatal(err)
			}
			if r := construct(correct); !r[1].IsNil() {
				t.Fatal(r[1].Interface())
			}
			r := construct(&Client{})
			if !r[1].IsNil() {
				t.Fatal(r[1].Interface())
			}
			contract := r[0].Elem().FieldByName("Contract").Interface().(*Contract)
			a, err := loadABI(tc.profile)
			if err != nil {
				t.Fatal(err)
			}
			m, ok := a.Methods[tc.method]
			if !ok {
				t.Fatal("missing representative action")
			}
			args := make([]any, len(m.Inputs))
			for i, arg := range m.Inputs {
				args[i] = abiFixture(arg.Type.GetType()).Interface()
			}
			call, err := contract.Build(m.Sig, nil, args...)
			if err != nil {
				t.Fatal(err)
			}
			backend.chain = big.NewInt(202020)
			if p, err := wrong.Prepare(context.Background(), testAddress, call, TxOptions{}); err == nil || p != nil || backend.calls != 0 {
				t.Fatal("offline-built mainnet call reached cross-network RPC", err)
			}
		})
	}
}

func TestMainnetNFTCreationExactArguments(t *testing.T) {
	client := &Client{}
	f := reflect.ValueOf(client).MethodByName("MainnetNFTCenter20261103")
	if !f.IsValid() {
		t.Fatal("mainnet NFT constructor missing")
	}
	r := f.Call([]reflect.Value{reflect.ValueOf(testAddress)})
	if !r[1].IsNil() {
		t.Fatal(r[1].Interface())
	}
	a, err := loadABI("MainnetNFTCenter20261103")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CreateDRC721", "CreateDRC1155", "CreateDRC721Reserveless", "CreateDRC1155Reserveless"} {
		t.Run(name, func(t *testing.T) {
			build := r[0].MethodByName("Build" + name)
			if !build.IsValid() {
				t.Fatal("missing NFT builder", name)
			}
			if build.Type().NumIn() != 1 || build.Type().In(0).Kind() != reflect.Struct {
				t.Fatal("unreviewed NFT create signature")
			}
			meta := reflect.New(build.Type().In(0)).Elem()
			for i := 0; i < meta.NumField(); i++ {
				v := meta.Field(i)
				switch v.Kind() {
				case reflect.Array:
					v.Set(reflect.ValueOf(common.HexToAddress("0x123456")))
				case reflect.String:
					v.SetString([]string{"unused", "SYM-distinct", "Name-distinct", "ipfs://distinct"}[i])
				case reflect.Bool:
					v.SetBool(i%2 == 0)
				default:
					t.Fatal("unexpected NFT metadata field")
				}
			}
			out := build.Call([]reflect.Value{meta})
			if !out[1].IsNil() {
				t.Fatal(out[1].Interface())
			}
			call := out[0].Interface().(Call)
			m, err := a.MethodById(call.Data[:4])
			if err != nil {
				t.Fatal(err)
			}
			if m.RawName != "c"+name[1:] || call.To != testAddress || call.Profile != "MainnetNFTCenter20261103" {
				t.Fatal("incorrect route/selector")
			}
			values, err := m.Inputs.Unpack(call.Data[4:])
			if err != nil {
				t.Fatal(err)
			}
			actual := reflect.ValueOf(values[0])
			for i := 0; i < meta.NumField(); i++ {
				if !reflect.DeepEqual(meta.Field(i).Interface(), actual.Field(i).Interface()) {
					t.Fatalf("metadata field %d reordered", i)
				}
			}
		})
	}
}

func TestMainnetTokenCapabilitiesAreNotInvented(t *testing.T) {
	a, err := loadABI("MainnetReserveToken20261103")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"updateMinTotalSupply", "updateMaxTotalSupply"} {
		if _, ok := a.Methods[name]; !ok {
			t.Fatal("missing reserve supply control", name)
		}
	}
	for _, name := range []string{"mint", "setMintable"} {
		if _, ok := a.Methods[name]; ok {
			t.Fatal("unsupported reserve method invented", name)
		}
	}
	raw, err := os.ReadFile("docs/deployments/mainnet-2026-11-03-token/reserveless.abi.json")
	if err != nil {
		t.Fatal(err)
	}
	unbound, err := abi.JSON(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := unbound.Methods["mint"]; !ok {
		t.Fatal("official unbound reserveless mint missing")
	}
	if _, ok := unbound.Methods["setMintable"]; ok {
		t.Fatal("unbound reserveless setMintable invented")
	}
	if _, err := loadABI("MainnetReservelessToken20261103"); err == nil {
		t.Fatal("unbound reserveless interface promoted without proxy evidence")
	}
}

func TestMainnetReferenceDoesNotInventMint(t *testing.T) {
	c, e := (&Client{}).MainnetReserveToken20261103(testAddress)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.BuildReferenceOperation("updateTokenIdentity", nil, "identity"); e != nil {
		t.Fatal(e)
	}
	if _, e = c.BuildReferenceOperation("mintTokenReserveless", nil, testAddress, big.NewInt(1)); e == nil {
		t.Fatal("reserve profile admitted reserveless mint")
	}
}
