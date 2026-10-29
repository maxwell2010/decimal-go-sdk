package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type proxyRPC struct {
	*fakeRPC
	implementation common.Address
	changed        bool
	blocks         []uint64
	slot           []byte
}

func (p *proxyRPC) BlockNumber(context.Context) (uint64, error) { return 88, nil }
func (p *proxyRPC) StorageAt(_ context.Context, address common.Address, slot common.Hash, block *big.Int) ([]byte, error) {
	p.blocks = append(p.blocks, block.Uint64())
	if slot != implementationSlot {
		return nil, nil
	}
	if p.slot != nil {
		return p.slot, nil
	}
	return common.LeftPadBytes(p.implementation.Bytes(), 32), nil
}
func (p *proxyRPC) CodeAt(_ context.Context, address common.Address, block *big.Int) ([]byte, error) {
	p.blocks = append(p.blocks, block.Uint64())
	if p.changed {
		return []byte{99}, nil
	}
	if address == p.implementation {
		return []byte{2}, nil
	}
	return []byte{1}, nil
}
func TestProxyABIResolutionRetainsTargetAndPinnedEvidence(t *testing.T) {
	f := &proxyRPC{fakeRPC: newFake(), implementation: common.HexToAddress("0x2000000000000000000000000000000000000002")}
	c, e := NewClient(Testnet(), f)
	if e != nil {
		t.Fatal(e)
	}
	var queried string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queried = r.URL.Path
		_, _ = w.Write([]byte(`{"Result":{"abi":[{"type":"function","name":"pause","inputs":[],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"owner","inputs":[],"outputs":[{"type":"address"}],"stateMutability":"view"}]}}`))
	}))
	defer s.Close()
	api, _ := NewAPI(s.URL, nil)
	p, e := c.LoadProxyDeploymentContract(context.Background(), api, testAddress, 77)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasSuffix(queried, strings.ToLower(f.implementation.Hex())) || p.Evidence.Block != 77 {
		t.Fatal("wrong ABI provenance")
	}
	for _, b := range f.blocks {
		if b != 77 {
			t.Fatal("mixed-block evidence")
		}
	}
	call, e := p.Contract.Build("pause()", nil)
	if e != nil || call.To != testAddress {
		t.Fatal("implementation substituted for proxy")
	}
	p.Evidence.Implementation = common.Address{} // Display metadata is not the Call's guard.
	f.changed = true
	reads := 0
	f.call = func(_ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		reads++
		return common.LeftPadBytes(testAddress.Bytes(), 32), nil
	}
	if _, e = p.Contract.ReadAt(context.Background(), big.NewInt(88), "owner()"); e == nil || reads != 0 {
		t.Fatal("changed proxy read accepted")
	}
	if _, e = c.Prepare(context.Background(), testAddress, call, TxOptions{}); e == nil {
		t.Fatal("changed code accepted")
	}
	if len(f.sent) != 0 {
		t.Fatal("proxy loader broadcast")
	}
}
func TestProxySlotRejectsUnsupportedShapes(t *testing.T) {
	for _, slot := range [][]byte{{1}, make([]byte, 32), append([]byte{1}, make([]byte, 31)...)} {
		f := &proxyRPC{fakeRPC: newFake(), slot: slot}
		c, _ := NewClient(Testnet(), f)
		if _, e := c.proxyEvidence(context.Background(), testAddress, 5); e == nil {
			t.Fatal("invalid proxy slot accepted")
		}
	}
}
