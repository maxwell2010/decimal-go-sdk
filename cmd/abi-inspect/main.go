// abi-inspect records public testnet registry/ABI evidence. No wallet/signing API is used.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	decimal "github.com/maxwell2010/decimal-go-sdk"
)

type entry struct {
	AddressSource string   `json:"address_source,omitempty"`
	Proxy         string   `json:"proxy,omitempty"`
	Slug          string   `json:"slug"`
	Address       string   `json:"address,omitempty"`
	CodeHash      string   `json:"code_hash,omitempty"`
	APIURL        string   `json:"api_url,omitempty"`
	ABIHash       string   `json:"abi_sha256,omitempty"`
	Functions     []string `json:"functions,omitempty"`
	Error         string   `json:"error,omitempty"`
}

func main() {
	out := flag.String("out", "docs/deployments/testnet-2026-10-29", "evidence output directory")
	only := flag.String("only", "", "comma-separated slugs; empty scans all")
	flag.Parse()
	if _, e := os.Stat(filepath.Join(*out, "metadata.json")); e == nil {
		fail("evidence snapshot already exists; choose a new -out directory")
	}
	cfg := decimal.Testnet()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	httpClient := &http.Client{Timeout: 6 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	transport, e := rpc.DialOptions(ctx, cfg.RPCURL, rpc.WithHTTPClient(httpClient))
	if e != nil {
		fail("RPC unavailable")
	}
	defer transport.Close()
	client := ethclient.NewClient(transport)
	chain, e := client.ChainID(ctx)
	if e != nil || chain.Cmp(big.NewInt(202020)) != 0 {
		fail("testnet chain ID not confirmed")
	}
	block, e := client.BlockNumber(ctx)
	if e != nil {
		fail("block unavailable")
	}
	at := new(big.Int).SetUint64(block)
	raw, e := decimal.ABIJSON("ContractCenter")
	if e != nil {
		fail("registry ABI unavailable")
	}
	registry, e := abi.JSON(strings.NewReader(string(raw)))
	if e != nil {
		fail("registry ABI invalid")
	}
	if e = os.MkdirAll(*out, 0755); e != nil {
		fail("output directory unavailable")
	}
	var entries []entry
	selected := map[string]bool{}
	for _, s := range strings.Split(*only, ",") {
		selected[s] = true
	}
	static := map[string]common.Address{"checks-static": cfg.Checks, "safe-static": cfg.Safe, "safe-factory-static": cfg.SafeFactory, "multisend-static": cfg.MultiSend, "gas-center-static": common.HexToAddress("0xB32439bF0C3742D0a227BFEc78276F7bA15F8Aa1"), "reserveless-token-candidate": common.HexToAddress("0xc310f25e3429efc5ede11a908682f0a31cfc92ce"), "reserve-token-candidate": common.HexToAddress("0x37748f3e857a71e4334125a935052ce1e603b7b5")}
	static["reserveless-implementation-candidate"] = common.HexToAddress("0x9d3e4e9953b031f2c89da77c0f2abad169edd43f")
	static["reserve-implementation-candidate"] = common.HexToAddress("0x8e215123f9cf7d2c7d99457c5096bea82a868cbe")
	static["bridge-subgraph-candidate"] = common.HexToAddress("0x827ea5381adb912ed0befda26afe8cdc5a5e050a")
	slugs := []string{"contract-center", "delegation", "delegation-nft", "token-center", "nft-center", "master-validator", "gas-center", "bridge", "bridge-v2", "checks-static", "safe-static", "safe-factory-static", "multisend-static", "gas-center-static", "reserveless-token-candidate", "reserve-token-candidate"}
	if *only != "" {
		slugs = strings.Split(*only, ",")
	}
	for _, slug := range slugs {
		r := entry{Slug: slug}
		if *only != "" && !selected[slug] {
			continue
		}
		address := cfg.ContractCenter
		if fixed, ok := static[slug]; ok {
			address = fixed
			r.AddressSource = "https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/src/endpoints.ts"
			if strings.HasSuffix(slug, "token-candidate") {
				r.AddressSource = "candidate from local C# docs/VALIDATION.md; runtime and compact ABI checked independently via official public endpoints"
			}
			if strings.HasSuffix(slug, "implementation-candidate") {
				r.AddressSource = "official TokenCenter implementation()/implementationReserveless() read at block22474794; docs/deployments/testnet-2026-10-29-tokens/role-probes.json"
			}
			if slug == "bridge-subgraph-candidate" {
				r.AddressSource = "official JS getBridgeContracts via testnet-thegraph bridges query; docs/deployments/testnet-2026-10-29-additional/bridge-discovery.json"
			}
		} else if slug != "contract-center" {
			input, _ := registry.Pack("getContractAddress", slug)
			response, err := client.CallContract(ctx, ethereum.CallMsg{To: &cfg.ContractCenter, Data: input}, at)
			if err != nil {
				r.Error = "registry call failed"
				entries = append(entries, r)
				continue
			}
			values, err := registry.Unpack("getContractAddress", response)
			if err != nil || len(values) != 1 {
				r.Error = "registry decode failed"
				entries = append(entries, r)
				continue
			}
			address = values[0].(common.Address)
		}
		r.Address = address.Hex()
		if address == (common.Address{}) {
			r.Error = "unregistered slug"
			entries = append(entries, r)
			continue
		}
		code, err := client.CodeAt(ctx, address, at)
		if err != nil || len(code) == 0 {
			r.Error = "runtime unavailable"
			entries = append(entries, r)
			continue
		}
		r.CodeHash = crypto.Keccak256Hash(code).Hex()
		r.APIURL = strings.TrimRight(cfg.APIURL, "/") + "/contracts/compact/" + strings.ToLower(address.Hex())
		data, err := fetchABI(ctx, httpClient, r.APIURL)
		if err != nil {
			r.Error = err.Error()
			entries = append(entries, r)
			continue
		}
		parsed, err := abi.JSON(strings.NewReader(string(data)))
		if err != nil {
			r.Error = "API ABI is invalid"
			entries = append(entries, r)
			continue
		}
		for _, method := range parsed.Methods {
			r.Functions = append(r.Functions, method.Sig)
		}
		sort.Strings(r.Functions)
		hash := sha256.Sum256(data)
		r.ABIHash = hex.EncodeToString(hash[:])
		if err = os.WriteFile(filepath.Join(*out, slug+".abi.json"), data, 0644); err != nil {
			fail("cannot save ABI")
		}
		entries = append(entries, r)
	}
	// Compact endpoints return proxy interfaces. Resolve the standard ERC1967
	// implementation storage slot at the same snapshot block before requesting
	// the implementation's independently addressed ABI.
	for _, proxy := range entries {
		if proxy.CodeHash == "" {
			continue
		}
		address := common.HexToAddress(proxy.Address)
		slot := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
		stored, err := client.StorageAt(ctx, address, slot, at)
		r := entry{Slug: proxy.Slug + "-implementation", Proxy: proxy.Address}
		if err != nil || len(stored) != 32 {
			r.Error = "implementation storage unavailable"
			entries = append(entries, r)
			continue
		}
		implementation := common.BytesToAddress(stored)
		r.Address = implementation.Hex()
		if implementation == (common.Address{}) {
			r.Error = "ERC1967 implementation slot is zero"
			entries = append(entries, r)
			continue
		}
		code, err := client.CodeAt(ctx, implementation, at)
		if err != nil || len(code) == 0 {
			r.Error = "implementation runtime unavailable"
			entries = append(entries, r)
			continue
		}
		r.CodeHash = crypto.Keccak256Hash(code).Hex()
		r.APIURL = strings.TrimRight(cfg.APIURL, "/") + "/contracts/compact/" + strings.ToLower(implementation.Hex())
		raw, err := fetchABI(ctx, httpClient, r.APIURL)
		if err != nil {
			r.Error = err.Error()
			entries = append(entries, r)
			continue
		}
		parsed, err := abi.JSON(strings.NewReader(string(raw)))
		if err != nil {
			r.Error = "implementation ABI invalid"
			entries = append(entries, r)
			continue
		}
		for _, method := range parsed.Methods {
			r.Functions = append(r.Functions, method.Sig)
		}
		sort.Strings(r.Functions)
		if len(r.Functions) == 0 {
			r.Error = "implementation API ABI contains no methods"
		}
		hash := sha256.Sum256(raw)
		r.ABIHash = hex.EncodeToString(hash[:])
		if err = os.WriteFile(filepath.Join(*out, r.Slug+".abi.json"), raw, 0644); err != nil {
			fail("cannot save implementation ABI")
		}
		entries = append(entries, r)
	}
	evidence := map[string]any{"checked_utc": time.Now().UTC().Format(time.RFC3339), "network": "testnet", "chain_id": chain.String(), "block": block, "rpc_url": cfg.RPCURL, "registry": cfg.ContractCenter.Hex(), "entries": entries, "scope": "read-only registry and API evidence; no transaction execution; API ABI is not cryptographic proof of proxy implementation ABI"}
	data, _ := json.MarshalIndent(evidence, "", "  ")
	if e = os.WriteFile(filepath.Join(*out, "metadata.json"), append(data, '\n'), 0644); e != nil {
		fail("cannot save metadata")
	}
	fmt.Println(string(data))
}
func fetchABI(ctx context.Context, c *http.Client, url string) (json.RawMessage, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, e := c.Do(req)
	if e != nil {
		return nil, fmt.Errorf("compact API unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("compact API HTTP %d", res.StatusCode)
	}
	body, e := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if e != nil {
		return nil, fmt.Errorf("compact API body failed")
	}
	var obj struct {
		Result struct {
			ABI json.RawMessage `json:"abi"`
		} `json:"Result"`
	}
	if json.Unmarshal(body, &obj) != nil {
		return nil, fmt.Errorf("compact API invalid JSON")
	}
	data := obj.Result.ABI
	if len(data) == 0 {
		return nil, fmt.Errorf("compact API has no Result.abi")
	}
	if data[0] == '"' {
		var s string
		if json.Unmarshal(data, &s) != nil {
			return nil, fmt.Errorf("compact API ABI string invalid")
		}
		data = json.RawMessage(s)
	}
	return data, nil
}
func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
