package decimal

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EVMCheckSignerFromPassword follows the official JS Checks password derivation:
// Keccak256 of the exact nonempty UTF-8 string, without Unicode normalization.
// This is a check-secret signer, not an application account derivation scheme.
// Close the returned signer after use. Caller-owned string copies cannot be erased.
func EVMCheckSignerFromPassword(password string) (*KeySigner, error) {
	if password == "" || !utf8.ValidString(password) {
		return nil, errors.New("check password must be a nonempty valid UTF-8 string")
	}
	input := []byte(password)
	defer clear(input)
	material := crypto.Keccak256(input)
	defer clear(material)
	key, err := crypto.ToECDSA(material)
	if err != nil {
		return nil, errors.New("check password does not derive a valid signing key")
	}
	return &KeySigner{key: key, address: crypto.PubkeyToAddress(key.PublicKey)}, nil
}

// EVMCheckRedemptionHash is Solidity keccak256(abi.encodePacked(checkHash, redeemer)):
// exactly 32 hash bytes followed by 20 address bytes. checkHash is opaque contract
// output; this helper does not derive, issue or look up a check.
func EVMCheckRedemptionHash(checkHash common.Hash, redeemer common.Address) common.Hash {
	return crypto.Keccak256Hash(checkHash[:], redeemer[:])
}

// EVMCheckRedemptionDigest applies the EIP-191 personal-message prefix to the 32
// hash bytes, matching ethers Wallet.signMessage(arrayify(messageHash)). It does
// not prefix the hexadecimal text and does not include an EIP-712 domain.
// This is the JS source profile; the bundled historical Checks bytecode uses a
// different digest. Deployment compatibility must be established separately.
func EVMCheckRedemptionDigest(checkHash common.Hash, redeemer common.Address) common.Hash {
	hash := EVMCheckRedemptionHash(checkHash, redeemer)
	return common.BytesToHash(accounts.TextHash(hash[:]))
}

// RecoverEVMCheckRedemptionSigner accepts canonical low-S 65-byte signatures,
// with either 0/1 or 27/28 recovery IDs. It verifies only the mathematical proof,
// not check existence, funding, expiry, redemption status or deployment identity.
func RecoverEVMCheckRedemptionSigner(checkHash common.Hash, redeemer common.Address, signature []byte) (common.Address, error) {
	_, owner, err := normalizeEVMCheckSignature(checkHash, redeemer, signature)
	return owner, err
}

func normalizeEVMCheckSignature(checkHash common.Hash, redeemer common.Address, signature []byte) ([]byte, common.Address, error) {
	if len(signature) != 65 {
		return nil, common.Address{}, errors.New("signature must have 65 bytes")
	}
	raw := append([]byte(nil), signature...)
	parts, err := SplitSignature(raw)
	if err != nil {
		return nil, common.Address{}, err
	}
	raw[64] = parts.V - 27
	digest := EVMCheckRedemptionDigest(checkHash, redeemer)
	pub, err := crypto.SigToPub(digest[:], raw)
	if err != nil {
		return nil, common.Address{}, errors.New("cannot recover EVM check signer")
	}
	raw[64] = parts.V
	return raw, crypto.PubkeyToAddress(*pub), nil
}

// VerifyEVMCheckRedemption binds a proof to the explicitly supplied check signer
// and redeemer. Nonzero signer/redeemer are SDK policy, not inferred contract
// validation rules. Pure hash/digest/recovery helpers accept all fixed-width values.
func VerifyEVMCheckRedemption(checkHash common.Hash, redeemer, expectedSigner common.Address, signature []byte) error {
	if redeemer == (common.Address{}) || expectedSigner == (common.Address{}) {
		return errors.New("nonzero check redeemer and expected signer required")
	}
	_, recovered, err := normalizeEVMCheckSignature(checkHash, redeemer, signature)
	if err != nil {
		return err
	}
	if recovered != expectedSigner {
		return errors.New("EVM check signature does not match the expected signer")
	}
	return nil
}

