package decimal

import (
	"errors"
	"math/big"
	"strings"
)

// ParseUnits converts an unsigned decimal amount without floating point or rounding.
func ParseUnits(s string, decimals uint8) (*big.Int, error) {
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" {
		return nil, errors.New("invalid decimal amount")
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
		if frac == "" {
			return nil, errors.New("missing fraction")
		}
	}
	if len(frac) > int(decimals) {
		return nil, errors.New("amount exceeds token precision")
	}
	for _, c := range parts[0] + frac {
		if c < '0' || c > '9' {
			return nil, errors.New("invalid decimal amount")
		}
	}
	n, ok := new(big.Int).SetString(parts[0]+frac+strings.Repeat("0", int(decimals)-len(frac)), 10)
	if !ok || n.BitLen() > 256 {
		return nil, errors.New("amount exceeds uint256")
	}
	return n, nil
}

// FormatUnits formats an integer exactly; its result can be parsed with ParseUnits.
func FormatUnits(n *big.Int, decimals uint8) string {
	if n == nil {
		return "0"
	}
	sign := ""
	if n.Sign() < 0 {
		sign = "-"
	}
	s := new(big.Int).Abs(n).String()
	if decimals == 0 {
		return sign + s
	}
	if len(s) <= int(decimals) {
		s = strings.Repeat("0", int(decimals)+1-len(s)) + s
	}
	p := len(s) - int(decimals)
	f := strings.TrimRight(s[p:], "0")
	if f == "" {
		return sign + s[:p]
	}
	return sign + s[:p] + "." + f
}

func uint256(n *big.Int) bool { return n != nil && n.Sign() >= 0 && n.BitLen() <= 256 }
func copyInt(n *big.Int) *big.Int {
	if n == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(n)
}
