package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// These tests exercise the actual HTTP JSON-RPC transport, not a Backend fake.
func resilienceClient(t *testing.T, method string, fault func(http.ResponseWriter, *http.Request, json.RawMessage), hits *atomic.Int32) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if req.Method == "eth_chainId" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x31524"}`, req.ID)
			return
		}
		if req.Method != method {
			t.Errorf("unexpected RPC %s", req.Method)
			http.Error(w, "unexpected", 500)
			return
		}
		hits.Add(1)
		fault(w, r, req.ID)
	}))
	t.Cleanup(server.Close)
	cfg := Testnet()
	cfg.RPCURL = server.URL + "/?token=fixture-endpoint-secret"
	c, err := Dial(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func resilienceFaults() map[string]func(http.ResponseWriter, *http.Request, json.RawMessage) {
	return map[string]func(http.ResponseWriter, *http.Request, json.RawMessage){
		"rate-limit": func(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "fixture-response-secret", 429)
		},
		"unavailable": func(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
			http.Error(w, "fixture-response-secret", 503)
		},
		"malformed-json": func(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
			fmt.Fprint(w, `{"fixture-response-secret":`)
		},
		"rpc-error": func(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"fixture-response-secret"}}`, id)
		},
		"connection-cut": func(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		},
	}
}

func TestRPCResilienceReadFailuresAreBoundedAndRedacted(t *testing.T) {
	for name, fault := range resilienceFaults() {
		t.Run(name, func(t *testing.T) {
			var hits atomic.Int32
			c := resilienceClient(t, "eth_getBalance", fault, &hits)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			got, err := c.Balance(ctx, testAddress)
			if err == nil || got != nil {
				t.Fatalf("failure returned data: %v %v", got, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("transport detail leaked", err)
			}
			if hits.Load() != 1 {
				t.Fatalf("unexpected retry count %d", hits.Load())
			}
		})
	}
}

func TestRPCResilienceDeadlineStopsPendingHTTP(t *testing.T) {
	var hits atomic.Int32
	c := resilienceClient(t, "eth_getBalance", func(w http.ResponseWriter, r *http.Request, id json.RawMessage) { <-r.Context().Done() }, &hits)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := c.Balance(ctx, testAddress)
	if !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Fatalf("deadline lost: %v %v", got, err)
	}
	if hits.Load() != 1 {
		t.Fatal("missing request or unexpected retries", hits.Load())
	}
}

func TestRPCResilienceBroadcastFailuresNeverResubmit(t *testing.T) {
	signer := testSigner(t, 1)
	unsigned := types.NewTx(&types.LegacyTx{Nonce: 0, GasPrice: big.NewInt(10), Gas: 21000, To: &testAddress, Value: big.NewInt(1)})
	signed, err := signer.SignTransaction(context.Background(), unsigned, big.NewInt(202020))
	if err != nil {
		t.Fatal(err)
	}
	for name, fault := range resilienceFaults() {
		t.Run(name, func(t *testing.T) {
			var hits atomic.Int32
			c := resilienceClient(t, "eth_sendRawTransaction", fault, &hits)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			hash, err := c.Broadcast(ctx, signed)
			var unknown *SubmissionUnknownError
			if hash != signed.Hash() || !errors.As(err, &unknown) || unknown.Hash != signed.Hash() {
				t.Fatalf("ambiguous submission not preserved: %s %v", hash, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("transport detail leaked", err)
			}
			if hits.Load() != 1 {
				t.Fatalf("submission retried %d times", hits.Load())
			}
		})
	}
}

func TestRPCResilienceReceiptIdentityAndShape(t *testing.T) {
	hash := common.HexToHash("0x1234")
	for _, name := range []string{"wrong-transaction", "invalid-height", "invalid-json-type"} {
		t.Run(name, func(t *testing.T) {
			var hits atomic.Int32
			c := resilienceClient(t, "eth_getTransactionReceipt", func(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
				if name == "invalid-json-type" {
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"fixture-response-secret"}`, id)
					return
				}
				receipt := &types.Receipt{Type: 0, Status: 1, TxHash: hash, BlockHash: common.HexToHash("0x4321"), BlockNumber: big.NewInt(10), Logs: []*types.Log{}}
				if name == "wrong-transaction" {
					receipt.TxHash = common.HexToHash("0x9999")
				} else {
					receipt.BlockNumber.SetInt64(0)
				}
				data, err := json.Marshal(receipt)
				if err != nil {
					t.Error(err)
					return
				}
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, id, data)
			}, &hits)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			got, err := c.WaitReceipt(ctx, hash, time.Millisecond)
			if err == nil || got != nil {
				t.Fatalf("malformed receipt accepted: %v %v", got, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("response details leaked", err)
			}
			if hits.Load() != 1 {
				t.Fatal("bad receipt caused retry", hits.Load())
			}
		})
	}
}

func TestRPCResilienceReceiptTransportErrorsDoNotPollOrResubmit(t *testing.T) {
	for name, fault := range resilienceFaults() {
		t.Run(name, func(t *testing.T) {
			var hits atomic.Int32
			c := resilienceClient(t, "eth_getTransactionReceipt", fault, &hits)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			mined, err := c.WaitReceipt(ctx, common.HexToHash("0x1234"), time.Millisecond)
			if err == nil || mined != nil {
				t.Fatalf("transport failure returned receipt: %v %v", mined, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("transport detail leaked", err)
			}
			if hits.Load() != 1 {
				t.Fatal("transport error unexpectedly polled", hits.Load())
			}
		})
	}
}

// Identity changes must remain observable; do not cache a number-to-hash pair
// or substitute Ethereum header hashing when the same height is reorganized.
func TestRPCResilienceBlockIdentityChangesAtSameHeight(t *testing.T) {
	var reads atomic.Int32
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
		result := `"0x31524"`
		switch req.Method {
		case "eth_chainId":
		case "eth_getBlockByNumber":
			if len(req.Params) != 2 || string(req.Params[0]) != `"0x7"` || string(req.Params[1]) != "false" {
				t.Error("unexpected snapshot request")
				return
			}
			hash := common.HexToHash("0x11")
			if reads.Add(1) > 1 {
				hash = common.HexToHash("0x22")
			}
			result = fmt.Sprintf(`{"number":"0x7","hash":%q}`, hash.Hex())
		default:
			t.Error("unexpected RPC", req.Method)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
	}))
	defer server.Close()
	cfg := Testnet()
	cfg.RPCURL = server.URL
	client, err := Dial(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	before, err := client.BlockIdentity(context.Background(), big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	after, err := client.BlockIdentity(context.Background(), big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	if before.Number != 7 || after.Number != 7 || before.Hash != common.HexToHash("0x11") || after.Hash != common.HexToHash("0x22") || reads.Load() != 2 {
		t.Fatal("identity change hidden", before, after, reads.Load())
	}
}