// SignEVMCheckRedemptionWith passes the final EIP-191 digest to a signing callback.
// The callback must sign that digest directly; applying personal_sign again would
// double-prefix it. The result is owner-verified, copied and normalized to v=27/28.
// Neither chain ID, contract, amount, nonce nor expiry is added to the official
// payload. A signature alone cannot establish the check's chain or contract.
func SignEVMCheckRedemptionWith(ctx context.Context, checkHash common.Hash, redeemer, expectedSigner common.Address, sign func(context.Context, common.Hash) ([]byte, error)) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if redeemer == (common.Address{}) || expectedSigner == (common.Address{}) || sign == nil {
		return nil, errors.New("nonzero check redeemer, expected signer and signing callback required")
	}
	digest := EVMCheckRedemptionDigest(checkHash, redeemer)
	signature, err := sign(ctx, digest)
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if err != nil {
		return nil, err
	}
	normalized, recovered, err := normalizeEVMCheckSignature(checkHash, redeemer, signature)
	if err != nil {
		return nil, err
	}
	if recovered != expectedSigner {
		return nil, errors.New("EVM check callback signed with another key")
	}
	return normalized, nil
}

// SignEVMCheckRedemption signs a recipient-bound EVM Checks proof offline.
// For a direct Checks call, use its transaction sender as redeemer.
func SignEVMCheckRedemption(ctx context.Context, signer *KeySigner, checkHash common.Hash, redeemer common.Address) ([]byte, error) {
	if signer == nil {
		return nil, errors.New("check signer required")
	}
	return SignEVMCheckRedemptionWith(ctx, checkHash, redeemer, signer.Address(), signer.SignDigest)
}

// EVMCheckRedemptionProof pairs an opaque check identity with its declared check
// signer and proof. Signer is not implicitly the creator or transaction sender.
type EVMCheckRedemptionProof struct {
	CheckHash common.Hash
	Signer    common.Address
	Signature []byte
}

// BuildSignedRedemption verifies all proofs and builds redeemChecks offline in
// the supplied order. Duplicate hashes are rejected as SDK policy. This Call
// does not carry a From restriction: a direct call's sender must equal redeemer.
// No RPC, signature generation, check-state query or broadcast is performed.
func (c *Checks) BuildSignedRedemption(redeemer common.Address, proofs []EVMCheckRedemptionProof) (Call, error) {
	if c == nil || c.Contract == nil || redeemer == (common.Address{}) || len(proofs) == 0 {
		return Call{}, errors.New("Checks contract, nonzero redeemer and proofs required")
	}
	// Snapshot every caller-owned signature before inspecting the batch.
	snapshot := make([]EVMCheckRedemptionProof, len(proofs))
	for i, proof := range proofs {
		if len(proof.Signature) != 65 {
			return Call{}, errors.New("signature must have 65 bytes")
		}
		snapshot[i] = proof
		snapshot[i].Signature = append([]byte(nil), proof.Signature...)
	}
	signatures := make([][]byte, len(snapshot))
	hashes := make([][32]byte, len(snapshot))
	seen := make(map[common.Hash]bool, len(snapshot))
	for i, proof := range snapshot {
		if seen[proof.CheckHash] {
			return Call{}, errors.New("duplicate EVM check hash")
		}
		seen[proof.CheckHash] = true
		if proof.Signer == (common.Address{}) {
			return Call{}, errors.New("nonzero expected check signer required")
		}
		normalized, recovered, err := normalizeEVMCheckSignature(proof.CheckHash, redeemer, proof.Signature)
		if err != nil {
			return Call{}, err
		}
		if recovered != proof.Signer {
			return Call{}, errors.New("EVM check signature does not match the expected signer")
		}
		signatures[i] = normalized
		hashes[i] = proof.CheckHash
	}
	return c.BuildRedeemChecks(signatures, hashes)
}
