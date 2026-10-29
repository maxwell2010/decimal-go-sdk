package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// BuildNamed resolves an ABI overload by successful type checking. Ambiguous
// overloads require Build with an explicit full signature.
func (c *Contract) BuildNamed(name string, value *big.Int, args ...any) (Call, error) {
	a, e := c.contractABI()
	if e != nil {
		return Call{}, e
	}
	var result Call
	matches := 0
	for _, m := range a.Methods {
		if m.RawName != name || m.IsConstant() {
			continue
		}
		call, e := c.Build(m.Sig, value, args...)
		if e == nil {
			result = call
			matches++
		}
	}
	if matches != 1 {
		return Call{}, errors.New("method absent, invalid arguments or ambiguous overload; provide exact deployment ABI/signature")
	}
	return result, nil
}

// These workflows intentionally accept an explicit deployment Contract because
// go-smart-node has no pinned bridge-v2, GasCenter, reserveless-token or Safe ABI.
// Integer widths are taken from that supplied ABI, not guessed from JS numbers.
func (c *Contract) BuildBridgeTransferNative(amount, fee *big.Int, recipient [32]byte, toChain uint16, nonce *big.Int) (Call, error) {
	if !uint256(amount) || !uint256(fee) {
		return Call{}, errors.New("invalid bridge amount or fee")
	}
	value := new(big.Int).Add(amount, fee)
	return c.BuildNamed("wrapAndTransferETH", value, recipient, toChain, nonce, fee)
}
func (c *Contract) BuildBridgeTransferToken(token common.Address, amount, fee *big.Int, recipient [32]byte, toChain uint16, nonce *big.Int) (Call, error) {
	if !uint256(amount) || !uint256(fee) {
		return Call{}, errors.New("invalid bridge amount or fee")
	}
	return c.BuildNamed("transferTokens", fee, token, recipient, amount, toChain, nonce)
}
func (c *Contract) BuildBridgeCompleteTransfer(encodedVM []byte, unwrap bool) (Call, error) {
	if len(encodedVM) == 0 {
		return Call{}, errors.New("bridge proof required")
	}
	return c.BuildNamed("completeTransfer", nil, encodedVM, unwrap)
}
func (c *Contract) BuildReservelessMint(recipient common.Address, amount *big.Int) (Call, error) {
	return c.BuildNamed("mint", nil, recipient, amount)
}
func (c *Contract) BuildGasConversion(owner, token common.Address, amount, estimatedGas, deadline *big.Int, sig Signature) (Call, error) {
	return c.BuildNamed("convertToDEL", nil, owner, token, amount, estimatedGas, deadline, sig.V, sig.R, sig.S)
}

// NFTPermitTypedData follows the official JS DRC721/DRC1155 permit schemas.
// DRC721 uses tokenID; DRC1155 passes nil. Verify DOMAIN_SEPARATOR and nonce
// on the deployment before signing, as CheckedPermit does for ERC20.
func NFTPermitTypedData(name, version string, chainID *big.Int, collection, spender common.Address, tokenID, nonce, deadline *big.Int) (apitypes.TypedData, error) {
	if name == "" || version == "" || !uint256(chainID) || chainID.Sign() == 0 || collection == (common.Address{}) || !uint256(nonce) || !uint256(deadline) || (tokenID != nil && !uint256(tokenID)) {
		return apitypes.TypedData{}, errors.New("invalid NFT permit")
	}
	fields := []apitypes.Type{{Name: "spender", Type: "address"}}
	message := apitypes.TypedDataMessage{"spender": spender.Hex(), "nonce": nonce.String(), "deadline": deadline.String()}
	if tokenID != nil {
		fields = append(fields, apitypes.Type{Name: "tokenId", Type: "uint256"})
		message["tokenId"] = tokenID.String()
	}
	fields = append(fields, apitypes.Type{Name: "nonce", Type: "uint256"}, apitypes.Type{Name: "deadline", Type: "uint256"})
	return apitypes.TypedData{Types: apitypes.Types{"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}, {Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"}}, "Permit": fields}, PrimaryType: "Permit", Domain: apitypes.TypedDataDomain{Name: name, Version: version, ChainId: (*math.HexOrDecimal256)(copyInt(chainID)), VerifyingContract: collection.Hex()}, Message: message}, nil
}

