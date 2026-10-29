package cosmos

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/cosmos/btcutil/bech32"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"os"
	"testing"
)

func TestValidationAndSignersRejectSpoofedDescriptor(t *testing.T) {
	file, e := protodesc.NewFile(&descriptorpb.FileDescriptorProto{Name: proto.String("spoof.proto"), Syntax: proto.String("proto3"), Package: proto.String("decimal.coin.v1"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("MsgSendCoin"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("sender"), Number: proto.Int32(99), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}}}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	spoof := dynamicpb.NewMessage(file.Messages().Get(0))
	address, _ := EncodeAccountAddress(bytes.Repeat([]byte{1}, 20))
	spoof.Set(spoof.Descriptor().Fields().ByName("sender"), protoreflect.ValueOfString(address))
	if e = ValidateBasic(spoof); e == nil {
		t.Fatal("spoofed validation accepted")
	}
	if _, e = GetSigners(spoof); e == nil {
		t.Fatal("spoofed signer schema accepted")
	}
}

func TestValidateBasicExecutedSDKUpgradeReference(t *testing.T) {
	raw, e := os.ReadFile("schema/validation-reference.json")
	if e != nil {
		t.Fatal(e)
	}
	var records []struct {
		TypeURL string `json:"type_url"`
		Raw     string `json:"protobuf_hex"`
		Valid   bool   `json:"valid"`
	}
	if e = json.Unmarshal(raw, &records); e != nil {
		t.Fatal(e)
	}
	for _, record := range records {
		b, e := hex.DecodeString(record.Raw)
		if e != nil {
			t.Fatal(e)
		}
		m, e := Decode(record.TypeURL, b)
		if e != nil {
			t.Fatal(e)
		}
		if valid := ValidateBasic(m) == nil; valid != record.Valid {
			t.Fatalf("SDK differential mismatch %s", record.TypeURL)
		}
	}
}

func TestValidateBasicExecutedNodeReferenceAll43(t *testing.T) {
	raw, e := os.ReadFile("schema/validation-node-reference.json")
	if e != nil {
		t.Fatal(e)
	}
	var records []struct {
		TypeURL     string `json:"type_url"`
		Case        string `json:"case"`
		Raw         string `json:"protobuf_hex"`
		Valid       bool   `json:"valid"`
		Panic       bool   `json:"panic"`
		DecodeError bool   `json:"decode_error"`
	}
	if e = json.Unmarshal(raw, &records); e != nil {
		t.Fatal(e)
	}
	if len(records) != 129 {
		t.Fatalf("unexpected node vector count %d", len(records))
	}
	covered := map[string]int{}
	for _, record := range records {
		if record.Panic || record.DecodeError {
			t.Fatal("reference did not execute validator")
		}
		b, e := hex.DecodeString(record.Raw)
		if e != nil {
			t.Fatal(e)
		}
		m, e := Decode(record.TypeURL, b)
		if e != nil {
			t.Fatal(e)
		}
		if valid := ValidateBasic(m) == nil; valid != record.Valid {
			t.Fatalf("node differential mismatch %s %s", record.TypeURL, record.Case)
		}
		covered[record.TypeURL]++
	}
	for _, name := range Inventory() {
		if covered[name] != 3 {
			t.Fatalf("missing reference cases %s", name)
		}
	}
}

func FuzzPinnedValidateBasic(f *testing.F) {
	f.Add(uint8(0), []byte{})
	f.Add(uint8(42), []byte{10, 1, 65})
	f.Fuzz(func(t *testing.T, index uint8, raw []byte) {
		if len(raw) > 4096 {
			return
		}
		inventory := Inventory()
		m, e := Decode(inventory[int(index)%len(inventory)], raw)
		if e != nil {
			return
		}
		rule := signerRules[inventory[int(index)%len(inventory)]]
		address, _ := EncodeAddress(rule.hrp, bytes.Repeat([]byte{1}, 20))
		m.Set(m.Descriptor().Fields().ByName(protoreflect.Name(rule.field)), protoreflect.ValueOfString(address))
		before := proto.Clone(m)
		_ = ValidateBasic(m)
		if !proto.Equal(m, before) {
			t.Fatal("validation mutates payload")
		}
	})
}

func fillValidationFixture(m protoreflect.Message) {
	account, _ := EncodeAccountAddress(bytes.Repeat([]byte{1}, 20))
	recipient, _ := EncodeAccountAddress(bytes.Repeat([]byte{2}, 20))
	validator, _ := EncodeAddress(ValidatorHRP, bytes.Repeat([]byte{1}, 20))
	validator2, _ := EncodeAddress(ValidatorHRP, bytes.Repeat([]byte{2}, 20))
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		name := string(f.Name())
		if name == "time" || name == "upgraded_client_state" {
			continue
		}
		if f.IsList() {
			l := m.Mutable(f).List()
			switch f.Kind() {
			case protoreflect.StringKind:
				if name == "owners" {
					l.Append(protoreflect.ValueOfString(account))
					l.Append(protoreflect.ValueOfString(recipient))
				}
			case protoreflect.Uint32Kind:
				l.Append(protoreflect.ValueOfUint32(1))
				l.Append(protoreflect.ValueOfUint32(2))
			case protoreflect.MessageKind:
				v := l.NewElement()
				fillValidationFixture(v.Message())
				l.Append(v)
			}
			continue
		}
		switch f.Kind() {
		case protoreflect.MessageKind:
			fillValidationFixture(m.Mutable(f).Message())
		case protoreflect.StringKind:
			s := "fixture"
			switch name {
			case "sender", "oracle", "authority", "delegator", "reward_address", "wallet":
				s = account
			case "recipient":
				s = recipient
			case "validator", "operator_address", "validator_src":
				s = validator
			case "validator_dst":
				s = validator2
			case "denom", "quote", "coin_denom_to_sell":
				s = "abc"
			case "amount", "price", "commission", "transaction_number":
				s = "1"
			case "initial_volume", "initial_reserve", "min_volume":
				s = "1000000000000000000"
			case "limit_volume":
				s = "100000000000000000000"
			case "r", "s":
				s = string(bytes.Repeat([]byte{'0'}, 64))
			case "id":
				words, _ := bech32.ConvertBits([]byte{1}, 8, 5, true)
				s, _ = bech32.Encode("d0mstx", words)
			}
			m.Set(f, protoreflect.ValueOfString(s))
		case protoreflect.BytesKind:
			n := 1
			if name == "public_key" {
				n = 33
			}
			m.Set(f, protoreflect.ValueOfBytes(make([]byte, n)))
		case protoreflect.Int64Kind:
			m.Set(f, protoreflect.ValueOfInt64(1))
		case protoreflect.Uint64Kind:
			m.Set(f, protoreflect.ValueOfUint64(1))
		case protoreflect.Uint32Kind:
			n := uint32(1)
			if name == "crr" {
				n = 50
			}
			m.Set(f, protoreflect.ValueOfUint32(n))
		}
	}
	// Distinct buy/sell denominations are a source predicate.
	for _, name := range []protoreflect.Name{"max_coin_to_sell", "min_coin_to_buy"} {
		if f := fields.ByName(name); f != nil {
			coin := m.Mutable(f).Message()
			coin.Set(coin.Descriptor().Fields().ByName("denom"), protoreflect.ValueOfString("xyz"))
		}
	}
	if f := fields.ByName("dest_chain"); f != nil {
		m.Set(f, protoreflect.ValueOfUint32(2))
	}
}
func TestPinnedValidateBasicAllMessages(t *testing.T) {
	for _, name := range Inventory() {
		t.Run(name, func(t *testing.T) {
			m, _ := NewMessage(name)
			if e := ValidateBasic(m); e == nil {
				t.Fatal("empty message accepted")
			}
			fillValidationFixture(m.ProtoReflect())
			before := proto.Clone(m)
			if e := ValidateBasic(m); e != nil {
				t.Fatal(e)
			}
			if !proto.Equal(before, m) {
				t.Fatal("validation mutated message")
			}
			rule := signerRules[name]
			m.Set(m.Descriptor().Fields().ByName(protoreflect.Name(rule.field)), protoreflect.ValueOfString("invalid"))
			if e := ValidateBasic(m); e == nil {
				t.Fatal("invalid signer accepted")
			}
		})
	}
}

func TestPinnedValidationNegativeBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		change func(*dynamicpb.Message)
	}{
		{"/decimal.coin.v1.MsgCreateCoin", func(m *dynamicpb.Message) { set(m, "crr", protoreflect.ValueOfUint32(9)) }},
		{"/decimal.coin.v1.MsgCreateCoin", func(m *dynamicpb.Message) {
			set(m, "title", protoreflect.ValueOfString(string(bytes.Repeat([]byte{'a'}, 65))))
		}},
		{"/decimal.coin.v1.MsgCreateCoin", func(m *dynamicpb.Message) {
			set(m, "initial_reserve", protoreflect.ValueOfString("999999999999999999"))
		}},
		{"/decimal.coin.v1.MsgSendCoin", func(m *dynamicpb.Message) { set(m, "recipient", m.Get(m.Descriptor().Fields().ByName("sender"))) }},
		{"/decimal.nft.v1.MsgMintToken", func(m *dynamicpb.Message) { set(m, "quantity", protoreflect.ValueOfUint32(0)) }},
		{"/decimal.nft.v1.MsgBurnToken", func(m *dynamicpb.Message) {
			l := m.Mutable(m.Descriptor().Fields().ByName("sub_token_ids")).List()
			l.Append(l.Get(0))
		}},
		{"/decimal.validator.v1.MsgCreateValidator", func(m *dynamicpb.Message) { set(m, "commission", protoreflect.ValueOfString("-1")) }},
		{"/decimal.validator.v1.MsgRedelegate", func(m *dynamicpb.Message) {
			set(m, "validator_dst", m.Get(m.Descriptor().Fields().ByName("validator_src")))
		}},
		{"/decimal.validator.v1.MsgHaltChain", func(m *dynamicpb.Message) { set(m, "height", protoreflect.ValueOfInt64(-1)) }},
		{"/decimal.swap.v1.MsgRedeemSwap", func(m *dynamicpb.Message) { set(m, "r", protoreflect.ValueOfString("0x01")) }},
		{"/decimal.fee.v1.MsgUpdateCoinPrices", func(m *dynamicpb.Message) {
			l := m.Mutable(m.Descriptor().Fields().ByName("prices")).List()
			l.Append(l.Get(0))
		}},
		{"/decimal.multisig.v1.MsgCreateWallet", func(m *dynamicpb.Message) { set(m, "threshold", protoreflect.ValueOfUint32(9999)) }},
	}
	for _, tc := range cases {
		m, _ := NewMessage(tc.name)
		fillValidationFixture(m.ProtoReflect())
		tc.change(m)
		if e := ValidateBasic(m); e == nil {
			t.Fatalf("invalid boundary accepted %s", tc.name)
		}
	}
}
func TestValidateBasicRetainsSourceQuirks(t *testing.T) {
	m := NewMsgRedeemCheck()
	fillValidationFixture(m.ProtoReflect())
	set(m, "check", protoreflect.ValueOfString(""))
	if e := ValidateBasic(m); e != nil {
		t.Fatal("source TODO checks more than sender", e)
	}
	nft := NewMsgUpdateReserve()
	fillValidationFixture(nft.ProtoReflect())
	reserve := nft.Mutable(nft.Descriptor().Fields().ByName("reserve")).Message()
	reserve.Set(reserve.Descriptor().Fields().ByName("amount"), protoreflect.ValueOfString("-1"))
	if e := ValidateBasic(nft); e != nil {
		t.Fatal("source only rejects zero reserve", e)
	}
	var absent *dynamicpb.Message
	if e := ValidateBasic(absent); e == nil {
		t.Fatal("nil message")
	}
}
