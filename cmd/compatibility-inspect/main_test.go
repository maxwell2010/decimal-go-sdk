package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStrictAddressWords(t *testing.T) {
	good := "0x" + strings.Repeat("00", 12) + strings.Repeat("11", 20)
	if addressResult(good) != "0x1111111111111111111111111111111111111111" {
		t.Fatal("valid address")
	}
	for _, s := range []string{"0x", "0x" + strings.Repeat("00", 32), "0x01" + strings.Repeat("00", 31), "0x" + strings.Repeat("11", 20)} {
		if addressResult(s) != "" {
			t.Fatal("invalid word accepted")
		}
	}
}
func TestHistoricalComparisonDoesNotCrossRoles(t *testing.T) {
	targets, expected, e := manifests("../../docs")
	if e != nil {
		t.Fatal(e)
	}
	if len(targets) != 2 {
		t.Fatal("token candidates", len(targets))
	}
	// The token manifest runtime_code_hash belongs to implementation, not proxy.
	for _, target := range targets {
		if len(expected[strings.ToLower(target.address)]) != 0 {
			t.Fatal("implementation hash mislabeled as proxy")
		}
	}
	r := reference{Address: "0x1111111111111111111111111111111111111111", CodeHash: "new"}
	compareReference(&r, map[string][]string{strings.ToLower(r.Address): {"old"}})
	if r.Matches == nil || *r.Matches {
		t.Fatal("upgrade mismatch lost")
	}
}
func TestReadSelectorsAndEmptyRuntime(t *testing.T) {
	if selector("nonces()") != "0x93757526" {
		t.Fatal(selector("nonces()"))
	}
	if _, _, e := codeHash("0x"); e == nil {
		t.Fatal("empty runtime accepted")
	}
}

func TestSnapshotRequiresExactRawIdentity(t *testing.T) {
	hash := "0x" + strings.Repeat("11", 32)
	cases := []struct {
		body  string
		valid bool
	}{
		{fmt.Sprintf(`{"number":"0x7","hash":%q}`, hash), true},
		{fmt.Sprintf(`{"number":"0x8","hash":%q}`, hash), false},
		{fmt.Sprintf(`{"hash":%q}`, hash), false},
		{fmt.Sprintf(`{"number":"0x10000000000000000","hash":%q}`, hash), false},
		{`{"number":"0x7","hash":"0x1"}`, false},
		{`{"number":"0x7","hash":"0x` + strings.Repeat("00", 32) + `"}`, false},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var q struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			_ = json.NewDecoder(r.Body).Decode(&q)
			result := `"0x31524"`
			if q.Method == "eth_getBlockByNumber" {
				result = tc.body
			} else if q.Method != "eth_chainId" {
				t.Error("unexpected method", q.Method)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, q.ID, result)
		}))
		raw, e := rpc.DialContext(context.Background(), server.URL)
		if e != nil {
			t.Fatal(e)
		}
		c, e := decimal.NewClient(decimal.Testnet(), ethclient.NewClient(raw))
		if e != nil {
			t.Fatal(e)
		}
		_, e = snapshotIdentity(context.Background(), c, "0x7")
		raw.Close()
		server.Close()
		if (e == nil) != tc.valid {
			t.Fatal("identity acceptance", tc.body, e)
		}
	}
	for _, tag := range []string{"-1", "0x10000000000000000", "0x00"} {
		if _, e := snapshotIdentity(context.Background(), nil, tag); e == nil {
			t.Fatal("invalid block tag")
		}
	}
}
