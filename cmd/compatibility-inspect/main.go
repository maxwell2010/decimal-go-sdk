// compatibility-inspect records bounded, block-pinned public RPC reads only.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const implSlot = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"
const beaconSlot = "0xa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d50"

type probe struct {
	Signature string `json:"signature"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}
type reference struct {
	Expected []string `json:"historical_expected_hashes,omitempty"`
	Matches  *bool    `json:"matches_historical_hash,omitempty"`
	Kind     string   `json:"kind"`
	Address  string   `json:"address"`
	CodeHash string   `json:"code_hash,omitempty"`
	Error    string   `json:"error,omitempty"`
}
type entry struct {
	Role               string      `json:"role"`
	Address            string      `json:"address,omitempty"`
	Source             string      `json:"source"`
	CodeHash           string      `json:"code_hash,omitempty"`
	CodeBytes          int         `json:"code_bytes"`
	Expected           []string    `json:"historical_expected_hashes,omitempty"`
	Matches            *bool       `json:"matches_historical_hash,omitempty"`
	ImplementationSlot string      `json:"implementation_slot,omitempty"`
	BeaconSlot         string      `json:"beacon_slot,omitempty"`
	References         []reference `json:"references,omitempty"`
	Probes             []probe     `json:"probes,omitempty"`
	Error              string      `json:"error,omitempty"`
}
type target struct{ role, address, source string }

func selector(signature string) string {
	return "0x" + hex.EncodeToString(crypto.Keccak256([]byte(signature))[:4])
}
func addressResult(raw string) string {
	b, e := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
	if e != nil || len(b) != 32 {
		return ""
	}
	for _, v := range b[:12] {
		if v != 0 {
			return ""
		}
	}
	a := common.BytesToAddress(b[12:])
	if a == (common.Address{}) {
		return ""
	}
	return a.Hex()
}
func codeHash(raw string) (string, int, error) {
	b, e := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
	if e != nil {
		return "", 0, e
	}
	if len(b) == 0 {
		return "", 0, fmt.Errorf("empty runtime")
	}
	return crypto.Keccak256Hash(b).Hex(), len(b), nil
}
func getters(role string) []string {
	switch role {
	case "checks":
		return []string{"nonces()", "getImpl()", "owner()"}
	case "token-center":
		return []string{"owner()", "getImpl()", "implementation()", "implementationReserveless()"}
	case "reserve-token", "reserveless-token":
		return []string{"name()", "symbol()", "decimals()", "totalSupply()", "owner()"}
	case "safe":
		return []string{"nonce()", "getWeightThreshold()", "getOwnersWithWeights()"}
	case "safe-factory", "multisend", "multicall":
		return nil
	default:
		return []string{"getImpl()", "owner()"}
	}
}
func readTarget(ctx context.Context, c *rpc.Client, t target, block string, expected map[string][]string) entry {
	r := entry{Role: t.role, Address: common.HexToAddress(t.address).Hex(), Source: t.source}
	var code, impl, beacon string
	vals := []*string{&code, &impl, &beacon}
	calls := []rpc.BatchElem{{Method: "eth_getCode", Args: []any{r.Address, block}, Result: vals[0]}, {Method: "eth_getStorageAt", Args: []any{r.Address, implSlot, block}, Result: vals[1]}, {Method: "eth_getStorageAt", Args: []any{r.Address, beaconSlot, block}, Result: vals[2]}}
	sigs := getters(t.role)
	answers := make([]string, len(sigs))
	for i, s := range sigs {
		calls = append(calls, rpc.BatchElem{Method: "eth_call", Args: []any{map[string]string{"to": r.Address, "data": selector(s)}, block}, Result: &answers[i]})
	}
	if e := c.BatchCallContext(ctx, calls); e != nil {
		r.Error = "read batch unavailable"
		return r
	}
	if calls[0].Error != nil {
		r.Error = "runtime unavailable"
	} else {
		h, n, e := codeHash(code)
		r.CodeHash = h
		r.CodeBytes = n
		if e != nil {
			r.Error = e.Error()
		}
	}
	r.Expected = expected[strings.ToLower(r.Address)]
	if len(r.Expected) > 0 && r.CodeHash != "" {
		v := false
		for _, h := range r.Expected {
			if strings.EqualFold(h, r.CodeHash) {
				v = true
			}
		}
		r.Matches = &v
	}
	if calls[1].Error == nil {
		r.ImplementationSlot = impl
	}
	if calls[2].Error == nil {
		r.BeaconSlot = beacon
	}
	for i, s := range sigs {
		p := probe{Signature: s, Result: answers[i]}
		if calls[3+i].Error != nil {
			p.Result = ""
			p.Error = "eth_call failed (revert or RPC error)"
		}
		r.Probes = append(r.Probes, p)
	}
	refs := map[string]string{"eip1967-implementation": addressResult(impl), "eip1967-beacon": addressResult(beacon)}
	for i, s := range sigs {
		if s == "getImpl()" || s == "implementation()" || s == "implementationReserveless()" {
			if calls[3+i].Error == nil {
				refs[s] = addressResult(answers[i])
			}
		}
	}
	keys := make([]string, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		a := refs[k]
		if a == "" {
			continue
		}
		ref := reference{Kind: k, Address: a}
		var raw string
		if e := c.CallContext(ctx, &raw, "eth_getCode", a, block); e != nil {
			ref.Error = "reference code unavailable"
		} else {
			h, _, e := codeHash(raw)
			ref.CodeHash = h
			if e != nil {
				ref.Error = e.Error()
			}
		}
		compareReference(&ref, expected)
		r.References = append(r.References, ref)
		if k == "eip1967-beacon" {
			var out string
			e := c.CallContext(ctx, &out, "eth_call", map[string]string{"to": a, "data": selector("implementation()")}, block)
			if e == nil {
				v := addressResult(out)
				if v != "" {
					bref := reference{Kind: "beacon-implementation-getter", Address: v}
					var bcode string
					if e = c.CallContext(ctx, &bcode, "eth_getCode", v, block); e == nil {
						bref.CodeHash, _, _ = codeHash(bcode)
					} else {
						bref.Error = "beacon implementation runtime unavailable"
					}
					compareReference(&bref, expected)
					r.References = append(r.References, bref)
				}
			}
		}
	}
	return r
}
func manifests(dir string) ([]target, map[string][]string, error) {
	var targets []target
	expected := map[string][]string{}
	files, e := filepath.Glob(filepath.Join(dir, "*deployment-sources.json"))
	if e != nil {
		return nil, nil, e
	}
	for _, p := range files {
		raw, e := os.ReadFile(p)
		if e != nil {
			return nil, nil, e
		}
		var doc struct {
			Profiles []map[string]any `json:"profiles"`
		}
		if e = json.Unmarshal(raw, &doc); e != nil {
			return nil, nil, e
		}
		for _, m := range doc.Profiles {
			add := func(a, h string) {
				addr, _ := m[a].(string)
				hash, _ := m[h].(string)
				if common.IsHexAddress(addr) && hash != "" {
					key := strings.ToLower(common.HexToAddress(addr).Hex())
					if !contains(expected[key], hash) {
						expected[key] = append(expected[key], hash)
					}
				}
			}
			add("target_proxy", "proxy_code_hash")
			if _, tokenInterface := m["abi_address"]; !tokenInterface {
				add("target_proxy", "runtime_code_hash")
			}
			add("implementation", "implementation_code_hash")
			add("abi_address", "runtime_code_hash")
			add("beacon", "beacon_code_hash")
			role := ""
			profile, _ := m["profile"].(string)
			switch profile {
			case "TestnetReserveToken20261029":
				role = "reserve-token"
			case "TestnetReservelessToken20261029":
				role = "reserveless-token"
			}
			if role != "" {
				a, _ := m["target_proxy"].(string)
				targets = append(targets, target{role, a, filepath.Base(p) + " explicit testnet token binding"})
			}
		}
	}
	return targets, expected, nil
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func compareReference(r *reference, expected map[string][]string) {
	r.Expected = expected[strings.ToLower(r.Address)]
	if len(r.Expected) > 0 && r.CodeHash != "" {
		v := contains(r.Expected, r.CodeHash)
		r.Matches = &v
	}
}
func main() {
	blockFlag := flag.Uint64("block", 0, "explicit snapshot block; zero selects current block")
	network := flag.String("network", "testnet", "testnet or mainnet")
	out := flag.String("out", "", "new output JSON path (required)")
	docs := flag.String("docs", "docs", "historical manifest directory")
	flag.Parse()
	if *out == "" {
		panic("-out required")
	}
	if _, e := os.Stat(*out); !os.IsNotExist(e) {
		panic("output already exists or cannot be inspected")
	}
	cfg := decimal.Testnet()
	if *network == "mainnet" {
		cfg = decimal.Mainnet()
	} else if *network != "testnet" {
		panic("invalid network")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	transport, e := rpc.DialOptions(ctx, cfg.RPCURL, rpc.WithHTTPClient(&http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
	must(e)
	defer transport.Close()
	var chain, block string
	must(transport.CallContext(ctx, &chain, "eth_chainId"))
	n, ok := new(big.Int).SetString(strings.TrimPrefix(chain, "0x"), 16)
	if !ok || n.Cmp(cfg.ChainID) != 0 {
		panic("chain mismatch")
	}
	if *blockFlag == 0 {
		must(transport.CallContext(ctx, &block, "eth_blockNumber"))
	} else {
		block = fmt.Sprintf("0x%x", *blockFlag)
	}
	sdk, e := decimal.NewClient(cfg, ethclient.NewClient(transport))
	must(e)
	header, e := snapshotIdentity(ctx, sdk, block)
	must(e)
	historical, expected, e := manifests(*docs)
	must(e)
	targets := []target{{"contract-center", cfg.ContractCenter.Hex(), "official SDK Config"}, {"checks", cfg.Checks.Hex(), "official SDK Config"}, {"safe", cfg.Safe.Hex(), "official SDK Config"}, {"safe-factory", cfg.SafeFactory.Hex(), "official SDK Config"}, {"multisend", cfg.MultiSend.Hex(), "official SDK Config"}, {"multicall", cfg.Multicall.Hex(), "official SDK Config"}}
	if *network == "testnet" {
		targets = append(targets, historical...)
		targets = append(targets, target{"bridge", "0x827ea5381adb912ed0befda26afe8cdc5a5e050a", "prior official TheGraph discovery docs/deployments/testnet-2026-10-29-bridge"})
	} else {
		expected = map[string][]string{}
	}
	raw, e := decimal.ABIJSON("ContractCenter")
	must(e)
	registry, e := abi.JSON(strings.NewReader(string(raw)))
	must(e)
	var rows []entry
	for _, slug := range []string{"token-center", "nft-center", "master-validator", "delegation", "delegation-nft", "gas-center", "bridge", "bridge-v2"} {
		call, e := registry.Pack("getContractAddress", slug)
		must(e)
		var response string
		e = transport.CallContext(ctx, &response, "eth_call", map[string]string{"to": cfg.ContractCenter.Hex(), "data": "0x" + hex.EncodeToString(call)}, block)
		if e != nil {
			rows = append(rows, entry{Role: slug, Source: "ContractCenter getContractAddress(string)", Error: "registry call failed"})
			continue
		}
		a := addressResult(response)
		if a == "" {
			rows = append(rows, entry{Role: slug, Source: "ContractCenter getContractAddress(string)", Error: "registry returned zero or invalid address"})
			continue
		}
		targets = append(targets, target{slug, a, "ContractCenter getContractAddress(string)"})
	}
	for _, t := range targets {
		rows = append(rows, readTarget(ctx, transport, t, block, expected))
	}
	final, e := snapshotIdentity(ctx, sdk, block)
	must(e)
	if final.Hash != header.Hash {
		panic("snapshot block changed")
	}
	result := map[string]any{"network": *network, "rpc": cfg.RPCURL, "chain_id": cfg.ChainID.String(), "block": block, "block_hash": header.Hash, "checked_utc": time.Now().UTC().Format(time.RFC3339), "entries": rows, "boundary": "read-only code/storage/calls; no ABI inference, stateful authorization, simulation or write acceptance; failed getters do not prove absence"}
	data, e := json.MarshalIndent(result, "", "  ")
	must(e)
	must(os.MkdirAll(filepath.Dir(*out), 0755))
	must(os.WriteFile(*out, append(data, '\n'), 0644))
	fmt.Printf("%s block=%s entries=%d saved=%s\n", *network, block, len(rows), *out)
}
func must(e error) {
	if e != nil {
		panic(e)
	}
}

// snapshotIdentity uses the SDK's strict native identity reader, never Header.Hash.
func snapshotIdentity(ctx context.Context, c *decimal.Client, tag string) (decimal.BlockIdentity, error) {
	height, e := hexutil.DecodeUint64(tag)
	if e != nil {
		return decimal.BlockIdentity{}, e
	}
	return c.BlockIdentity(ctx, new(big.Int).SetUint64(height))
}
