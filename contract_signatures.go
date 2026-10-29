package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type ContractSignatureInterface uint8

const (
	ERC1271Bytes32 ContractSignatureInterface = iota
	SafeLegacyBytes
)

// ContractSignatureVerifier explicitly selects one interface from supplied ABI.
// Legacy requires the exact preimage, whose hash must equal the requested digest.
// It never probes or falls back to another signature convention.
func (c *Client) ContractSignatureVerifier(raw json.RawMessage, block *big.Int, kind ContractSignatureInterface, preimage []byte) (SafeContractVerifier, error) {
	return c.ContractSignatureVerifierFrom(raw, block, kind, preimage, common.Address{})
}

// ContractSignatureVerifierFrom preserves an explicit caller (normally the
// target Safe) for caller-dependent ERC1271 wallet policies.
func (c *Client) ContractSignatureVerifierFrom(raw json.RawMessage, block *big.Int, kind ContractSignatureInterface, preimage []byte, caller common.Address) (SafeContractVerifier, error) {
	raw = append(json.RawMessage(nil), raw...)
	if c == nil || !uint256(block) {
		return nil, errors.New("client and signature snapshot block required")
	}
	signature := "isValidSignature(bytes32,bytes)"
	magic := [4]byte{0x16, 0x26, 0xba, 0x7e}
	if kind == SafeLegacyBytes {
		signature = "isValidSignature(bytes,bytes)"
		magic = [4]byte{0x20, 0xc1, 0x3b, 0x0b}
		if len(preimage) == 0 {
			return nil, errors.New("legacy signing preimage required")
		}
	} else if kind != ERC1271Bytes32 {
		return nil, errors.New("unsupported contract signature interface")
	}
	// Parse/validate once, without assuming a deployment address.
	parsed, e := c.ContractWithABI(common.HexToAddress("0x01"), raw)
	if e != nil {
		return nil, e
	}
	a, e := parsed.contractABI()
	if e != nil {
		return nil, e
	}
	_, m, e := method(a, signature)
	if e != nil || !m.IsConstant() || len(m.Outputs) != 1 || m.Outputs[0].Type.String() != "bytes4" {
		return nil, errors.New("explicit ABI lacks selected signature interface")
	}
	snapshot := copyInt(block)
	data := append([]byte(nil), preimage...)
	return func(ctx context.Context, owner common.Address, hash common.Hash, sig []byte) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		contract, e := c.ContractWithABI(owner, raw)
		if e != nil {
			return e
		}
		code, e := c.ContractCode(ctx, owner, snapshot)
		if e != nil || len(code) == 0 {
			return errors.New("signature owner has no code at snapshot")
		}
		var argument any = [32]byte(hash)
		if kind == SafeLegacyBytes {
			if crypto.Keccak256Hash(data) != hash {
				return errors.New("legacy preimage/hash mismatch")
			}
			argument = append([]byte(nil), data...)
		}
		result, e := contract.ReadFromAt(ctx, caller, snapshot, signature, argument, append([]byte(nil), sig...))
		if e != nil {
			return e
		}
		if len(result) != 1 {
			return errors.New("invalid contract signature response")
		}
		got, ok := result[0].([4]byte)
		if !ok || got != magic {
			return errors.New("contract signature magic mismatch")
		}
		return nil
	}, nil
}
