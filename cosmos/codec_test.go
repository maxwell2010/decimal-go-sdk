package cosmos

import (
	"bytes"
	"encoding/hex"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"testing"
)

func TestInventoryAndAllAnyRoundTrips(t *testing.T) {
	if len(Inventory()) != 43 {
		t.Fatalf("expected43 requests, got%d: %v", len(Inventory()), Inventory())
	}
	for _, name := range Inventory() {
		t.Run(name, func(t *testing.T) {
			m, e := NewMessage(name)
			if e != nil {
				t.Fatal(e)
			}
			a, e := Pack(m)
			if e != nil {
				t.Fatal(e)
			}
			if a.TypeUrl != name {
				t.Fatal("wrong Cosmos type URL")
			}
			round, e := Unpack(a)
			if e != nil || !proto.Equal(m, round) {
				t.Fatal("Any roundtrip", e)
			}
		})
	}
}
func TestSendCoinEnvelopeWireAndSigner(t *testing.T) {
	m, e := NewMessage("/decimal.coin.v1.MsgSendCoin")
	if e != nil {
		t.Fatal(e)
	}
	if e = protojson.Unmarshal([]byte(`{"sender":"d0sender","recipient":"d0recipient","coin":{"denom":"del","amount":"123"}}`), m); e != nil {
		t.Fatal(e)
	}
	body, e := EncodeBody([]proto.Message{m}, "memo", 7)
	if e != nil {
		t.Fatal(e)
	}
	pub := append([]byte{2}, bytes.Repeat([]byte{1}, 32)...)
	auth, e := EncodeAuthInfo([]SignerInfo{{pub, 5}}, []Coin{{"del", "7"}}, 200000)
	if e != nil {
		t.Fatal(e)
	}
	doc, e := EncodeSignDoc(body, auth, "decimal_75-1", 9)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := Decode("cosmos.tx.v1beta1.SignDoc", doc)
	if e != nil {
		t.Fatal(e)
	}
	if decoded.Get(decoded.Descriptor().Fields().ByName("chain_id")).String() != "decimal_75-1" {
		t.Fatal("chain ID lost")
	}
	raw, e := EncodeTxRaw(body, auth, [][]byte{bytes.Repeat([]byte{1}, 65)})
	if e != nil {
		t.Fatal(e)
	}
	tag, n := protowire.ConsumeVarint(raw)
	if n <= 0 || tag != 10 {
		t.Fatal("TxRaw body field wire number")
	}
	payload, n := protowire.ConsumeBytes(raw[n:])
	if n <= 0 || !bytes.Equal(payload, body) {
		t.Fatal("TxRaw must retain exact signed body bytes")
	}
	if TransactionHash(raw) == [32]byte{} {
		t.Fatal("hash")
	}
	bad, _ := NewMessage("/decimal.coin.v1.MsgSendCoin")
	if e = protojson.Unmarshal([]byte(`{"invented":1}`), bad); e == nil {
		t.Fatal("invented fields accepted")
	}
}
func TestEnvelopeRejectsMalformedAndWrongSignatureCount(t *testing.T) {
	auth, _ := EncodeAuthInfo([]SignerInfo{{append([]byte{2}, bytes.Repeat([]byte{1}, 32)...), 0}}, nil, 0)
	body, _ := EncodeBody(nil, "", 0)
	if _, e := EncodeSignDoc([]byte{0xff}, auth, "chain", 0); e == nil {
		t.Fatal("malformed body")
	}
	if _, e := EncodeSignDoc(body, auth, "", 0); e == nil {
		t.Fatal("empty chain")
	}
	if _, e := EncodeTxRaw(body, auth, nil); e == nil {
		t.Fatal("missing signatures")
	}
	if _, e := EncodeTxRaw(body, auth, [][]byte{{1}}); e == nil {
		t.Fatal("wrong signature length")
	}
	decoded, _ := Decode("cosmos.tx.v1beta1.AuthInfo", auth)
	signer := list(decoded, "signer_infos").Get(0).Message()
	mode := signer.Get(signer.Descriptor().Fields().ByName("mode_info")).Message()
	single := mode.Get(mode.Descriptor().Fields().ByName("single")).Message()
	if single.Get(single.Descriptor().Fields().ByName(protoreflect.Name("mode"))).Enum() != 1 {
		t.Fatal("not SIGN_MODE_DIRECT")
	}
}
func TestEthermintDigestUsesKeccakAnd32ByteBypass(t *testing.T) {
	h := EthermintSigningDigest(nil)
	if hex.EncodeToString(h[:]) != "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470" {
		t.Fatal("wrong Keccak")
	}
	raw := bytes.Repeat([]byte{3}, 32)
	h = EthermintSigningDigest(raw)
	if !bytes.Equal(h[:], raw) {
		t.Fatal("upstream32byte digest bypass differs")
	}
}

func TestPinnedNativeRouteIsEightNotFortyThree(t *testing.T) {
	count := 0
	for _, url := range Inventory() {
		if PinnedNativeRoute(url) {
			count++
		}
	}
	if count != 8 {
		t.Fatalf("expected8 ante-admitted types, got%d", count)
	}
	if PinnedNativeRoute("/decimal.coin.v1.MsgSendCoin") {
		t.Fatal("legacy keeper existence mistaken for ante admission")
	}
	if e := ValidateTxRaw(nil); e == nil {
		t.Fatal("unsigned empty tx accepted")
	}
}
