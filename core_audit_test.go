package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

type canceledReadRPC struct {
	*fakeRPC
	cancel context.CancelFunc
}

func (r *canceledReadRPC) fail() error {
	r.cancel()
	return errors.New("transport error with private endpoint")
}
func (r *canceledReadRPC) BlockNumber(context.Context) (uint64, error) { return 0, r.fail() }
func (r *canceledReadRPC) HeaderByNumber(context.Context, *big.Int) (*types.Header, error) {
	return nil, r.fail()
}
func (r *canceledReadRPC) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return nil, r.fail()
}
func (r *canceledReadRPC) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, r.fail()
}
func (r *canceledReadRPC) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, r.fail()
}
func (r *canceledReadRPC) SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, r.fail()
}

func TestReadRPCPreservesInFlightCancellation(t *testing.T) {
	cases := map[string]func(context.Context, *Client) error{
		"block":  func(ctx context.Context, c *Client) error { _, e := c.BlockNumber(ctx); return e },
		"header": func(ctx context.Context, c *Client) error { _, e := c.Header(ctx, nil); return e },
		"code":   func(ctx context.Context, c *Client) error { _, e := c.ContractCode(ctx, testAddress, nil); return e },
		"logs":   func(ctx context.Context, c *Client) error { _, e := c.Logs(ctx, ethereum.FilterQuery{}); return e },
		"log subscription": func(ctx context.Context, c *Client) error {
			_, e := c.SubscribeLogs(ctx, ethereum.FilterQuery{}, make(chan types.Log))
			return e
		},
		"head subscription": func(ctx context.Context, c *Client) error {
			_, e := c.SubscribeHeads(ctx, make(chan *types.Header))
			return e
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, e := NewClient(Testnet(), &canceledReadRPC{newFake(), cancel})
			if e != nil {
				t.Fatal(e)
			}
			if e = run(ctx, c); !errors.Is(e, context.Canceled) {
				t.Fatalf("lost cancellation: %v", e)
			}
		})
	}
}

func TestExhaustedNonceRejectedBeforeSigningOrBroadcast(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	nonce := ^uint64(0)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	for _, explicit := range []bool{true, false} {
		opts := TxOptions{}
		if explicit {
			opts.Nonce = &nonce
		} else {
			f.nonce = nonce
		}
		if _, e := c.Prepare(context.Background(), testAddress, call, opts); e == nil {
			t.Fatal("exhausted nonce prepared")
		}
	}
	s := testSigner(t, 1)
	tx, e := s.SignTransaction(context.Background(), types.NewTx(&types.LegacyTx{Nonce: nonce, To: &testAddress, Gas: 21000, GasPrice: big.NewInt(1), Value: big.NewInt(1)}), c.ChainID())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Broadcast(context.Background(), tx); e == nil {
		t.Fatal("exhausted nonce broadcast")
	}
	if len(f.sent) != 0 {
		t.Fatal("invalid nonce reached backend")
	}
	nonce--
	if _, e = c.Prepare(context.Background(), testAddress, call, TxOptions{Nonce: &nonce}); e != nil {
		t.Fatalf("last usable nonce rejected: %v", e)
	}
}
