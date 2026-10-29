package decimal

import "testing"

func FuzzExactAmounts(f *testing.F) {
	for _, s := range []string{"0", "1.25", "0001.000", "-1", "1e5", "0.000000000000000001"} {
		f.Add(s, uint8(18))
	}
	f.Fuzz(func(t *testing.T, s string, decimals uint8) {
		if len(s) > 4096 {
			return
		}
		n, e := ParseUnits(s, decimals)
		if e != nil {
			return
		}
		if n.Sign() < 0 || n.BitLen() > 256 {
			t.Fatal("invalid successful amount")
		}
		round, e := ParseUnits(FormatUnits(n, decimals), decimals)
		if e != nil || round.Cmp(n) != 0 {
			t.Fatal("loss of amount precision")
		}
	})
}
func FuzzSignatureValidation(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 65))
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		sig, e := SplitSignature(data)
		if e == nil && (len(data) != 65 || (sig.V != 27 && sig.V != 28)) {
			t.Fatal("invalid successful signature")
		}
	})
}
