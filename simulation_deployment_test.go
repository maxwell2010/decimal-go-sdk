package decimal

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestBoundSimulationChecksDeploymentAndPinsBlock(t *testing.T) {
	ctx := context.Background()
	f := &proxyRPC{fakeRPC: newFake(), implementation: common.HexToAddress("0x2000000000000000000000000000000000000002")}
	c, _ := NewClient(Testnet(), f)
	p, e := c.BindProxyABI(ctx, testAddress, 77, json.RawMessage(`[{"type":"function","name":"pause","inputs":[],"outputs":[],"stateMutability":"nonpayable"}]`))
	if e != nil {
		t.Fatal(e)
	}
	call, e := p.Contract.Build("pause()", nil)
	if e != nil {
		t.Fatal(e)
	}
	reads := 0
	f.call = func(_ ethereum.CallMsg, b *big.Int) ([]byte, error) {
		reads++
		if b == nil || b.Uint64() != 88 {
			t.Fatal("simulation not pinned")
		}
		return nil, nil
	}
	for _, run := range []func(Call) ([]byte, error){func(x Call) ([]byte, error) { return c.Simulate(ctx, testAddress, x, nil) }, func(x Call) ([]byte, error) { return p.Contract.Simulate(ctx, testAddress, x, nil) }} {
		if _, e = run(call); e != nil {
			t.Fatal(e)
		}
		f.changed = true
		before := reads
		if _, e = run(call); e == nil || reads != before {
			t.Fatal("upgrade reached simulation")
		}
		f.changed = false
	}
	// Contract evidence is enforced even when the caller strips Call metadata.
	stripped := call
	stripped.deployment = nil
	f.changed = true
	if _, e = p.Contract.Simulate(ctx, testAddress, stripped, nil); e == nil {
		t.Fatal("contract evidence bypassed")
	}
	f.changed = false
	changed := call
	changed.To = f.implementation
	if _, e = c.Simulate(ctx, testAddress, changed, nil); e == nil {
		t.Fatal("target mismatch accepted")
	}
	if _, e = c.Simulate(ctx, testAddress, call, big.NewInt(-1)); e == nil {
		t.Fatal("negative snapshot")
	}
	if len(f.sent) != 0 {
		t.Fatal("simulation broadcast")
	}
}
