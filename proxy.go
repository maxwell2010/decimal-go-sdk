package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

var implementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
var beaconSlot = common.HexToHash("0xa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d50")

type DeploymentEvidence struct {
	ChainID                                        string
	Block                                          uint64
	Proxy, Implementation                          common.Address
	ProxyCodeHash, ImplementationCodeHash, ABIHash common.Hash
	Beacon                                         common.Address
	BeaconCodeHash                                 common.Hash
}
type ProxyDeployment struct {
	Contract *Contract
	Evidence DeploymentEvidence
}
type storageReader interface {
	StorageAt(context.Context, common.Address, common.Hash, *big.Int) ([]byte, error)
}

func (c *Client) proxyEvidence(ctx context.Context, address common.Address, block uint64) (DeploymentEvidence, error) {
	if e := c.checkChain(ctx); e != nil {
		return DeploymentEvidence{}, e
	}
	storage, ok := c.rpc.(storageReader)
	if !ok {
		return DeploymentEvidence{}, errors.New("backend does not support EIP1967 storage reads")
	}
	height := new(big.Int).SetUint64(block)
	proxy, e := c.rpc.CodeAt(ctx, address, height)
	if e != nil || len(proxy) == 0 {
		return DeploymentEvidence{}, rpcFailure(ctx, "proxy code unavailable")
	}
	slot, e := storage.StorageAt(ctx, address, implementationSlot, height)
	if e != nil {
		return DeploymentEvidence{}, rpcFailure(ctx, "implementation slot unavailable")
	}
	if len(slot) != 32 {
		return DeploymentEvidence{}, errors.New("invalid EIP1967 slot length")
	}
	for _, b := range slot[:12] {
		if b != 0 {
			return DeploymentEvidence{}, errors.New("noncanonical EIP1967 address")
		}
	}
	implementation := common.BytesToAddress(slot[12:])
	if implementation == (common.Address{}) || implementation == address {
		return DeploymentEvidence{}, errors.New("missing or circular EIP1967 implementation")
	}
	code, e := c.rpc.CodeAt(ctx, implementation, height)
	if e != nil || len(code) == 0 {
		return DeploymentEvidence{}, rpcFailure(ctx, "implementation code unavailable")
	}
	return DeploymentEvidence{ChainID: c.ChainID().String(), Block: block, Proxy: address, Implementation: implementation, ProxyCodeHash: crypto.Keccak256Hash(proxy), ImplementationCodeHash: crypto.Keccak256Hash(code)}, nil
}

// LoadProxyDeploymentContract explicitly loads an implementation-address ABI,
// while retaining the proxy as transaction target. All chain evidence is read at
// one requested block. API ABI attribution is recorded, not bytecode verification.
// Use LoadBeaconDeploymentContract for standard beacons; minimal and
// non-EIP1967 proxies require separate explicit handling.
func (c *Client) LoadProxyDeploymentContract(ctx context.Context, api *API, address common.Address, block uint64) (*ProxyDeployment, error) {
	if api == nil || address == (common.Address{}) {
		return nil, errors.New("explicit API and proxy address required")
	}
	evidence, e := c.proxyEvidence(ctx, address, block)
	if e != nil {
		return nil, e
	}
	raw, e := api.DeploymentABI(ctx, evidence.Implementation)
	if e != nil {
		return nil, e
	}
	return c.proxyWithEvidence(address, raw, evidence)
}
func (c *Client) proxyWithEvidence(address common.Address, raw json.RawMessage, evidence DeploymentEvidence) (*ProxyDeployment, error) {
	contract, e := c.ContractWithABI(address, raw)
	if e != nil {
		return nil, e
	}
	parsed, _ := contract.contractABI()
	if len(parsed.Methods) == 0 {
		return nil, errors.New("implementation ABI has no callable methods")
	}
	evidence.ABIHash = crypto.Keccak256Hash(raw)
	internal := evidence
	contract.deployment = &internal
	return &ProxyDeployment{contract, evidence}, nil
}

// VerifyAt checks whether the observed proxy/implementation code and slot still
// match at another block. It never switches to another ABI automatically.
func (p *ProxyDeployment) VerifyAt(ctx context.Context, block uint64) error {
	if p == nil || p.Contract == nil || p.Contract.client == nil {
		return errors.New("proxy deployment client required")
	}
	if p.Contract.deployment == nil {
		return errors.New("proxy deployment evidence required")
	}
	return p.Contract.client.verifyDeployment(ctx, *p.Contract.deployment, block)
}
func (c *Client) verifyDeployment(ctx context.Context, expected DeploymentEvidence, block uint64) error {
	var actual DeploymentEvidence
	var e error
	if expected.Beacon != (common.Address{}) {
		actual, e = c.beaconEvidence(ctx, expected.Proxy, block)
	} else {
		actual, e = c.proxyEvidence(ctx, expected.Proxy, block)
	}
	if e != nil {
		return e
	}
	if actual.ChainID != expected.ChainID || actual.Implementation != expected.Implementation || actual.ProxyCodeHash != expected.ProxyCodeHash || actual.ImplementationCodeHash != expected.ImplementationCodeHash || actual.Beacon != expected.Beacon || actual.BeaconCodeHash != expected.BeaconCodeHash {
		return errors.New("proxy deployment changed; review ABI before signing")
	}
	return nil
}

