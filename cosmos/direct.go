package cosmos

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
)

type DigestSigner func(context.Context, [32]byte) ([]byte, error)
type DirectSigner struct {
	PublicKey               []byte
	AccountNumber, Sequence uint64
	Sign                    DigestSigner
}
type DirectTx struct {
	Body        []byte
	ChainID     string
	Fee         []Coin
	Gas         uint64
	SignerOrder [][]byte
	Signers     []DirectSigner
}
type SignedDirectTx struct {
	Raw, Body, AuthInfo []byte
	Hash                [32]byte
}

// SignDirectTx snapshots the transaction before callbacks, orders AuthInfo and
// signatures by the explicitly declared message signer order, and verifies each
// recoverable signature against its declared compressed Ethermint public key.
// SignerOrder must come from the messages' required signers, not lexicographic
// sorting of addresses. A node's business validation remains authoritative.
func SignDirectTx(ctx context.Context, tx DirectTx) (SignedDirectTx, error) {
	if e := ctx.Err(); e != nil {
		return SignedDirectTx{}, e
	}
	body := append([]byte(nil), tx.Body...)
	chain := tx.ChainID
	fee := append([]Coin(nil), tx.Fee...)
	gas := tx.Gas
	if len(tx.SignerOrder) == 0 || len(tx.Signers) != len(tx.SignerOrder) {
		return SignedDirectTx{}, errors.New("explicit signer order/count required")
	}
	byKey := map[string]DirectSigner{}
	for _, s := range tx.Signers {
		pub, e := crypto.DecompressPubkey(s.PublicKey)
		if e != nil || s.Sign == nil {
			return SignedDirectTx{}, errors.New("valid compressed key and signing callback required")
		}
		key := string(crypto.CompressPubkey(pub))
		if _, ok := byKey[key]; ok {
			return SignedDirectTx{}, errors.New("duplicate signer")
		}
		s.PublicKey = append([]byte(nil), s.PublicKey...)
		byKey[key] = s
	}
	ordered := make([]DirectSigner, len(tx.SignerOrder))
	info := make([]SignerInfo, len(ordered))
	used := map[string]bool{}
	for i, key := range tx.SignerOrder {
		s, ok := byKey[string(key)]
		if !ok || used[string(key)] {
			return SignedDirectTx{}, errors.New("invalid required signer order")
		}
		used[string(key)] = true
		ordered[i] = s
		info[i] = SignerInfo{s.PublicKey, s.Sequence}
	}
	auth, e := EncodeAuthInfo(info, fee, gas)
	if e != nil {
		return SignedDirectTx{}, e
	}
	signatures := make([][]byte, len(ordered))
	for i, s := range ordered {
		if e = ctx.Err(); e != nil {
			return SignedDirectTx{}, e
		}
		doc, e := EncodeSignDoc(body, auth, chain, s.AccountNumber)
		if e != nil {
			return SignedDirectTx{}, e
		}
		digest := EthermintSigningDigest(doc)
		sig, e := s.Sign(ctx, digest)
		if e != nil {
			return SignedDirectTx{}, errors.New("Cosmos signing callback failed")
		}
		if e = ctx.Err(); e != nil {
			return SignedDirectTx{}, e
		}
		if len(sig) != 65 || sig[64] > 1 {
			return SignedDirectTx{}, errors.New("invalid Ethermint recovery signature")
		}
		sig = append([]byte(nil), sig...)
		pub, e := crypto.SigToPub(digest[:], sig)
		if e != nil || !bytes.Equal(crypto.CompressPubkey(pub), s.PublicKey) || !crypto.VerifySignature(s.PublicKey, digest[:], sig[:64]) {
			return SignedDirectTx{}, errors.New("signature does not match declared public key")
		}
		signatures[i] = sig
	}
	raw, e := EncodeTxRaw(body, auth, signatures)
	if e != nil {
		return SignedDirectTx{}, e
	}
	return SignedDirectTx{Raw: raw, Body: body, AuthInfo: auth, Hash: TransactionHash(raw)}, nil
}
