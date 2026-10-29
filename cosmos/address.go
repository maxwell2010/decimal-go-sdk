package cosmos

import (
	"encoding/hex"
	"errors"
	"github.com/cosmos/btcutil/bech32"
	"strings"
)

const AccountHRP = "d0"
const ValidatorHRP = "d0valoper"
const ConsensusHRP = "d0valcons"

// EncodeAddress follows Cosmos SDK0.46.16's default nonempty/max255-byte address
// verifier and Bech32 conversion. It does not hash a public key or derive a
// legacy address. Applications with a custom node verifier must enforce it too.
func EncodeAddress(hrp string, raw []byte) (string, error) {
	if len(raw) == 0 || len(raw) > 255 {
		return "", errors.New("address must contain 1..255 bytes")
	}
	if hrp == "" || len(hrp) > 83 || strings.ToLower(hrp) != hrp {
		return "", errors.New("lowercase address HRP required")
	}
	for _, c := range hrp {
		if c < 33 || c > 126 {
			return "", errors.New("invalid address HRP character")
		}
	}
	words, e := bech32.ConvertBits(raw, 8, 5, true)
	if e != nil {
		return "", errors.New("address conversion failed")
	}
	return bech32.Encode(hrp, words)
}
func DecodeAddress(address, expectedHRP string) ([]byte, error) {
	if expectedHRP == "" {
		return nil, errors.New("expected address HRP required")
	}
	hrp, words, e := bech32.Decode(address, 1023)
	if e != nil || hrp != expectedHRP {
		return nil, errors.New("invalid address checksum or HRP")
	}
	raw, e := bech32.ConvertBits(words, 5, 8, false)
	if e != nil || len(raw) == 0 || len(raw) > 255 {
		return nil, errors.New("invalid address bytes or padding")
	}
	return append([]byte(nil), raw...), nil
}
func EncodeAccountAddress(raw []byte) (string, error)     { return EncodeAddress(AccountHRP, raw) }
func DecodeAccountAddress(address string) ([]byte, error) { return DecodeAddress(address, AccountHRP) }
func HexToAccountAddress(address string) (string, error) {
	if !strings.HasPrefix(address, "0x") {
		return "", errors.New("hex address requires 0x prefix")
	}
	raw, e := hex.DecodeString(address[2:])
	if e != nil {
		return "", errors.New("invalid hex address")
	}
	return EncodeAccountAddress(raw)
}
func AccountAddressToHex(address string) (string, error) {
	raw, e := DecodeAccountAddress(address)
	if e != nil {
		return "", e
	}
	return "0x" + hex.EncodeToString(raw), nil
}

// ValidatorToAccount changes only the HRP of existing operator-address bytes,
// reproducing sdk.AccAddress(sdk.ValAddress(...)); no public-key hash is applied.
func ValidatorToAccount(address string) (string, error) {
	raw, e := DecodeAddress(address, ValidatorHRP)
	if e != nil {
		return "", e
	}
	return EncodeAccountAddress(raw)
}

// ReencodeAddress is the explicit arbitrary-source-HRP conversion used by the
// node's GetDecimalAddressFromBech32 helper. GetSigners uses strict HRPs instead.
func ReencodeAddress(address, targetHRP string) (string, error) {
	hrp, _, e := bech32.Decode(address, 1023)
	if e != nil {
		return "", errors.New("invalid source address")
	}
	raw, e := DecodeAddress(address, hrp)
	if e != nil {
		return "", e
	}
	return EncodeAddress(targetHRP, raw)
}
