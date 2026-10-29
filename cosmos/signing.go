package cosmos

import "golang.org/x/crypto/sha3"

// EthermintSigningDigest reproduces PrivKey.Sign in Decimal Ethermint v0.22.13:
// input of length32 is already a digest; other inputs are Keccak256 hashed.
// It does not perform signing or accept a private key.
func EthermintSigningDigest(signBytes []byte) [32]byte {
	var result [32]byte
	if len(signBytes) == 32 {
		copy(result[:], signBytes)
		return result
	}
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(signBytes)
	h.Sum(result[:0])
	return result
}
