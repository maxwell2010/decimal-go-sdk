package decimal

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/btcsuite/btcd/btcutil/v2/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg/v2"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	bip39 "github.com/tyler-smith/go-bip39"
)

type Signer interface {
	Address() common.Address
	SignTransaction(context.Context, *types.Transaction, *big.Int) (*types.Transaction, error)
}

// KeySigner derives once and reuses the key. fmt and JSON never expose key material.
type KeySigner struct {
	mu      sync.RWMutex
	key     *ecdsa.PrivateKey
	address common.Address
}

func NewKeySigner(hexKey string) (*KeySigner, error) {
	if len(hexKey) >= 2 && hexKey[:2] == "0x" {
		hexKey = hexKey[2:]
	}
	key, e := crypto.HexToECDSA(hexKey)
	if e != nil {
		return nil, errors.New("invalid private key")
	}
	return &KeySigner{key: key, address: crypto.PubkeyToAddress(key.PublicKey)}, nil
}
func SignerFromMnemonic(words, passphrase, path string) (*KeySigner, error) {
	if !bip39.IsMnemonicValid(words) {
		return nil, errors.New("invalid BIP39 mnemonic")
	}
	if path == "" {
		path = "m/44'/60'/0'/0/0"
	}
	indexes, e := accounts.ParseDerivationPath(path)
	if e != nil {
		return nil, errors.New("invalid derivation path")
	}
	seed := bip39.NewSeed(words, passphrase)
	defer clear(seed)
	k, e := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if e != nil {
		return nil, errors.New("BIP32 derivation failed")
	}
	defer func() { k.Zero() }()
	for _, index := range indexes {
		child, e := k.Derive(index)
		k.Zero()
		if e != nil {
			return nil, errors.New("BIP32 child derivation failed")
		}
		k = child
	}
	priv, e := k.ECPrivKey()
	if e != nil {
		return nil, errors.New("private key unavailable")
	}
	defer priv.Zero()
	key := priv.ToECDSA()
	return &KeySigner{key: key, address: crypto.PubkeyToAddress(key.PublicKey)}, nil
}
func (s *KeySigner) Address() common.Address { return s.address }
func (s *KeySigner) Ready() error {
	if s == nil {
		return errors.New("signer required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.key == nil {
		return errors.New("signer closed")
	}
	return nil
}
func (s *KeySigner) String() string             { return "KeySigner(" + s.address.Hex() + ")" }
func (s *KeySigner) GoString() string           { return s.String() }
func (s *KeySigner) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(s.String())) }
func (s *KeySigner) SignTransaction(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if tx == nil || !uint256(chainID) || chainID.Sign() == 0 {
		return nil, errors.New("invalid transaction or chain ID")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.key == nil {
		return nil, errors.New("signer closed")
	}
	return types.SignTx(tx, types.LatestSignerForChainID(chainID), s.key)
}
func (s *KeySigner) SignDigest(ctx context.Context, digest common.Hash) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.key == nil {
		return nil, errors.New("signer closed")
	}
	return crypto.Sign(digest[:], s.key)
}
func (s *KeySigner) SignTypedData(ctx context.Context, data apitypes.TypedData) ([]byte, error) {
	h, _, e := apitypes.TypedDataAndHash(data)
	if e != nil {
		return nil, errors.New("invalid EIP712 typed data")
	}
	sig, e := s.SignDigest(ctx, common.BytesToHash(h))
	if e != nil {
		return nil, e
	}
	sig[64] += 27
	return sig, nil
}

// Close invalidates this signer. Go cannot guarantee erasure of all historical heap copies.
func (s *KeySigner) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.key != nil {
		s.key.D.SetInt64(0)
		s.key = nil
	}
}

type Permit struct {
	Owner, Spender         common.Address
	Value, Nonce, Deadline *big.Int
}

func PermitTypedData(name, version string, chainID *big.Int, token common.Address, p Permit) (apitypes.TypedData, error) {
	if name == "" || version == "" || token == (common.Address{}) || !uint256(chainID) || chainID.Sign() == 0 || !uint256(p.Value) || !uint256(p.Nonce) || !uint256(p.Deadline) {
		return apitypes.TypedData{}, errors.New("invalid permit domain or values")
	}
	return apitypes.TypedData{Types: apitypes.Types{
		"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}, {Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"}},
		"Permit":       {{Name: "owner", Type: "address"}, {Name: "spender", Type: "address"}, {Name: "value", Type: "uint256"}, {Name: "nonce", Type: "uint256"}, {Name: "deadline", Type: "uint256"}},
	}, PrimaryType: "Permit", Domain: apitypes.TypedDataDomain{Name: name, Version: version, ChainId: (*math.HexOrDecimal256)(copyInt(chainID)), VerifyingContract: token.Hex()}, Message: apitypes.TypedDataMessage{"owner": p.Owner.Hex(), "spender": p.Spender.Hex(), "value": p.Value.String(), "nonce": p.Nonce.String(), "deadline": p.Deadline.String()}}, nil
}

type Signature struct {
	R, S [32]byte
	V    uint8
}

func SplitSignature(sig []byte) (Signature, error) {
	if len(sig) != 65 {
		return Signature{}, errors.New("signature must have 65 bytes")
	}
	var s Signature
	copy(s.R[:], sig[:32])
	copy(s.S[:], sig[32:64])
	s.V = sig[64]
	if s.V < 27 {
		s.V += 27
	}
	if s.V != 27 && s.V != 28 {
		return Signature{}, errors.New("invalid recovery ID")
	}
	if !crypto.ValidateSignatureValues(s.V-27, new(big.Int).SetBytes(s.R[:]), new(big.Int).SetBytes(s.S[:]), true) {
		return Signature{}, errors.New("invalid ECDSA signature")
	}
	return s, nil
}