// Simulate executes the exact write calldata without signing or sending it.
func (c *Client) Simulate(ctx context.Context, from common.Address, call Call, block *big.Int) ([]byte, error) {
	if !uint256(call.Value) {
		return nil, errors.New("invalid value")
	}
	// Capture intent before invoking any backend callback.
	call.Value = copyInt(call.Value)
	call.Data = append([]byte(nil), call.Data...)
	if block != nil {
		block = new(big.Int).Set(block)
	}
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	var e error
	block, e = c.simulationDeploymentBlock(ctx, call.To, call.deployment, block)
	if e != nil {
		return nil, e
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	result, e := c.rpc.CallContract(ctx, callMessage(from, call), block)
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if e != nil {
		a, _ := loadABI(call.Profile)
		return nil, decodeRPCError(e, a)
	}
	if result != nil {
		owned := make([]byte, len(result))
		copy(owned, result)
		result = owned
	}
	return result, nil
}
func callMessage(from common.Address, call Call) ethereum.CallMsg {
	return ethereum.CallMsg{From: from, To: &call.To, Value: copyInt(call.Value), Data: append([]byte(nil), call.Data...)}
}

// Simulate preserves deployment-specific custom errors as well as the exact
// target and snapshot. Unlike Client.Simulate, it has the explicit ABI available.
func (c *Contract) Simulate(ctx context.Context, from common.Address, call Call, block *big.Int) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("client required for simulation")
	}
	if call.To != c.Address || !uint256(call.Value) {
		return nil, errors.New("simulation target or value mismatch")
	}
	// Capture intent before invoking any backend callback.
	call.Value = copyInt(call.Value)
	call.Data = append([]byte(nil), call.Data...)
	if block != nil {
		block = new(big.Int).Set(block)
	}
	if e := c.client.checkChain(ctx); e != nil {
		return nil, e
	}
	var e error
	// A bound Contract retains its guard even if the caller supplies a fresh Call.
	block, e = c.client.simulationDeploymentBlock(ctx, call.To, c.deployment, block)
	if e != nil {
		return nil, e
	}
	block, e = c.client.simulationDeploymentBlock(ctx, call.To, call.deployment, block)
	if e != nil {
		return nil, e
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	returnData, e := c.client.rpc.CallContract(ctx, callMessage(from, call), block)
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if e != nil {
		a, ae := c.contractABI()
		if ae != nil {
			return nil, ae
		}
		return nil, decodeRPCError(e, a)
	}
	if returnData != nil {
		owned := make([]byte, len(returnData))
		copy(owned, returnData)
		returnData = owned
	}
	return returnData, nil
}

// CheckedNFTPermit checks ownership (DRC721), nonce and domain separator before
// signing either official NFT permit schema. A nil tokenID selects DRC1155.
func (c *Client) CheckedNFTPermit(ctx context.Context, signer *KeySigner, collection *Contract, name, version string, spender common.Address, tokenID, nonce, deadline *big.Int) (Signature, error) {
	if signer == nil || collection == nil || collection.client != c {
		return Signature{}, errors.New("signer and collection on this client required")
	}
	if e := signer.Ready(); e != nil {
		return Signature{}, e
	}
	d, e := NFTPermitTypedData(name, version, c.cfg.ChainID, collection.Address, spender, tokenID, nonce, deadline)
	if e != nil {
		return Signature{}, e
	}
	var values []any
	if tokenID != nil {
		owner, e := collection.Read(ctx, "ownerOf(uint256)", tokenID)
		if e != nil {
			return Signature{}, e
		}
		if len(owner) != 1 || owner[0] != signer.Address() {
			return Signature{}, errors.New("NFT permit signer is not token owner")
		}
		values, e = collection.Read(ctx, "nonces(uint256)", tokenID)
	} else {
		values, e = collection.Read(ctx, "nonces(address)", signer.Address())
	}
	if e != nil {
		return Signature{}, e
	}
	if len(values) != 1 {
		return Signature{}, errors.New("invalid NFT nonce response")
	}
	actual, ok := values[0].(*big.Int)
	if !ok || actual.Cmp(nonce) != 0 {
		return Signature{}, errors.New("NFT permit nonce mismatch")
	}
	domain, e := collection.Read(ctx, "DOMAIN_SEPARATOR()")
	if e != nil {
		return Signature{}, e
	}
	expected, e := typedDomainHash(d)
	if e != nil {
		return Signature{}, e
	}
	if len(domain) != 1 || domain[0] != [32]byte(expected) {
		return Signature{}, errors.New("NFT permit domain mismatch")
	}
	sig, e := signer.SignTypedData(ctx, d)
	if e != nil {
		return Signature{}, e
	}
	return SplitSignature(sig)
}

// simulationDeploymentBlock checks bound identity before executing a simulation
// and pins the call to the very block used for the identity check.
func (c *Client) simulationDeploymentBlock(ctx context.Context, target common.Address, evidence *DeploymentEvidence, block *big.Int) (*big.Int, error) {
	if evidence == nil {
		return block, nil
	}
	if target != evidence.Proxy {
		return nil, errors.New("proxy simulation target changed")
	}
	var height uint64
	if block == nil {
		var e error
		height, e = c.BlockNumber(ctx)
		if e != nil {
			return nil, e
		}
	} else {
		if block.Sign() < 0 || !block.IsUint64() {
			return nil, errors.New("explicit nonnegative proxy snapshot required")
		}
		height = block.Uint64()
	}
	if e := c.verifyDeployment(ctx, *evidence, height); e != nil {
		return nil, e
	}
	return new(big.Int).SetUint64(height), nil
}
