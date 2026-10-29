package decimal

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"os"
	"testing"
)

func creationFixture(t *testing.T) types.Log {
	t.Helper()
	a, e := loadABI("Checks")
	if e != nil {
		t.Fatal(e)
	}
	b := make([]byte, 128)
	b[0] = 1
	b[32] = 2
	b[95] = 3
	b[127] = 2
	return types.Log{Address: testAddress, Topics: []common.Hash{a.Events["CreatedCheck"].ID}, Data: b, BlockNumber: 50, BlockHash: common.HexToHash("0x50"), TxHash: common.HexToHash("0x51"), TxIndex: 2, Index: 3}
}
func TestChecksCreationDecodeStrictAndSnapshot(t *testing.T) {
	l := creationFixture(t)
	row, e := DecodeChecksCreationLog(testAddress, l)
	if e != nil || row.CheckHash[0] != 1 || row.DetailsHash[0] != 2 || row.Signer[19] != 3 || row.Status != 2 {
		t.Fatal(row, e)
	}
	l.Data[0] = 9
	l.Topics[0] = common.Hash{}
	if row.CheckHash[0] != 1 || row.Log.Data[0] != 1 || row.Log.Topics[0] == (common.Hash{}) {
		t.Fatal("retained caller data")
	}
	for _, mut := range []func(*types.Log){func(x *types.Log) { x.Removed = true }, func(x *types.Log) { x.Address = common.Address{} }, func(x *types.Log) { x.Topics = append(x.Topics, common.Hash{}) }, func(x *types.Log) { x.Data = append(x.Data, 0) }, func(x *types.Log) { x.Data[64] = 1 }, func(x *types.Log) { x.Data[96] = 1 }, func(x *types.Log) { x.BlockHash = common.Hash{} }} {
		l := creationFixture(t)
		mut(&l)
		if _, e := DecodeChecksCreationLog(testAddress, l); e == nil {
			t.Fatal("malformed log accepted")
		}
	}
}
func TestChecksReceiptAndDetails(t *testing.T) {
	l := creationFixture(t)
	r := &types.Receipt{Status: 1, BlockNumber: big.NewInt(50), BlockHash: l.BlockHash, TxHash: l.TxHash, TransactionIndex: 2, Logs: []*types.Log{&l}}
	rows, e := ChecksCreationsFromReceipt(testAddress, r)
	if e != nil || len(rows) != 1 {
		t.Fatal(e)
	}
	r.TxHash = common.HexToHash("0x99")
	if _, e = ChecksCreationsFromReceipt(testAddress, r); e == nil {
		t.Fatal("receipt mismatch accepted")
	}
	r.TxHash = l.TxHash
	r.Status = 0
	if _, e = ChecksCreationsFromReceipt(testAddress, r); e == nil {
		t.Fatal("failed receipt accepted")
	}
	a, _ := loadABI("Checks")
	d := creationFixture(t)
	d.Topics = []common.Hash{a.Events["CreatedCheckDetails"].ID}
	d.Data = make([]byte, 192)
	d.Data[0] = 2
	d.Data[63] = 1
	d.Data[95] = 9
	d.Data[127] = 20
	d.Data[159] = 4
	d.Data[191] = 5
	details, e := DecodeChecksCreationDetailsLog(testAddress, d)
	if e != nil || details.Type != 1 || details.Amount.Int64() != 9 || details.DueBlock.Int64() != 20 || details.Token[19] != 4 || details.Creator[19] != 5 {
		t.Fatal(details, e)
	}
	d.Data[128] = 1
	if _, e = DecodeChecksCreationDetailsLog(testAddress, d); e == nil {
		t.Fatal("padding")
	}
}

type historyRPC struct {
	*fakeRPC
	logs  []types.Log
	query ethereum.FilterQuery
	calls int
}

