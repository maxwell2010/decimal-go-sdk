package cosmos

import (
	"bytes"
	"encoding/json"
	"errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"os"
	"testing"
)

func TestSignerRulesCoverAllPinnedRequests(t *testing.T) {
	var sources []struct {
		TypeURL string `json:"type_url"`
		Field   string `json:"field"`
		HRP     string `json:"hrp"`
	}
	raw, e := os.ReadFile("schema/signer-sources.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &sources); e != nil {
		t.Fatal(e)
	}
	if len(sources) != 43 || len(signerRules) != len(Inventory()) {
		t.Fatal("signer/source inventory mismatch")
	}
	expected := bytes.Repeat([]byte{7}, 20)
	seen := map[string]bool{}
	for _, source := range sources {
		t.Run(source.TypeURL, func(t *testing.T) {
			message, e := NewMessage(source.TypeURL)
			if e != nil {
				t.Fatal(e)
			}
			address, e := EncodeAddress(source.HRP, expected)
			if e != nil {
				t.Fatal(e)
			}
			field := message.Descriptor().Fields().ByName(protoreflect.Name(source.Field))
			if field == nil {
				t.Fatal("source rule differs from protobuf descriptor")
			}
			message.Set(field, protoreflect.ValueOfString(address))
			signers, e := GetSigners(message)
			if e != nil || len(signers) != 1 || !bytes.Equal(signers[0], expected) {
				t.Fatal("wrong source-backed signer", e)
			}
			seen[source.TypeURL] = true
		})
	}
	for _, name := range Inventory() {
		if !seen[name] {
			t.Fatal("uncovered request", name)
		}
	}
}
func TestSignerExtractionRejectsInvalidRolePrefixAndUnknownTypes(t *testing.T) {
	raw := bytes.Repeat([]byte{1}, 20)
	account, _ := EncodeAccountAddress(raw)
	operator, _ := EncodeAddress(ValidatorHRP, raw)
	m := NewMsgSetOnline()
	field := m.Descriptor().Fields().ByName("validator")
	m.Set(field, protoreflect.ValueOfString(account))
	if _, e := GetSigners(m); e == nil {
		t.Fatal("account HRP accepted for operator role")
	}
	m.Set(field, protoreflect.ValueOfString(operator))
	if _, e := GetSigners(m); e != nil {
		t.Fatal(e)
	}
	unknown, e := NewMessage("/cosmos.base.v1beta1.Coin")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = GetSigners(unknown); !errors.Is(e, ErrUnsupportedSigners) {
		t.Fatal("unknown type accepted", e)
	}
	if _, e = GetSigners(nil); e == nil {
		t.Fatal("nil accepted")
	}
	missing := NewMsgSendCoin()
	missing = nil
	if _, e = GetSigners(missing); e == nil {
		t.Fatal("typed nil accepted")
	}
	if _, e = GetSigners(NewMsgUpdateCoinPrices()); e == nil {
		t.Fatal("empty oracle accepted")
	}
}
func TestTransactionSignerOrderAndFeePayer(t *testing.T) {
	a := bytes.Repeat([]byte{1}, 20)
	b := bytes.Repeat([]byte{2}, 20)
	c := bytes.Repeat([]byte{3}, 20)
	messages := []proto.Message{}
	for _, raw := range [][]byte{b, a, b} {
		m := NewMsgSendCoin()
		address, _ := EncodeAccountAddress(raw)
		m.Set(m.Descriptor().Fields().ByName("sender"), protoreflect.ValueOfString(address))
		messages = append(messages, m)
	}
	payer, _ := EncodeAccountAddress(c)
	signers, e := TransactionSigners(messages, payer)
	if e != nil || len(signers) != 3 || !bytes.Equal(signers[0], b) || !bytes.Equal(signers[1], a) || !bytes.Equal(signers[2], c) {
		t.Fatal("first-seen ordering or payer", e)
	}
	payer, _ = EncodeAccountAddress(b)
	signers, e = TransactionSigners(messages, payer)
	if e != nil || len(signers) != 2 {
		t.Fatal("payer duplicated")
	}
	if _, e = TransactionSigners(messages, "invalid"); e == nil {
		t.Fatal("invalid payer")
	}
	if _, e = TransactionSigners(nil, ""); e == nil {
		t.Fatal("no messages")
	}
}
