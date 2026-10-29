package cosmos

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/cosmos/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"strings"
	"testing"
)

func issuedFixture(t *testing.T) (string, []byte) {
	t.Helper()
	key, e := crypto.HexToECDSA(strings.Repeat("01", 32))
	if e != nil {
		t.Fatal(e)
	}
	secret := []byte("public historical check fixture")
	encoded, e := IssueCheck(context.Background(), "decimal_test-1", "del", big.NewInt(123), big.NewInt(7), 100, secret, crypto.CompressPubkey(&key.PublicKey), func(_ context.Context, h [32]byte) ([]byte, error) { return crypto.Sign(h[:], key) })
	if e != nil {
		t.Fatal(e)
	}
	if string(secret) != "public historical check fixture" {
		t.Fatal("caller passphrase mutated")
	}
	return encoded, crypto.PubkeyToAddress(key.PublicKey).Bytes()
}
func TestHistoricalCheckWireAndRedemption(t *testing.T) {
	encoded, issuer := issuedFixture(t)
	c, e := ParseCheck(encoded)
	if e != nil {
		t.Fatal(e)
	}
	sender, e := c.Sender()
	if e != nil || !bytes.Equal(sender, issuer) {
		t.Fatal("issuer recovery", e)
	}
	independent, e := rlp.EncodeToBytes([]any{c.ChainID, c.Denom, c.Amount, c.Nonce, c.DueBlock, c.Lock, c.V, c.R, c.S})
	if e != nil || !bytes.Equal(independent, base58.Decode(encoded)) {
		t.Fatal("upstream RLP field order")
	}
	receiver := bytes.Repeat([]byte{2}, 20)
	message, e := CreateRedeemCheck(context.Background(), encoded, c.ChainID, 100, receiver, []byte("public historical check fixture"))
	if e != nil {
		t.Fatal(e)
	}
	proof, e := base64.StdEncoding.DecodeString(message.Get(message.Descriptor().Fields().ByName("proof")).String())
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := rlp.EncodeToBytes([]any{receiver})
	digest := crypto.Keccak256Hash(raw)
	pub, e := crypto.Ecrecover(digest[:], proof)
	lockPub, e2 := c.lockPublicKey()
	if e != nil || e2 != nil || !bytes.Equal(pub, lockPub) {
		t.Fatal("redemption proof is not bound to lock and receiver")
	}
}
func TestHistoricalCheckRejectsMalformedOrInvalidContext(t *testing.T) {
	encoded, _ := issuedFixture(t)
	for _, input := range []string{"", "0OIl", strings.Repeat("1", 6001), encoded + "1"} {
		if _, e := ParseCheck(input); e == nil {
			t.Fatal("malformed check accepted")
		}
	}
	c, _ := ParseCheck(encoded)
	c.S = new(big.Int).Set(crypto.S256().Params().N)
	raw, _ := rlp.EncodeToBytes(c)
	if _, e := ParseCheck(base58.Encode(raw)); e == nil {
		t.Fatal("invalid signature scalar accepted")
	}
	receiver := bytes.Repeat([]byte{2}, 20)
	for _, tc := range []struct {
		chain  string
		height uint64
		secret []byte
	}{{"wrong", 1, []byte("public historical check fixture")}, {"decimal_test-1", 101, []byte("public historical check fixture")}, {"decimal_test-1", 1, []byte("wrong")}} {
		if _, e := CreateRedeemCheck(context.Background(), encoded, tc.chain, tc.height, receiver, tc.secret); e == nil {
			t.Fatal("crosschain/expired/wronglock check accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := CreateRedeemCheck(ctx, encoded, "decimal_test-1", 1, receiver, nil); e == nil {
		t.Fatal("cancellation ignored")
	}
}
