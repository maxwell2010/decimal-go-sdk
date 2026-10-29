package decimal

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

type beaconRPC struct {
	*fakeRPC
	beacon, implementation     common.Address
	direct, changed, dependent bool
	heights                    []uint64
}

func (f *beaconRPC) BlockNumber(context.Context) (uint64, error) { return 90, nil }
func (f *beaconRPC) StorageAt(_ context.Context, _ common.Address, slot common.Hash, height *big.Int) ([]byte, error) {
	f.heights = append(f.heights, height.Uint64())
	if slot == implementationSlot {
		if f.direct {
			return common.LeftPadBytes(f.implementation.Bytes(), 32), nil
		}
		return make([]byte, 32), nil
	}
	return common.LeftPadBytes(f.beacon.Bytes(), 32), nil
}
func (f *beaconRPC) CodeAt(_ context.Context, address common.Address, height *big.Int) ([]byte, error) {
	if height != nil {
		f.heights = append(f.heights, height.Uint64())
	}
	if f.changed && address == f.beacon {
		return []byte{99}, nil
	}
	return append([]byte{1}, address.Bytes()...), nil
}
func (f *beaconRPC) CallContract(_ context.Context, msg ethereum.CallMsg, height *big.Int) ([]byte, error) {
	f.heights = append(f.heights, height.Uint64())
	if f.dependent && msg.From == (common.Address{}) {
		return make([]byte, 32), nil
	}
	return common.LeftPadBytes(f.implementation.Bytes(), 32), nil
}
func TestBeaconBindingChecksSlotCallerAndUpgrade(t *testing.T) {
	f := &beaconRPC{fakeRPC: newFake(), beacon: common.HexToAddress("0x2000000000000000000000000000000000000002"), implementation: common.HexToAddress("0x3000000000000000000000000000000000000003")}
	c, _ := NewClient(Testnet(), f)
	raw := json.RawMessage(`[{"type":"function","name":"pause","inputs":[],"outputs":[],"stateMutability":"nonpayable"}]`)
	p, e := c.BindBeaconABI(context.Background(), testAddress, 80, raw)
	if e != nil {
		t.Fatal(e)
	}
	if p.Evidence.Beacon != f.beacon || p.Evidence.Implementation != f.implementation {
		t.Fatal("wrong beacon chain")
	}
	for _, height := range f.heights {
		if height != 80 {
			t.Fatal("mixed-block beacon evidence")
		}
	}
	if e = p.VerifyAt(context.Background(), 81); e != nil {
		t.Fatal(e)
	}
	call, e := p.Contract.Build("pause()", nil)
	if e != nil || call.To != testAddress {
		t.Fatal("beacon/implementation used as transaction target")
	}
	f.changed = true
	if _, e = c.Prepare(context.Background(), testAddress, call, TxOptions{}); e == nil {
		t.Fatal("changed beacon accepted")
	}
	f.changed = false
	f.dependent = true
	if _, e = c.BindBeaconABI(context.Background(), testAddress, 82, raw); e == nil {
		t.Fatal("caller-dependent beacon accepted")
	}
	f.dependent = false
	f.direct = true
	if _, e = c.BindBeaconABI(context.Background(), testAddress, 82, raw); e == nil {
		t.Fatal("direct implementation treated as beacon")
	}
	if len(f.sent) != 0 {
		t.Fatal("beacon inspection broadcast")
	}
}
