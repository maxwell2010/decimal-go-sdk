package decimal

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:embed abi/*.json
var abiFiles embed.FS

//go:embed docs/*deployment-sources.json
var deploymentProfileSources embed.FS

func (c *Client) checkProfileNetwork(profile string) error {
	if c == nil || c.cfg.ChainID == nil {
		return nil
	} // Offline construction has no network.
	paths, e := fs.Glob(deploymentProfileSources, "docs/*deployment-sources.json")
	if e != nil {
		return e
	}
	for _, path := range paths {
		var source struct {
			Profiles []struct {
				Profile string
				ChainID string `json:"chain_id"`
			}
		}
		raw, e := deploymentProfileSources.ReadFile(path)
		if e != nil {
			return e
		}
		if e := json.Unmarshal(raw, &source); e != nil {
			return errors.New("invalid deployment profile manifest")
		}
		for _, p := range source.Profiles {
			if p.Profile == profile && p.ChainID != c.cfg.ChainID.String() {
				return errors.New("versioned ABI profile belongs to another network")
			}
		}
	}
	return nil
}

var abiCache sync.Map

func loadABI(name string) (abi.ABI, error) {
	if a, ok := abiCache.Load(name); ok {
		return a.(abi.ABI), nil
	}
	b, e := abiFiles.ReadFile("abi/" + name + ".json")
	if e != nil {
		return abi.ABI{}, errors.New("unknown ABI profile")
	}
	a, e := abi.JSON(strings.NewReader(string(b)))
	if e != nil {
		return a, e
	}
	abiCache.Store(name, a)
	return a, nil
}

// Call is an offline transaction intent in base units. It contains no private key.
type Call struct {
	To                 common.Address
	Value              *big.Int
	Data               []byte
	Contract           bool
	Profile, Signature string
	deployment         *DeploymentEvidence
}

func NativeTransfer(to common.Address, value *big.Int, memo string) (Call, error) {
	if !uint256(value) {
		return Call{}, errors.New("invalid uint256 value")
	}
	return Call{To: to, Value: copyInt(value), Data: []byte(memo)}, nil
}

type Contract struct {
	client     *Client
	Address    common.Address
	Profile    string
	override   *abi.ABI
	deployment *DeploymentEvidence
}

func (c *Client) Contract(profile string, address common.Address) (*Contract, error) {
	if e := c.checkProfileNetwork(profile); e != nil {
		return nil, e
	}
	if address == (common.Address{}) {
		return nil, errors.New("contract address is zero")
	}
	if _, e := loadABI(profile); e != nil {
		return nil, e
	}
	return &Contract{client: c, Address: address, Profile: profile}, nil
}

// ContractWithABI opts into a caller-supplied deployment ABI. It does not fetch
// or trust a remote ABI implicitly. Keep the deployment address/network/block
// and artifact provenance alongside this ABI in the application.
func (c *Client) ContractWithABI(address common.Address, raw json.RawMessage) (*Contract, error) {
	if address == (common.Address{}) {
		return nil, errors.New("contract address is zero")
	}
	a, e := abi.JSON(strings.NewReader(string(raw)))
	if e != nil {
		return nil, errors.New("invalid deployment ABI")
	}
	return &Contract{client: c, Address: address, Profile: "deployment", override: &a}, nil
}
func (c *Contract) contractABI() (abi.ABI, error) {
	if c.override != nil {
		return *c.override, nil
	}
	return loadABI(c.Profile)
}
func method(a abi.ABI, sig string) (string, abi.Method, error) {
	for k, m := range a.Methods {
		if m.Sig == sig {
			return k, m, nil
		}
	}
	return "", abi.Method{}, fmt.Errorf("unknown contract signature %q", sig)
}

func (c *Contract) Build(signature string, value *big.Int, args ...any) (Call, error) {
	a, e := c.contractABI()
	if e != nil {
		return Call{}, e
	}
	k, m, e := method(a, signature)
	if e != nil {
		return Call{}, e
	}
	if c.override == nil && c.Profile == "DelegationNFT" && c.client != nil &&
		c.client.cfg.ChainID != nil && c.client.cfg.ChainID.Cmp(big.NewInt(75)) == 0 &&
		c.Address == common.HexToAddress("0xe45adfcc739a0d10ce9462b58866c9a1a06035e2") &&
		(strings.HasPrefix(m.RawName, "delegate") || m.RawName == "hold") {
		return Call{}, errors.New("retired mainnet NFT delegation accepts historical reads and exits only; use UnifiedNFTDelegation for new deposits")
	}
	if m.IsConstant() {
		return Call{}, errors.New("read method cannot build a transaction")
	}
	v := copyInt(value)
	if !uint256(v) {
		return Call{}, errors.New("invalid call value")
	}
	if m.StateMutability != "payable" && v.Sign() != 0 {
		return Call{}, errors.New("nonpayable method has value")
	}
	args, e = normalizeArguments(m.Inputs, args)
	if e != nil {
		return Call{}, e
	}
	data, e := a.Pack(k, args...)
	if e != nil {
		return Call{}, e
	}
	call := Call{To: c.Address, Value: v, Data: data, Contract: true, Profile: c.Profile, Signature: signature}
	if c.deployment != nil {
		snapshot := *c.deployment
		call.deployment = &snapshot
	}
	return call, nil
}
func (c *Contract) Read(ctx context.Context, signature string, args ...any) ([]any, error) {
	return c.ReadAt(ctx, nil, signature, args...)
}

// ReadAt pins a read to a block number; nil requests latest.
func (c *Contract) ReadAt(ctx context.Context, block *big.Int, signature string, args ...any) ([]any, error) {
	return c.ReadFromAt(ctx, common.Address{}, block, signature, args...)
}

// ReadFromAt supplies an explicit eth_call sender for caller-dependent views.
// It does not sign, transfer value or prove nested STATICCALL behavior.
func (c *Contract) ReadFromAt(ctx context.Context, from common.Address, block *big.Int, signature string, args ...any) ([]any, error) {
	if ctx == nil {
		return nil, errors.New("context required")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if c == nil || c.client == nil {
		return nil, errors.New("client required for contract reads")
	}
	// Preserve caller-owned identity even when a custom backend borrows and
	// mutates its request parameters during chain checking or eth_call.
	target := c.Address
	if block != nil {
		block = new(big.Int).Set(block)
	}
	if c.deployment != nil {
		if target != c.deployment.Proxy {
			return nil, errors.New("proxy read target changed")
		}
		var height uint64
		if block == nil {
			current, e := c.client.BlockNumber(ctx)
			if e != nil {
				return nil, e
			}
			height = current
		} else {
			if block.Sign() < 0 || !block.IsUint64() {
				return nil, errors.New("explicit nonnegative proxy snapshot required")
			}
			height = block.Uint64()
		}
		if e := c.client.verifyDeployment(ctx, *c.deployment, height); e != nil {
			return nil, e
		}
		block = new(big.Int).SetUint64(height)
	}
	a, e := c.contractABI()
	if e != nil {
		return nil, e
	}
	k, m, e := method(a, signature)
	if e != nil {
		return nil, e
	}
	if !m.IsConstant() {
		return nil, errors.New("write method requires simulation or a transaction")
	}
	args, e = normalizeArguments(m.Inputs, args)
	if e != nil {
		return nil, e
	}
	data, e := a.Pack(k, args...)
	if e != nil {
		return nil, e
	}
	if e = c.client.checkChain(ctx); e != nil {
		return nil, e
	}
	b, e := c.client.rpc.CallContract(ctx, ethereum.CallMsg{From: from, To: &target, Data: data}, block)
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	if e != nil {
		return nil, decodeRPCError(e, a)
	}
	// ABI dynamic bytes may refer into the encoded response. Own that response
	// so a backend cannot change an already returned result by reusing buffers.
	owned := append([]byte(nil), b...)
	var values []any
	if c.Profile == "UnifiedNFTDelegation20261030" {
		values, e = canonicalDelegationUnpack(m.Outputs, owned)
	} else if c.override == nil && c.Profile == "Delegation" {
		values, e = unpackDelegationRead(c.Profile, m, owned)
	} else {
		values, e = m.Outputs.Unpack(owned)
	}
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	return values, e
}

type Event struct {
	Name      string
	Signature string
	Values    map[string]any
	Log       types.Log
}

func (c *Contract) DecodeLog(log types.Log) (Event, error) {
	if log.Address != c.Address || len(log.Topics) == 0 {
		return Event{}, errors.New("log is not from this contract")
	}
	a, e := c.contractABI()
	if e != nil {
		return Event{}, e
	}
	if (c.override == nil && (c.Profile == "Delegation" || c.Profile == "DelegationNFT")) || c.Profile == "UnifiedNFTDelegation20261030" {
		return decodeDelegationUpgradeLog(a, log, c.Profile == "Delegation")
	}
	ev, e := a.EventByID(log.Topics[0])
	if e != nil {
		return Event{}, e
	}
	values := map[string]any{}
	if e = ev.Inputs.UnpackIntoMap(values, log.Data); e != nil {
		return Event{}, e
	}
	var indexed abi.Arguments
	for _, arg := range ev.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if e = abi.ParseTopicsIntoMap(values, indexed, log.Topics[1:]); e != nil {
		return Event{}, e
	}
	return Event{ev.Name, ev.Sig, values, log}, nil
}

// ABIJSON returns a copy of the pinned ABI, for interoperability and verification.
func ABIJSON(profile string) (json.RawMessage, error) {
	b, e := abiFiles.ReadFile("abi/" + profile + ".json")
	return b, e
}
