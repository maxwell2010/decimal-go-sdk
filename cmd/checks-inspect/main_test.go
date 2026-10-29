package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
)

func TestRawHeaderRequiresExplicitNumberIncludingGenesis(t *testing.T) {
	hash := "0x" + strings.Repeat("12", 32)
	for _, tc := range []struct {
		name, payload string
		height        uint64
		valid         bool
	}{
		{"genesis", fmt.Sprintf(`{"number":"0x0","hash":%q}`, hash), 0, true},
		{"normal", fmt.Sprintf(`{"number":"0x7","hash":%q}`, hash), 7, true},
		{"missing-genesis-number", fmt.Sprintf(`{"hash":%q}`, hash), 0, false},
		{"null-genesis-number", fmt.Sprintf(`{"number":null,"hash":%q}`, hash), 0, false},
		{"wrong-height", fmt.Sprintf(`{"number":"0x8","hash":%q}`, hash), 7, false},
		{"overflow", fmt.Sprintf(`{"number":"0x10000000000000000","hash":%q}`, hash), 0, false},
		{"short-hash", `{"number":"0x0","hash":"0x12"}`, 0, false},
		{"zero-hash", fmt.Sprintf(`{"number":"0x0","hash":%q}`, "0x"+strings.Repeat("00", 32)), 0, false},
		{"missing-hash", `{"number":"0x0"}`, 0, false},
		{"null", `null`, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				if req.Method != "eth_getBlockByNumber" || len(req.Params) != 2 || string(req.Params[0]) != fmt.Sprintf(`"0x%x"`, tc.height) || string(req.Params[1]) != "false" {
					t.Error("unexpected read request", req)
					return
				}
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, tc.payload)
			}))
			defer server.Close()
			client, err := rpc.DialContext(context.Background(), server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			got, err := rawHeader(context.Background(), client, tc.height)
			if (err == nil) != tc.valid {
				t.Fatalf("got=%v error=%v", got, err)
			}
			if tc.valid && (got.Number == nil || uint64(*got.Number) != tc.height || got.Hash.Hex() != hash) {
				t.Fatal("identity changed", got)
			}
		})
	}
}
