package decimal

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"sort"
	"testing"
)

func TestDelegationDecodedLogsOwnReceiptBuffers(t *testing.T) {
	a, _ := loadABI("Delegation")
	e := a.Events["StakeUpdated"]
	var args []any
	for _, arg := range e.Inputs.NonIndexed() {
		args = append(args, delegationEventFixtureValue(arg.Type))
	}
	data, err := e.Inputs.NonIndexed().Pack(args...)
	if err != nil {
		t.Fatal(err)
	}
	log := &types.Log{Address: testAddress, Topics: []common.Hash{e.ID}, Data: data}
	before := append([]byte(nil), data...)
	c := &Contract{Address: testAddress, Profile: "Delegation"}
	rows, err := c.DecodeDelegationReceipt(&types.Receipt{Logs: []*types.Log{log, log}})
	if err != nil || len(rows) != 2 {
		t.Fatal(err)
	}
	rows[0].Log.Data[0] ^= 255
	rows[0].Log.Topics[0][0] ^= 255
	if !bytes.Equal(log.Data, before) || log.Topics[0] != e.ID || !bytes.Equal(rows[1].Log.Data, before) || rows[1].Log.Topics[0] != e.ID {
		t.Fatal("result aliases receipt or sibling result")
	}
	log.Data[1] ^= 255
	log.Topics[0][1] ^= 255
	if !bytes.Equal(rows[1].Log.Data, before) || rows[1].Log.Topics[0] != e.ID {
		t.Fatal("source mutation changes decoded event")
	}
}

func FuzzDelegationStakeDecoding(f *testing.F) {
	old, _ := loadABI("Delegation")
	current, _ := delegationUpgradeABI()
	names := []string{"getStake", "getStakes", "getHoldStake", "getHoldStakeDEL", "getFrozenStake", "getFrozenStakes"}
	for i, name := range names {
		for _, a := range []abi.ABI{old, current} {
			m := a.Methods[name]
			vals := make([]any, len(m.Outputs))
			for j, arg := range m.Outputs {
				vals[j] = stakeCompatFixture(arg.Type.GetType()).Interface()
			}
			b, err := m.Outputs.Pack(vals...)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(uint8(i), b)
		}
	}
	f.Add(uint8(0), []byte{})
	f.Add(uint8(1), bytes.Repeat([]byte{255}, 96))
	f.Fuzz(func(t *testing.T, index uint8, b []byte) {
		if len(b) > 16384 {
			return
		}
		before := append([]byte(nil), b...)
		name := names[int(index)%len(names)]
		vals, err := unpackDelegationRead("Delegation", old.Methods[name], b)
		if !bytes.Equal(before, b) {
			t.Fatal("decoder changed input")
		}
		if err != nil {
			return
		}
		if len(vals) != 1 {
			t.Fatal("unexpected output cardinality")
		}
		valid := false
		for _, a := range []abi.ABI{old, current} {
			m := a.Methods[name]
			v, e := m.Outputs.Unpack(b)
			if e == nil {
				enc, e := m.Outputs.Pack(v...)
				if e == nil && bytes.Equal(enc, b) {
					valid = true
				}
			}
		}
		if !valid {
			t.Fatal("accepted response is neither canonical layout")
		}
	})
}

func FuzzDelegationEventsAndReceipts(f *testing.F) {
	old, _ := loadABI("Delegation")
	current, _ := delegationUpgradeABI()
	var events []abi.Event
	for _, a := range []abi.ABI{old, current} {
		var names []string
		for name := range a.Events {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			events = append(events, a.Events[name])
		}
	}
	for i, e := range events {
		var args []any
		var indexed []byte
		for _, arg := range e.Inputs {
			v := delegationEventFixtureValue(arg.Type)
			if arg.Indexed {
				b, err := (abi.Arguments{{Type: arg.Type}}).Pack(v)
				if err != nil {
					f.Fatal(err)
				}
				indexed = append(indexed, b...)
			} else {
				args = append(args, v)
			}
		}
		data, err := e.Inputs.NonIndexed().Pack(args...)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(uint8(i), data, indexed)
	}
	c := &Contract{Address: testAddress, Profile: "Delegation"}
	f.Fuzz(func(t *testing.T, index uint8, data, indexed []byte) {
		if len(data) > 16384 || len(indexed) > 160 {
			return
		}
		e := events[int(index)%len(events)]
		topics := []common.Hash{e.ID}
		for len(indexed) > 0 {
			n := 32
			if n > len(indexed) {
				n = len(indexed)
			}
			topics = append(topics, common.BytesToHash(indexed[:n]))
			indexed = indexed[n:]
		}
		log := &types.Log{Address: testAddress, Topics: topics, Data: data}
		before := append([]byte(nil), data...)
		event, err := c.DecodeLog(*log)
		rows, receiptErr := c.DecodeDelegationReceipt(&types.Receipt{Logs: []*types.Log{nil, log, log}})
		if (err == nil) != (receiptErr == nil) {
			t.Fatal("single/receipt disagreement")
		}
		if err == nil {
			if len(rows) != 2 || rows[0].Signature != event.Signature || rows[1].Signature != event.Signature {
				t.Fatal("duplicate or identity lost")
			}
			if len(event.Log.Data) > 0 {
				event.Log.Data[0] ^= 255
			}
			event.Log.Topics[0][0] ^= 255
			if !bytes.Equal(data, before) || log.Topics[0] != e.ID {
				t.Fatal("decoded event aliases source")
			}
		} else if rows != nil {
			t.Fatal("partial result after malformed event")
		}
		foreign := *log
		foreign.Address = common.HexToAddress("0x1234")
		rows, err = c.DecodeDelegationReceipt(&types.Receipt{Logs: []*types.Log{&foreign}})
		if err != nil || len(rows) != 0 {
			t.Fatal("foreign event decoded")
		}
	})
}