func (r *historyRPC) FilterLogs(_ context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	r.calls++
	r.query = q
	return r.logs, nil
}
func TestChecksHistoryBoundsAndOrder(t *testing.T) {
	l := creationFixture(t)
	second := creationFixture(t)
	second.Index = 4
	second.Data = append([]byte{}, second.Data...)
	second.Data[0] = 9
	r := &historyRPC{fakeRPC: newFake(), logs: []types.Log{second, l}}
	c, err := NewClient(Testnet(), r)
	if err != nil {
		t.Fatal(err)
	}
	rows, e := c.ChecksCreationHistory(context.Background(), testAddress, 49, 50)
	if e != nil || len(rows) != 2 || rows[0].Log.Index != 3 {
		t.Fatal(rows, e)
	}
	if r.query.FromBlock.Uint64() != 49 || r.query.ToBlock.Uint64() != 50 || len(r.query.Topics) != 1 || !bytes.Equal(r.query.Addresses[0][:], testAddress[:]) {
		t.Fatal("query")
	}
	before := r.calls
	if _, e = c.ChecksCreationHistory(context.Background(), testAddress, 0, 2000); e == nil || r.calls != before {
		t.Fatal("unbounded query")
	}
	r.logs = []types.Log{l, l}
	if _, e = c.ChecksCreationHistory(context.Background(), testAddress, 49, 50); e == nil {
		t.Fatal("duplicates")
	}
	r.logs = []types.Log{l}
	if _, e = c.ChecksCreationHistory(context.Background(), testAddress, 51, 52); e == nil {
		t.Fatal("out of range")
	}
}

func TestChecksSavedOfficialReceipt(t *testing.T) {
	raw, e := os.ReadFile("docs/compatibility/checks-testnet-receipts.json")
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		Address  string
		Receipts []*types.Receipt
	}
	if e = json.Unmarshal(raw, &v); e != nil {
		t.Fatal(e)
	}
	if len(v.Receipts) != 1 {
		t.Fatal("missing official reference receipt")
	}
	a, e := ParseAddress(v.Address)
	if e != nil {
		t.Fatal(e)
	}
	rows, e := ChecksCreationsFromReceipt(a, v.Receipts[0])
	if e != nil || len(rows) != 1 || rows[0].CheckHash != common.HexToHash("0x58d605b116ba5b5833da55ad0b89b2b20564f982d4afe2e586bfc8256fca1d4c") {
		t.Fatal(rows, e)
	}
}

func TestChecksHistoryRejectsMissingReadBackendAndIncoherentTransactions(t *testing.T) {
	var missing *Client
	for _, c := range []*Client{missing, new(Client)} {
		if _, e := c.ChecksCreationHistory(context.Background(), testAddress, 49, 50); e == nil {
			t.Fatal("missing backend")
		}
		if _, e := c.ChecksCreationReceipt(context.Background(), testAddress, common.HexToHash("0x1")); e == nil {
			t.Fatal("missing receipt backend")
		}
	}
	c, _ := NewClient(Testnet(), &historyRPC{fakeRPC: newFake()})
	if _, e := c.ChecksCreationHistory(nil, testAddress, 49, 50); e == nil {
		t.Fatal("nil context")
	}
	if _, e := c.ChecksCreationReceipt(nil, testAddress, common.HexToHash("0x1")); e == nil {
		t.Fatal("nil receipt context")
	}
	first := creationFixture(t)
	for _, change := range []func(*types.Log){func(x *types.Log) { x.Index = 4; x.TxHash = common.HexToHash("0x999") }, func(x *types.Log) { x.Index = 4; x.TxIndex = 3 }, func(x *types.Log) { x.Index = 4; x.BlockNumber = 51; x.BlockHash = common.HexToHash("0x51") }, func(x *types.Log) { x.Index = 2; x.TxIndex = 3; x.TxHash = common.HexToHash("0x999") }} {
		second := creationFixture(t)
		change(&second)
		if _, e := checksCreationRows(testAddress, []types.Log{first, second}); e == nil {
			t.Fatal("incoherent transaction accepted")
		}
	}
	repeated := creationFixture(t)
	repeated.Index = 4
	rows, e := checksCreationRows(testAddress, []types.Log{repeated, first})
	if e != nil || len(rows) != 2 || rows[0].Log.Index != 3 {
		t.Fatal("repeated historical hash removed", e)
	}
}
