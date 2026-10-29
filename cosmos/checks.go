package cosmos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"github.com/cosmos/btcutil/base58"
	"github.com/cosmos/btcutil/bech32"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"math/big"
	"regexp"
)

// LegacyCheck preserves the historical RLP field order. These helpers are not
// EVM Checks and do not imply current native ante admission for MsgRedeemCheck.
type LegacyCheck struct {
	ChainID  string
	Denom    string
	Amount   *big.Int
	Nonce    []byte
	DueBlock uint64
	Lock     []byte
	V, R, S  *big.Int
}

var denomPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9/:._-]{2,127}$`)

func (c LegacyCheck) payloadValid() bool {
	return c.ChainID != "" && len(c.ChainID) <= 256 && denomPattern.MatchString(c.Denom) && c.Amount != nil && c.Amount.Sign() > 0 && c.Amount.BitLen() <= 256 && len(c.Nonce) <= 32
}
func (c LegacyCheck) hash(lock bool) ([32]byte, error) {
	items := []any{c.ChainID, c.Denom, c.Amount, c.Nonce, c.DueBlock}
	if lock {
		items = append(items, c.Lock)
	}
	b, e := rlp.EncodeToBytes(items)
	if e != nil {
		return [32]byte{}, errors.New("invalid historical check payload")
	}
	return [32]byte(crypto.Keccak256Hash(b)), nil
}
func (c LegacyCheck) signature() ([]byte, error) {
	if c.V == nil || c.R == nil || c.S == nil || !c.V.IsUint64() || c.V.Uint64() < 27 || c.V.Uint64() > 28 {
		return nil, errors.New("invalid check signature")
	}
	v := byte(c.V.Uint64() - 27)
	if !crypto.ValidateSignatureValues(v, c.R, c.S, true) {
		return nil, errors.New("invalid check signature")
	}
	sig := make([]byte, 65)
	c.R.FillBytes(sig[:32])
	c.S.FillBytes(sig[32:64])
	sig[64] = v
	return sig, nil
}
func (c LegacyCheck) Sender() ([]byte, error) {
	if !c.payloadValid() {
		return nil, errors.New("invalid check payload")
	}
	sig, e := c.signature()
	if e != nil {
		return nil, e
	}
	h, e := c.hash(true)
	if e != nil {
		return nil, e
	}
	pub, e := crypto.SigToPub(h[:], sig)
	if e != nil {
		return nil, errors.New("invalid issuer signature")
	}
	return crypto.PubkeyToAddress(*pub).Bytes(), nil
}
func (c LegacyCheck) lockPublicKey() ([]byte, error) {
	if len(c.Lock) != 65 || c.Lock[64] > 1 {
		return nil, errors.New("invalid lock signature")
	}
	h, e := c.hash(false)
	if e != nil {
		return nil, e
	}
	pub, e := crypto.Ecrecover(h[:], c.Lock)
	if e != nil {
		return nil, errors.New("invalid lock signature")
	}
	return pub, nil
}
func ParseCheck(encoded string) (LegacyCheck, error) {
	if len(encoded) == 0 || len(encoded) > 6000 {
		return LegacyCheck{}, errors.New("invalid historical check size")
	}
	b := base58.Decode(encoded)
	if len(b) == 0 || len(b) > 4096 || base58.Encode(b) != encoded {
		return LegacyCheck{}, errors.New("invalid check base58")
	}
	var c LegacyCheck
	if e := rlp.DecodeBytes(b, &c); e != nil || !c.payloadValid() {
		return LegacyCheck{}, errors.New("invalid check RLP/payload")
	}
	if _, e := c.Sender(); e != nil {
		return LegacyCheck{}, e
	}
	if _, e := c.lockPublicKey(); e != nil {
		return LegacyCheck{}, e
	}
	return c, nil
}

// IssueCheck creates an archival/historical Base58 RLP check using the exact
// node sdk/tx/checks.go hash layout. Callback signatures are bound to issuerKey.
// Passphrase bytes are copied and the temporary hash/private scalar is cleared;
// caller-owned input is unchanged. Go cannot guarantee historical heap erasure.
func IssueCheck(ctx context.Context, chainID, denom string, amount, nonce *big.Int, dueBlock uint64, passphrase, issuerKey []byte, sign DigestSigner) (string, error) {
	if e := ctx.Err(); e != nil {
		return "", e
	}
	if amount == nil || nonce == nil || nonce.Sign() < 0 || nonce.BitLen() > 256 || sign == nil {
		return "", errors.New("invalid check issue inputs")
	}
	c := LegacyCheck{ChainID: chainID, Denom: denom, Amount: new(big.Int).Set(amount), Nonce: append([]byte(nil), nonce.Bytes()...), DueBlock: dueBlock}
	if !c.payloadValid() {
		return "", errors.New("invalid check issue payload")
	}
	pub, e := crypto.DecompressPubkey(issuerKey)
	if e != nil {
		return "", errors.New("invalid issuer key")
	}
	expected := crypto.CompressPubkey(pub)
	secret := append([]byte(nil), passphrase...)
	defer clear(secret)
	seed := sha256.Sum256(secret)
	defer clear(seed[:])
	key, e := crypto.ToECDSA(seed[:])
	if e != nil {
		return "", errors.New("invalid lock key")
	}
	defer key.D.SetInt64(0)
	h, e := c.hash(false)
	if e != nil {
		return "", e
	}
	c.Lock, e = crypto.Sign(h[:], key)
	if e != nil {
		return "", errors.New("cannot sign lock")
	}
	h, e = c.hash(true)
	if e != nil {
		return "", e
	}
	sig, e := sign(ctx, h)
	if e != nil {
		return "", errors.New("issuer signing failed")
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	if len(sig) != 65 || sig[64] > 1 || !crypto.VerifySignature(expected, h[:], sig[:64]) {
		return "", errors.New("issuer signature mismatch")
	}
	recovered, e := crypto.SigToPub(h[:], sig)
	if e != nil || !bytes.Equal(expected, crypto.CompressPubkey(recovered)) {
		return "", errors.New("issuer recovery mismatch")
	}
	c.R = new(big.Int).SetBytes(sig[:32])
	c.S = new(big.Int).SetBytes(sig[32:64])
	c.V = new(big.Int).SetUint64(uint64(sig[64]) + 27)
	raw, e := rlp.EncodeToBytes(c)
	if e != nil {
		return "", errors.New("cannot encode check")
	}
	return base58.Encode(raw), nil
}

// CreateRedeemCheck builds the historical Cosmos message; it never sends it.
// It verifies chain, expiry and lock/passphrase before generating receiver proof.
func CreateRedeemCheck(ctx context.Context, encoded, chainID string, height uint64, receiver, passphrase []byte) (*dynamicpb.Message, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if len(receiver) != 20 {
		return nil, errors.New("receiver must contain20 address bytes")
	}
	receiver = append([]byte(nil), receiver...)
	c, e := ParseCheck(encoded)
	if e != nil {
		return nil, e
	}
	if c.ChainID != chainID {
		return nil, errors.New("check chain mismatch")
	}
	if c.DueBlock < height {
		return nil, errors.New("check expired")
	}
	secret := append([]byte(nil), passphrase...)
	defer clear(secret)
	seed := sha256.Sum256(secret)
	defer clear(seed[:])
	key, e := crypto.ToECDSA(seed[:])
	if e != nil {
		return nil, errors.New("invalid lock key")
	}
	defer key.D.SetInt64(0)
	lock, e := c.lockPublicKey()
	if e != nil {
		return nil, e
	}
	if !bytes.Equal(lock, crypto.FromECDSAPub(&key.PublicKey)) {
		return nil, errors.New("wrong check passphrase")
	}
	raw, e := rlp.EncodeToBytes([]any{receiver})
	if e != nil {
		return nil, e
	}
	digest := crypto.Keccak256Hash(raw)
	proof, e := crypto.Sign(digest[:], key)
	if e != nil {
		return nil, errors.New("cannot sign redemption proof")
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	sender, e := bech32.EncodeFromBase256("d0", receiver)
	if e != nil {
		return nil, e
	}
	m := NewMsgRedeemCheck()
	set(m, "sender", protoreflect.ValueOfString(sender))
	set(m, "check", protoreflect.ValueOfString(encoded))
	set(m, "proof", protoreflect.ValueOfString(base64.StdEncoding.EncodeToString(proof)))
	return m, nil
}
