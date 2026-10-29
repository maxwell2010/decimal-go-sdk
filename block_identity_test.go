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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestBlockIdentityUsesRPCProvidedHashAndStrictShape(t *testing.T) {
	hash := common.HexToHash("0x" + strings.Repeat("12", 32))
	if (&types.Header{Number: big.NewInt(7)}).Hash() == hash {
		t.Fatal("fixture must distinguish native identity")
	}
	cases := []struct {
		name, payload string
		valid         bool
	}{
		{"native-hash", fmt.Sprintf(`{"number":"0x7","hash":%q}`, hash.Hex()), true},
		{"null", `null`, false}, {"missing-hash", `{"number":"0x7"}`, false}, {"missing-number", fmt.Sprintf(`{"hash":%q}`, hash.Hex()), false},
		{"zero", `{"number":"0x7","hash":"0x` + strings.Repeat("00", 32) + `"}`, false},
		{"short-hash", `{"number":"0x7","hash":"0x12"}`, false},
		{"malformed-hash", `{"number":"0x7","hash":"0x` + strings.Repeat("zz", 32) + `"}`, false},
		{"negative", fmt.Sprintf(`{"number":"-0x1","hash":%q}`, hash.Hex()), false},
		{"overflow", fmt.Sprintf(`{"number":"0x10000000000000000","hash":%q}`, hash.Hex()), false},
		{"different-height", fmt.Sprintf(`{"number":"0x8","hash":%q}`, hash.Hex()), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requested := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
					t.Error(e)
					return
				}
				result := `"0x31524"`
				switch req.Method {
				case "eth_chainId":
				case "eth_getBlockByNumber":
					if len(req.Params) != 2 {
						t.Error("bad params")
						return
					}
					var tag string
					var full bool
					_ = json.Unmarshal(req.Params[0], &tag)
					_ = json.Unmarshal(req.Params[1], &full)
					if full {
						t.Error("full transactions requested")
					}
					requested <- tag
					result = tc.payload
				default:
					t.Error("unexpected RPC method", req.Method)
					return
				}
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
			}))
			defer server.Close()
			cfg := Testnet()
			cfg.RPCURL = server.URL
			c, e := Dial(context.Background(), cfg)
			if e != nil {
				t.Fatal(e)
			}
			defer c.Close()
			got, e := c.BlockIdentity(context.Background(), big.NewInt(7))
			if (e == nil) != tc.valid {
				t.Fatal("acceptance", got, e)
			}
			if tc.valid && (got.Number != 7 || got.Hash != hash) {
				t.Fatal("RPC identity replaced", got)
			}
			if tag := <-requested; tag != "0x7" {
				t.Fatal("wrong snapshot", tag)
			}
			if tc.valid {
				got, e = c.BlockIdentity(context.Background(), nil)
				if e != nil || got.Number != 7 || got.Hash != hash {
					t.Fatal("latest", e)
				}
				if tag := <-requested; tag != "latest" {
					t.Fatal(tag)
				}
			}
		})
	}
}

type identityBackend struct {
	*fakeRPC
	read   func(context.Context, *big.Int) (BlockIdentity, error)
	before func()
}

func (b *identityBackend) ChainID(ctx context.Context) (*big.Int, error) {
	if b.before != nil {
		b.before()
	}
	return b.fakeRPC.ChainID(ctx)
}
func (b *identityBackend) BlockIdentity(ctx context.Context, n *big.Int) (BlockIdentity, error) {
	return b.read(ctx, n)
}
func TestBlockIdentityCustomAdapterSnapshotsAndGuards(t *testing.T) {
	hash := common.HexToHash("0x" + strings.Repeat("34", 32))
	number := big.NewInt(7)
	b := &identityBackend{fakeRPC: newFake()}
	b.before = func() { number.SetInt64(8) }
	b.read = func(_ context.Context, n *big.Int) (BlockIdentity, error) {
		if n == nil || n.Uint64() != 7 {
			t.Fatal("caller input not snapshotted")
		}
		n.SetInt64(99)
		return BlockIdentity{Number: 7, Hash: hash}, nil
	}
	c, _ := NewClient(Testnet(), b)
	got, e := c.BlockIdentity(context.Background(), number)
	if e != nil || got.Number != 7 {
		t.Fatal("validation used mutated input", e)
	}
	b.before = nil
	beforeInvalid := b.fakeRPC.calls
	for _, n := range []*big.Int{big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 64)} {
		b.read = func(context.Context, *big.Int) (BlockIdentity, error) {
			t.Fatal("invalid height reached adapter")
			return BlockIdentity{}, nil
		}
		if _, e = c.BlockIdentity(context.Background(), n); e == nil {
			t.Fatal("invalid height")
		}
	}
	if b.fakeRPC.calls != beforeInvalid {
		t.Fatal("invalid height reached chain RPC")
	}
	b.read = func(context.Context, *big.Int) (BlockIdentity, error) {
		return BlockIdentity{Number: 8, Hash: hash}, nil
	}
	if _, e = c.BlockIdentity(context.Background(), big.NewInt(7)); e == nil {
		t.Fatal("custom mismatch")
	}
	b.read = func(context.Context, *big.Int) (BlockIdentity, error) { return BlockIdentity{Number: 7}, nil }
	if _, e = c.BlockIdentity(context.Background(), nil); e == nil {
		t.Fatal("custom zero hash")
	}
	b.read = func(context.Context, *big.Int) (BlockIdentity, error) {
		return BlockIdentity{}, errors.New("private endpoint credential")
	}
	if _, e = c.BlockIdentity(context.Background(), nil); e == nil || strings.Contains(e.Error(), "credential") {
		t.Fatal("backend detail leaked", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = c.BlockIdentity(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	b.read = func(context.Context, *big.Int) (BlockIdentity, error) {
		cancel()
		return BlockIdentity{Number: 7, Hash: hash}, nil
	}
	if _, e = c.BlockIdentity(ctx, nil); !errors.Is(e, context.Canceled) {
		t.Fatal("inflight cancellation", e)
	}
	unsupported, _ := NewClient(Testnet(), newFake())
	if _, e = unsupported.BlockIdentity(context.Background(), nil); e == nil {
		t.Fatal("unsupported backend silently hashed a header")
	}
	if _, e = c.BlockIdentity(nil, nil); e == nil {
		t.Fatal("nil context")
	}
	var absent *Client
	if _, e = absent.BlockIdentity(context.Background(), nil); e == nil {
		t.Fatal("nil client")
	}
}
