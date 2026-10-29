package cosmos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"
	"testing"
)

// This fixture was produced by executing tools/reference against the actual
// Cosmos SDK v0.46.16 generated codecs and Ethermint v0.22.13 signer.
func TestSignedTxRawMatchesExecutedOfficialReference(t *testing.T) {
	raw, e := os.ReadFile("schema/reference-tx.json")
	if e != nil {
		t.Fatal(e)
	}
	want := map[string]string{}
	if e = json.Unmarshal(raw, &want); e != nil {
		t.Fatal(e)
	}
	msg := NewMsgSendCoin()
	if e = protojson.Unmarshal([]byte(`{"sender":"d0sender","recipient":"d0recipient","coin":{"denom":"del","amount":"123"}}`), msg); e != nil {
		t.Fatal(e)
	}
	body, e := EncodeBody([]proto.Message{msg}, "reference", 7)
	if e != nil {
		t.Fatal(e)
	}
	key, _ := crypto.HexToECDSA(strings.Repeat("01", 32))
	pub := crypto.CompressPubkey(&key.PublicKey)
	auth, e := EncodeAuthInfo([]SignerInfo{{pub, 9}}, []Coin{{"del", "7"}}, 200000)
	if e != nil {
		t.Fatal(e)
	}
	doc, e := EncodeSignDoc(body, auth, "decimal_test-1", 11)
	if e != nil {
		t.Fatal(e)
	}
	result, e := SignDirectTx(context.Background(), DirectTx{Body: body, ChainID: "decimal_test-1", Fee: []Coin{{"del", "7"}}, Gas: 200000, SignerOrder: [][]byte{pub}, Signers: []DirectSigner{{PublicKey: pub, AccountNumber: 11, Sequence: 9, Sign: func(_ context.Context, d [32]byte) ([]byte, error) { return crypto.Sign(d[:], key) }}}})
	if e != nil {
		t.Fatal(e)
	}
	digest := EthermintSigningDigest(doc)
	sig, e := crypto.Sign(digest[:], key)
	if e != nil {
		t.Fatal(e)
	}
	got := map[string][]byte{"signature": sig, "body": body, "auth": auth, "sign_doc": doc, "tx_raw": result.Raw, "public_key": pub}
	for name, b := range got {
		if hex.EncodeToString(b) != want[name] {
			t.Fatalf("%s differs from executed official reference", name)
		}
	}
}
