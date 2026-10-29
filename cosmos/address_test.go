package cosmos

import (
	"bytes"
	"github.com/cosmos/btcutil/bech32"
	"strings"
	"testing"
)

func TestDecimalAddressConversionsPreserveBytes(t *testing.T) {
	raw := bytes.Repeat([]byte{0x12}, 20)
	account, e := EncodeAccountAddress(raw)
	if e != nil || !strings.HasPrefix(account, "d01") {
		t.Fatal(account, e)
	}
	decoded, e := DecodeAccountAddress(account)
	if e != nil || !bytes.Equal(decoded, raw) {
		t.Fatal(e)
	}
	h, e := AccountAddressToHex(account)
	if e != nil || h != "0x"+strings.Repeat("12", 20) {
		t.Fatal(h, e)
	}
	round, e := HexToAccountAddress(h)
	if e != nil || round != account {
		t.Fatal(round, e)
	}
	operator, e := EncodeAddress(ValidatorHRP, raw)
	if e != nil {
		t.Fatal(e)
	}
	converted, e := ValidatorToAccount(operator)
	if e != nil || converted != account {
		t.Fatal("operator bytes were hashed or changed", e)
	}
	if _, e = DecodeAccountAddress(operator); e == nil {
		t.Fatal("validator prefix accepted as account")
	}
	legacyHRP, e := EncodeAddress("dx", raw)
	if e != nil {
		t.Fatal(e)
	}
	converted, e = ReencodeAddress(legacyHRP, AccountHRP)
	if e != nil || converted != account {
		t.Fatal("explicit HRP conversion", e)
	}
}
func TestAddressChecksumPaddingAndBoundaries(t *testing.T) {
	for _, size := range []int{1, 20, 255} {
		input := bytes.Repeat([]byte{1}, size)
		encoded, e := EncodeAccountAddress(input)
		if e != nil {
			t.Fatal(e)
		}
		output, e := DecodeAccountAddress(encoded)
		if e != nil || !bytes.Equal(input, output) {
			t.Fatal(size, e)
		}
	}
	for _, size := range []int{0, 256} {
		if _, e := EncodeAccountAddress(make([]byte, size)); e == nil {
			t.Fatal("length", size)
		}
	}
	valid, _ := EncodeAccountAddress(bytes.Repeat([]byte{2}, 20))
	mutated := valid[:len(valid)-1] + "!"
	padding, _ := bech32.Encode(AccountHRP, []byte{1})
	for _, address := range []string{"", mutated, padding, "D0" + valid[2:], "0x1234", valid + " "} {
		if _, e := DecodeAccountAddress(address); e == nil {
			t.Fatalf("accepted invalid %q", address)
		}
	}
	for _, h := range []string{"12", "0x", "0x1", "0xGG"} {
		if _, e := HexToAccountAddress(h); e == nil {
			t.Fatal("invalid hex accepted")
		}
	}
	for _, hrp := range []string{"", "D0", "d\n0", strings.Repeat("a", 84)} {
		if _, e := EncodeAddress(hrp, []byte{1}); e == nil {
			t.Fatal("invalid HRP")
		}
	}
}
