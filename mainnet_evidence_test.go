package decimal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestMainnetEvidenceArtifactsAndRuntime(t *testing.T) {
	raw, err := os.ReadFile("docs/mainnet-deployment-sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Profiles []struct {
			Profile, Evidence string
			Chain             string            `json:"chain_id"`
			Proxy             string            `json:"target_proxy"`
			Implementation    string            `json:"abi_address"`
			Runtime           string            `json:"runtime_code_hash"`
			ProxyHash         string            `json:"proxy_code_hash"`
			Block             uint64            `json:"observed_block"`
			BlockHash         string            `json:"block_hash"`
			Beacon            string            `json:"beacon"`
			Files             map[string]string `json:"evidence_files_sha256"`
		}
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Profiles) != 2 {
		t.Fatal("expected two reviewed interfaces")
	}
	for _, p := range manifest.Profiles {
		t.Run(p.Profile, func(t *testing.T) {
			if p.Chain != "75" || p.Proxy == p.Implementation {
				t.Fatal("invalid mainnet execution identity")
			}
			dir := filepath.Dir(filepath.Join("docs", p.Evidence))
			codes := map[string]string{}
			chainSeen, blockSeen, slotSeen, implementationSeen := false, false, false, false
			for name, want := range p.Files {
				if filepath.Base(name) != name {
					t.Fatal("invalid evidence path")
				}
				b, e := os.ReadFile(filepath.Join(dir, name))
				if e != nil {
					t.Fatal(e)
				}
				sum := sha256.Sum256(b)
				if hex.EncodeToString(sum[:]) != want {
					t.Fatal("evidence drift", name)
				}
				if !strings.HasSuffix(name, "-request.json") {
					continue
				}
				var req struct {
					Method string
					Params []json.RawMessage
				}
				if json.Unmarshal(b, &req) != nil {
					continue
				}
				rb, e := os.ReadFile(filepath.Join(dir, strings.TrimSuffix(name, "-request.json")+"-response.json"))
				if e != nil {
					t.Fatal(e)
				}
				var envelope struct{ Result json.RawMessage }
				if json.Unmarshal(rb, &envelope) != nil {
					t.Fatal("bad evidence envelope")
				}
				if req.Method == "eth_chainId" {
					var v string
					_ = json.Unmarshal(envelope.Result, &v)
					chainSeen = chainSeen || v == "0x4b"
					continue
				}
				if req.Method == "eth_getBlockByNumber" {
					var v struct{ Number, Hash string }
					_ = json.Unmarshal(envelope.Result, &v)
					blockSeen = blockSeen || (v.Number == fmt.Sprintf("0x%x", p.Block) && v.Hash == p.BlockHash)
					continue
				}
				if req.Method == "eth_getStorageAt" && len(req.Params) >= 2 {
					var a, slot, v string
					_ = json.Unmarshal(req.Params[0], &a)
					_ = json.Unmarshal(req.Params[1], &slot)
					_ = json.Unmarshal(envelope.Result, &v)
					if strings.EqualFold(a, p.Proxy) {
						word := common.HexToAddress(v)
						if slot == implementationSlot.Hex() && word == common.HexToAddress(p.Implementation) {
							slotSeen = true
							implementationSeen = true
						}
						if slot == beaconSlot.Hex() && p.Beacon != "" && word == common.HexToAddress(p.Beacon) {
							slotSeen = true
						}
					}
					continue
				}
				if req.Method == "eth_call" && len(req.Params) > 0 && p.Beacon != "" {
					var call struct{ To, Data string }
					_ = json.Unmarshal(req.Params[0], &call)
					var v string
					_ = json.Unmarshal(envelope.Result, &v)
					if strings.EqualFold(call.To, p.Beacon) && call.Data == "0x5c60da1b" && common.HexToAddress(v) == common.HexToAddress(p.Implementation) {
						implementationSeen = true
					}
					continue
				}
				if req.Method != "eth_getCode" {
					continue
				}
				var res struct{ Result string }
				if json.Unmarshal(rb, &res) != nil {
					t.Fatal("bad RPC response")
				}
				var address string
				if len(req.Params) < 2 || json.Unmarshal(req.Params[0], &address) != nil {
					t.Fatal("bad code request")
				}
				code := common.FromHex(res.Result)
				if len(code) == 0 {
					t.Fatal("empty observed code")
				}
				codes[strings.ToLower(address)] = crypto.Keccak256Hash(code).Hex()
			}
			if codes[strings.ToLower(p.Proxy)] != p.ProxyHash || codes[strings.ToLower(p.Implementation)] != p.Runtime {
				t.Fatal("raw observed code does not match reviewed identities")
			}
			if !chainSeen || !blockSeen || !slotSeen || !implementationSeen {
				t.Fatal("missing chain/block/proxy binding proofs", chainSeen, blockSeen, slotSeen, implementationSeen)
			}
		})
	}
}