func canonicalSlotAddress(slot []byte) (common.Address, error) {
	if len(slot) != 32 {
		return common.Address{}, errors.New("invalid address word")
	}
	for _, b := range slot[:12] {
		if b != 0 {
			return common.Address{}, errors.New("noncanonical address word")
		}
	}
	return common.BytesToAddress(slot[12:]), nil
}

// beaconEvidence follows https://eips.ethereum.org/EIPS/eip-1967. A beacon
// is considered only when the implementation slot is empty, never by scanning
// bytecode for address-like constants.
func (c *Client) beaconEvidence(ctx context.Context, address common.Address, block uint64) (DeploymentEvidence, error) {
	if e := c.checkChain(ctx); e != nil {
		return DeploymentEvidence{}, e
	}
	storage, ok := c.rpc.(storageReader)
	if !ok {
		return DeploymentEvidence{}, errors.New("backend does not support beacon storage reads")
	}
	height := new(big.Int).SetUint64(block)
	direct, e := storage.StorageAt(ctx, address, implementationSlot, height)
	if e != nil {
		return DeploymentEvidence{}, rpcFailure(ctx, "implementation slot unavailable")
	}
	impl, e := canonicalSlotAddress(direct)
	if e != nil || impl != (common.Address{}) {
		return DeploymentEvidence{}, errors.New("beacon requires empty implementation slot")
	}
	word, e := storage.StorageAt(ctx, address, beaconSlot, height)
	if e != nil {
		return DeploymentEvidence{}, rpcFailure(ctx, "beacon slot unavailable")
	}
	beacon, e := canonicalSlotAddress(word)
	if e != nil || beacon == (common.Address{}) || beacon == address {
		return DeploymentEvidence{}, errors.New("invalid beacon address")
	}
	proxy, e := c.rpc.CodeAt(ctx, address, height)
	if e != nil || len(proxy) == 0 {
		return DeploymentEvidence{}, rpcFailure(ctx, "proxy code unavailable")
	}
	beaconCode, e := c.rpc.CodeAt(ctx, beacon, height)
	if e != nil || len(beaconCode) == 0 {
		return DeploymentEvidence{}, rpcFailure(ctx, "beacon code unavailable")
	}
	selector := crypto.Keccak256([]byte("implementation()"))[:4]
	result, e := c.rpc.CallContract(ctx, ethereum.CallMsg{From: address, To: &beacon, Data: selector}, height)
	if e != nil {
		return DeploymentEvidence{}, rpcFailure(ctx, "beacon implementation unavailable")
	}
	implementation, e := canonicalSlotAddress(result)
	if e != nil || implementation == (common.Address{}) || implementation == address || implementation == beacon {
		return DeploymentEvidence{}, errors.New("invalid beacon implementation")
	}
	alternate, e := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &beacon, Data: selector}, height)
	if e != nil || string(alternate) != string(result) {
		return DeploymentEvidence{}, errors.New("beacon implementation depends on caller or is unavailable")
	}
	code, e := c.rpc.CodeAt(ctx, implementation, height)
	if e != nil || len(code) == 0 {
		return DeploymentEvidence{}, rpcFailure(ctx, "implementation code unavailable")
	}
	return DeploymentEvidence{ChainID: c.ChainID().String(), Block: block, Proxy: address, Implementation: implementation, ProxyCodeHash: crypto.Keccak256Hash(proxy), ImplementationCodeHash: crypto.Keccak256Hash(code), Beacon: beacon, BeaconCodeHash: crypto.Keccak256Hash(beaconCode)}, nil
}
func (c *Client) LoadBeaconDeploymentContract(ctx context.Context, api *API, address common.Address, block uint64) (*ProxyDeployment, error) {
	if api == nil || address == (common.Address{}) {
		return nil, errors.New("explicit API and proxy address required")
	}
	evidence, e := c.beaconEvidence(ctx, address, block)
	if e != nil {
		return nil, e
	}
	raw, e := api.DeploymentABI(ctx, evidence.Implementation)
	if e != nil {
		return nil, e
	}
	return c.proxyWithEvidence(address, raw, evidence)
}

// BindProxyABI and BindBeaconABI attach caller-reviewed artifacts when an API
// is unavailable or misannotated. Code/slot evidence is still checked on-chain.
func (c *Client) BindProxyABI(ctx context.Context, address common.Address, block uint64, raw json.RawMessage) (*ProxyDeployment, error) {
	evidence, e := c.proxyEvidence(ctx, address, block)
	if e != nil {
		return nil, e
	}
	return c.proxyWithEvidence(address, raw, evidence)
}
func (c *Client) BindBeaconABI(ctx context.Context, address common.Address, block uint64, raw json.RawMessage) (*ProxyDeployment, error) {
	evidence, e := c.beaconEvidence(ctx, address, block)
	if e != nil {
		return nil, e
	}
	return c.proxyWithEvidence(address, raw, evidence)
}
