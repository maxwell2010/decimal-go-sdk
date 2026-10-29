package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	decimal "github.com/maxwell2010/decimal-go-sdk"
)

func TestEveryCataloguedWriteBuilderHasExecutableExample(t *testing.T) {
	expected := map[string]bool{}
	for _, file := range []string{"contract-catalog.json", "deployment-catalog.json", "additional-deployment-catalog.json", "token-deployment-catalog.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "docs", file))
		if err != nil {
			t.Fatal(err)
		}
		var records []struct{ Profile, Signature, Method string }
		if err = json.Unmarshal(data, &records); err != nil {
			t.Fatal(err)
		}
		for _, row := range records {
			if strings.HasPrefix(row.Method, "Build") {
				key := row.Profile + ":" + row.Signature
				if expected[key] {
					t.Fatal("duplicate catalog", key)
				}
				expected[key] = true
			}
		}
	}
	calls, err := allCalls()
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range calls {
		key := call.Profile + ":" + call.Signature
		if !expected[key] {
			t.Fatal("unexpected/duplicate example", key)
		}
		delete(expected, key)
		raw, err := decimal.ABIJSON(call.Profile)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := abi.JSON(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		if len(call.Data) < 4 {
			t.Fatal("missing selector", key)
		}
		method, err := parsed.MethodById(call.Data[:4])
		if err != nil {
			t.Fatal(err)
		}
		if method.Sig != call.Signature || method.IsConstant() {
			t.Fatal("wrong write selector", key)
		}
		values, err := method.Inputs.Unpack(call.Data[4:])
		if err != nil {
			t.Fatal(key, err)
		}
		encoded, err := method.Inputs.Pack(values...)
		if err != nil || !bytes.Equal(encoded, call.Data[4:]) {
			t.Fatal("noncanonical argument encoding", key, err)
		}
		if !call.Contract || call.To != fixtureAddress || call.Value == nil || call.Value.Sign() < 0 {
			t.Fatal("invalid offline intent", key)
		}
		if method.StateMutability != "payable" && call.Value.Sign() != 0 {
			t.Fatal("nonpayable value", key)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("%d builders lack runnable examples: %v", len(expected), expected)
	}
}
