package decimal

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type checkVector struct {
	Password       string `json:"password"`
	PrivateKey     string `json:"private_key"`
	Signer         string `json:"signer"`
	CheckHash      string `json:"check_hash"`
	Redeemer       string `json:"redeemer"`
	Packed         string `json:"packed"`
	MessageHash    string `json:"message_hash"`
	PersonalDigest string `json:"personal_digest"`
	Signature      string `json:"signature"`
	V              byte   `json:"v"`
}

func checkVectors(t *testing.T) []checkVector {
	t.Helper()
	raw, e := os.ReadFile("docs/checks-js-vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var data struct {
		Vectors []checkVector `json:"vectors"`
	}
	if e = json.Unmarshal(raw, &data); e != nil {
		t.Fatal(e)
	}
	if len(data.Vectors) != 3 {
		t.Fatal("missing executed reference cases")
	}
	return data.Vectors
}
func checkHex(t *testing.T, s string) []byte {
	t.Helper()
	b, e := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestEVMChecksMatchesExecutedOfficialJS(t *testing.T) {
	for i, v := range checkVectors(t) {
		t.Run(v.Signer, func(t *testing.T) {
			signer, e := EVMCheckSignerFromPassword(v.Password)
			if e != nil {
				t.Fatal(e)
			}
			defer signer.Close()
			if signer.Address() != common.HexToAddress(v.Signer) {
				t.Fatal("password/address reference mismatch", i)
			}
			h, to := common.HexToHash(v.CheckHash), common.HexToAddress(v.Redeemer)
			if !bytes.Equal(append(append([]byte{}, h[:]...), to[:]...), checkHex(t, v.Packed)) {
				t.Fatal("packed reference mismatch")
			}
			if EVMCheckRedemptionHash(h, to) != common.HexToHash(v.MessageHash) || EVMCheckRedemptionDigest(h, to) != common.HexToHash(v.PersonalDigest) {
				t.Fatal("digest reference mismatch")
			}
			signature, e := SignEVMCheckRedemption(context.Background(), signer, h, to)
			if e != nil || !bytes.Equal(signature, checkHex(t, v.Signature)) {
				t.Fatal("signature reference mismatch", e)
			}
			if signature[64] != v.V {
				t.Fatal("wrong recovery representation")
			}
			owner, e := RecoverEVMCheckRedemptionSigner(h, to, signature)
			if e != nil || owner != signer.Address() {
				t.Fatal("recover", e)
			}
			signature[64] -= 27
			if e = VerifyEVMCheckRedemption(h, to, owner, signature); e != nil {
				t.Fatal("v0/1 rejected", e)
			}
			normalized, e := SignEVMCheckRedemptionWith(context.Background(), h, to, owner, func(context.Context, common.Hash) ([]byte, error) { return signature, nil })
			if e != nil || !bytes.Equal(normalized, checkHex(t, v.Signature)) {
				t.Fatal("v normalization", e)
			}
			signature[0] ^= 1
			if !bytes.Equal(normalized, checkHex(t, v.Signature)) {
				t.Fatal("borrowed callback memory retained")
			}
		})
	}
}
func TestEVMChecksRejectsAlteredOrMalformedProofs(t *testing.T) {
	v := checkVectors(t)[0]
	h, to, owner := common.HexToHash(v.CheckHash), common.HexToAddress(v.Redeemer), common.HexToAddress(v.Signer)
	sig := checkHex(t, v.Signature)
	changed := h
	changed[0] ^= 1
	for _, pair := range []struct {
		h         common.Hash
		to, owner common.Address
	}{{changed, to, owner}, {h, common.HexToAddress("0x3000000000000000000000000000000000000003"), owner}, {h, to, common.Address{}}, {h, common.Address{}, owner}, {h, to, common.HexToAddress("0x4000000000000000000000000000000000000004")}} {
		if e := VerifyEVMCheckRedemption(pair.h, pair.to, pair.owner, sig); e == nil {
			t.Fatal("altered proof accepted")
		}
	}
	high := append([]byte{}, sig...)
	s := new(big.Int).SetBytes(high[32:64])
	s.Sub(crypto.S256().Params().N, s)
	s.FillBytes(high[32:64])
	high[64] = 27 + ((sig[64] - 27) ^ 1)
	badV := append([]byte{}, sig...)
	badV[64] = 29
	for _, bad := range [][]byte{nil, sig[:64], append(append([]byte{}, sig...), 0), high, badV, make([]byte, 65)} {
		if _, e := RecoverEVMCheckRedemptionSigner(h, to, bad); e == nil {
			t.Fatal("noncanonical proof recovered")
		}
	}
	signer, e := NewKeySigner(v.PrivateKey)
	if e != nil {
		t.Fatal(e)
	}
	defer signer.Close()
	for _, mode := range []string{"double-prefix", "hex-text", "wrong-key"} {
		_, e = SignEVMCheckRedemptionWith(context.Background(), h, to, owner, func(ctx context.Context, d common.Hash) ([]byte, error) {
			if mode == "double-prefix" {
				d = common.BytesToHash(accounts.TextHash(d[:]))
			}
			if mode == "hex-text" {
				d = common.BytesToHash(accounts.TextHash([]byte(EVMCheckRedemptionHash(h, to).Hex())))
			}
			if mode == "wrong-key" {
				other, _ := NewKeySigner(strings.Repeat("01", 32))
				defer other.Close()
				return other.SignDigest(ctx, d)
			}
			return signer.SignDigest(ctx, d)
		})
		if e == nil {
			t.Fatal("bad callback accepted", mode)
		}
	}
}
func TestEVMChecksContextAndPasswordBoundaries(t *testing.T) {
	v := checkVectors(t)[0]
	h, to, owner := common.HexToHash(v.CheckHash), common.HexToAddress(v.Redeemer), common.HexToAddress(v.Signer)
	for _, password := range []string{"", string([]byte{0xff, 0xfe})} {
		if s, e := EVMCheckSignerFromPassword(password); e == nil || s != nil {
			t.Fatal("invalid password accepted")
		}
	}
	nfc, _ := EVMCheckSignerFromPassword("\u00e9")
	nfd, _ := EVMCheckSignerFromPassword("e\u0301")
	defer nfc.Close()
	defer nfd.Close()
	if nfc.Address() == nfd.Address() {
		t.Fatal("password normalization introduced")
	}
	called := false
	callback := func(context.Context, common.Hash) ([]byte, error) {
		called = true
		return checkHex(t, v.Signature), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := SignEVMCheckRedemptionWith(ctx, h, to, owner, callback); !errors.Is(e, context.Canceled) || called {
		t.Fatal("cancel before callback", e)
	}
	if _, e := SignEVMCheckRedemptionWith(nil, h, to, owner, callback); e == nil {
		t.Fatal("nil context")
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	if _, e := SignEVMCheckRedemptionWith(ctx, h, to, owner, func(context.Context, common.Hash) ([]byte, error) {
		cancel()
		return checkHex(t, v.Signature), errors.New("callback failure")
	}); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel after callback", e)
	}
	wanted := errors.New("offline signer unavailable")
	if _, e := SignEVMCheckRedemptionWith(context.Background(), h, to, owner, func(context.Context, common.Hash) ([]byte, error) { return nil, wanted }); !errors.Is(e, wanted) {
		t.Fatal("callback error lost", e)
	}
	if _, e := SignEVMCheckRedemption(context.Background(), nil, h, to); e == nil {
		t.Fatal("nil signer")
	}
	s, _ := EVMCheckSignerFromPassword(v.Password)
	s.Close()
	if _, e := SignEVMCheckRedemption(context.Background(), s, h, to); e == nil {
		t.Fatal("closed signer")
	}
	if _, e := SignEVMCheckRedemptionWith(context.Background(), h, to, owner, nil); e == nil {
		t.Fatal("nil callback")
	}
	// Fixed-width pure hashing deliberately accepts zero values, as ABI packing does.
	if EVMCheckRedemptionHash(common.Hash{}, common.Address{}) != crypto.Keccak256Hash(make([]byte, 52)) {
		t.Fatal("zero hash policy")
	}
}
func TestEVMChecksBatchPreservesMappingAndOwnsCalldata(t *testing.T) {
	v := checkVectors(t)
	to := common.HexToAddress(v[0].Redeemer)
	offline := new(Client)
	contract, e := offline.Checks(common.HexToAddress("0x5000000000000000000000000000000000000005"))
	if e != nil {
		t.Fatal(e)
	}
	proofs := make([]EVMCheckRedemptionProof, 2)
	for i := range proofs {
		key, _ := EVMCheckSignerFromPassword(v[i].Password)
		defer key.Close()
		h := common.HexToHash(v[i].CheckHash)
		h[31] = byte(i + 10)
		sig, e := SignEVMCheckRedemption(context.Background(), key, h, to)
		if e != nil {
			t.Fatal(e)
		}
		sig[64] -= 27
		proofs[i] = EVMCheckRedemptionProof{CheckHash: h, Signer: key.Address(), Signature: sig}
	}
	call, e := contract.BuildSignedRedemption(to, proofs)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := ABIJSON("Checks")
	parsed, e := abi.JSON(bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	method, e := parsed.MethodById(call.Data[:4])
	if e != nil || method.Sig != "redeemChecks(bytes[],bytes32[])" {
		t.Fatal("wrong method", e)
	}
	values, e := method.Inputs.Unpack(call.Data[4:])
	if e != nil {
		t.Fatal(e)
	}
	signatures, hashes := values[0].([][]byte), values[1].([][32]byte)
	for i, p := range proofs {
		if hashes[i] != p.CheckHash || signatures[i][64] < 27 {
			t.Fatal("batch order/normalization")
		}
		if e = VerifyEVMCheckRedemption(p.CheckHash, to, p.Signer, signatures[i]); e != nil {
			t.Fatal(e)
		}
	}
	saved := append([]byte{}, call.Data...)
	proofs[0].Signature[0] ^= 1
	if !bytes.Equal(saved, call.Data) {
		t.Fatal("caller mutation changed intent")
	}
	if _, e = contract.BuildSignedRedemption(to, proofs); e == nil {
		t.Fatal("invalid signature accepted")
	}
	proofs[0].Signature[0] ^= 1
	duplicate := []EVMCheckRedemptionProof{proofs[0], proofs[0]}
	if _, e = contract.BuildSignedRedemption(to, duplicate); e == nil {
		t.Fatal("duplicate hash accepted")
	}
	swapped := append([]EVMCheckRedemptionProof{}, proofs...)
	swapped[0].Signature, swapped[1].Signature = swapped[1].Signature, swapped[0].Signature
	if _, e = contract.BuildSignedRedemption(to, swapped); e == nil {
		t.Fatal("swapped proof mapping accepted")
	}
	if _, e = contract.BuildSignedRedemption(to, nil); e == nil {
		t.Fatal("empty batch")
	}
	var nilContract *Checks
	if _, e = nilContract.BuildSignedRedemption(to, proofs); e == nil {
		t.Fatal("nil contract")
	}
}
