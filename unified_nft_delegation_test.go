package decimal

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestUnifiedNFTDelegationOfflineBuildersAndOperator(t *testing.T) {
	client := &Client{}
	d, e := client.UnifiedNFTDelegation(testAddress)
	if e != nil {
		t.Fatal(e)
	}
	if d.ApprovalOperator() != testAddress {
		t.Fatal("wrong operator")
	}
	a, _ := delegationUpgradeABI()
	value := reflect.ValueOf(d)
	count := 0
	for i := 0; i < value.NumMethod(); i++ {
		name := value.Type().Method(i).Name
		if !strings.HasPrefix(name, "BuildDelegateNFT") && !strings.HasPrefix(name, "BuildWithdrawNFT") && !strings.HasPrefix(name, "BuildTransferNFT") {
			continue
		}
		m := value.Method(i)
		args := make([]reflect.Value, m.Type().NumIn())
		for j := range args {
			typ := m.Type().In(j)
			switch typ {
			case reflect.TypeOf(common.Address{}):
				args[j] = reflect.ValueOf(common.HexToAddress("0x1234"))
			case reflect.TypeOf((*big.Int)(nil)):
				args[j] = reflect.ValueOf(big.NewInt(17))
			case reflect.TypeOf(uint8(0)):
				args[j] = reflect.ValueOf(uint8(27))
			default:
				args[j] = reflect.Zero(typ)
			}
		}
		result := m.Call(args)
		if !result[1].IsNil() {
			t.Fatal(name, result[1].Interface())
		}
		call := result[0].Interface().(Call)
		_, method, err := method(a, call.Signature)
		if err != nil {
			t.Fatal(err)
		}
		if call.To != testAddress || !bytes.Equal(call.Data[:4], method.ID) || call.Value.Sign() != 0 {
			t.Fatal("wrong route", name)
		}
		if _, err := method.Inputs.Unpack(call.Data[4:]); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 13 {
		t.Fatal("missing official builders", count)
	}
	collection := common.HexToAddress("0x9999")
	c721, err := d.BuildERC721Approval(collection, big.NewInt(42))
	if err != nil {
		t.Fatal(err)
	}
	c1155, err := d.BuildERC1155ApprovalForAll(collection)
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range []Call{c721, c1155} {
		aa, _ := loadABI(call.Profile)
		_, m, err := method(aa, call.Signature)
		if err != nil {
			t.Fatal(err)
		}
		args, err := m.Inputs.Unpack(call.Data[4:])
		if err != nil {
			t.Fatal(err)
		}
		if call.To != collection || args[0].(common.Address) != d.Address {
			t.Fatal("wrong approval operator")
		}
	}
	if _, err := client.UnifiedNFTDelegation(common.Address{}); err == nil {
		t.Fatal("zero address")
	}
}

func TestUnifiedNFTDelegationSavedViews(t *testing.T) {
	raw, err := os.ReadFile("testdata/delegation/mainnet_nft_views_post_recovery.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Block   uint64
		Address common.Address
		Calls   map[string]struct{ Data, Result string }
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	var response []byte
	backend := &contractReadAuditBackend{fakeRPC: newFake(), read: func(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		if block == nil || block.Uint64() != f.Block || msg.To == nil || *msg.To != f.Address {
			t.Fatal("snapshot identity")
		}
		for _, c := range f.Calls {
			if bytes.Equal(msg.Data, common.FromHex(c.Data)) {
				return append([]byte(nil), response...), nil
			}
		}
		t.Fatal("unrecorded call")
		return nil, nil
	}}
	backend.chain = big.NewInt(75)
	client, err := NewClient(Mainnet(), backend)
	if err != nil {
		t.Fatal(err)
	}
	d, err := client.UnifiedNFTDelegation(f.Address)
	if err != nil {
		t.Fatal(err)
	}
	zero := new(big.Int)
	block := new(big.Int).SetUint64(f.Block)
	response = common.FromHex(f.Calls["getNFTStakeId"].Result)
	id, err := d.ReadNFTStakeIDAt(context.Background(), block, common.Address{}, zero, common.Address{}, common.Address{}, zero)
	if err != nil || !bytes.Equal(id[:], response) {
		t.Fatal("saved id", err)
	}
	response = common.FromHex(f.Calls["getNFTStake"].Result)
	stake, err := d.ReadNFTStakeAt(context.Background(), block, common.Address{}, zero, common.Address{}, common.Address{}, zero)
	if err != nil || stake.IsActive || stake.Amount == nil || stake.Amount.Sign() != 0 || reflect.TypeOf(stake).NumField() != 10 {
		t.Fatal("saved empty tuple10", err)
	}
	for _, bad := range [][]byte{response[:len(response)-1], append(append([]byte{}, response...), make([]byte, 32)...)} {
		response = bad
		if _, err := d.ReadNFTStakeAt(context.Background(), block, common.Address{}, zero, common.Address{}, common.Address{}, zero); err == nil {
			t.Fatal("malformed tuple accepted")
		}
	}
	if _, err := d.ReadNFTStakeAt(context.Background(), nil, common.Address{}, zero, common.Address{}, common.Address{}, zero); err == nil {
		t.Fatal("missing block")
	}
}
