// abi-role-inspect performs fixed read-only role probes at the evidence block.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"os"
	"strings"
	"time"
)

func main() {
	tokens := flag.Bool("tokens", false, "probe candidate token views instead of role addresses")
	output := flag.String("out", "", "new output file path; existing files are never overwritten")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	c, e := ethclient.DialContext(ctx, "https://testnet-val.decimalchain.com/web3/")
	if e != nil {
		panic("RPC unavailable")
	}
	defer c.Close()
	id, e := c.ChainID(ctx)
	if e != nil || id.Cmp(big.NewInt(202020)) != 0 {
		panic("wrong chain")
	}
	block := big.NewInt(22474733)
	type observation struct {
		Role, Address, Signature string
		Result                   any    `json:"result,omitempty"`
		Error                    string `json:"error,omitempty"`
	}
	var rows []observation
	addresses := map[string]string{"checks": "0xb6161CAA8Bd7167C2fa67b93b041FF9a94d6a735", "gas-center": "0xB32439bF0C3742D0a227BFEc78276F7bA15F8Aa1"}
	if *tokens {
		addresses = map[string]string{"reserveless-token": "0xc310f25e3429efc5ede11a908682f0a31cfc92ce", "reserve-token": "0x37748f3e857a71e4334125a935052ce1e603b7b5", "token-center": "0xb8F9c4A26b91a65Ac0D3F34e230a4C9ad09aEF4a"}
		block = big.NewInt(22474794)
	}
	for role, address := range addresses {
		if *tokens {
			to := common.HexToAddress(address)
			code, err := c.CodeAt(ctx, to, block)
			row := observation{Role: role, Address: address, Signature: "eth_getCode"}
			if err != nil {
				row.Error = "code read failed"
			} else {
				row.Result = "0x" + common.Bytes2Hex(code)
			}
			rows = append(rows, row)
		}
		specs := []struct {
			Name, Output string
			WithAddress  bool
		}{{"nonces", "uint256", false}, {"name", "string", false}, {"symbol", "string", false}, {"nonces", "uint256", true}}
		if *tokens {
			specs = append(specs, struct {
				Name, Output string
				WithAddress  bool
			}{"cap", "uint256", false}, struct {
				Name, Output string
				WithAddress  bool
			}{"getMintable", "bool", false}, struct {
				Name, Output string
				WithAddress  bool
			}{"getBurnable", "bool", false}, struct {
				Name, Output string
				WithAddress  bool
			}{"implementationReserveless", "address", false}, struct {
				Name, Output string
				WithAddress  bool
			}{"implementation", "address", false})
		}
		for _, spec := range specs {
			inputs := "[]"
			sig := spec.Name + "()"
			args := []any{}
			if spec.WithAddress {
				inputs = `[{"type":"address","name":"owner"}]`
				sig = spec.Name + "(address)"
				args = append(args, common.Address{})
			}
			raw := `[{"type":"function","name":"` + spec.Name + `","stateMutability":"view","inputs":` + inputs + `,"outputs":[{"type":"` + spec.Output + `"}]}]`
			a, _ := abi.JSON(strings.NewReader(raw))
			input, _ := a.Pack(spec.Name, args...)
			to := common.HexToAddress(address)
			out, err := c.CallContract(ctx, ethereum.CallMsg{To: &to, Data: input}, block)
			row := observation{Role: role, Address: address, Signature: sig}
			if err != nil {
				row.Error = "eth_call failed"
			} else {
				decoded, err := a.Unpack(spec.Name, out)
				if err != nil {
					row.Error = "ABI decode failed"
				} else {
					row.Result = decoded
				}
			}
			rows = append(rows, row)
		}
	}
	evidence := map[string]any{"chain_id": "202020", "block": block.String(), "scope": "fixed read-only eth_call probes; not a proof of absent methods", "observations": rows}
	data, _ := json.MarshalIndent(evidence, "", "  ")
	path := "docs/deployments/testnet-2026-10-29-additional/role-probes.json"
	if *tokens {
		path = "docs/deployments/testnet-2026-10-29-tokens/role-probes.json"
	}
	if *output != "" {
		path = *output
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		panic("snapshot exists or output unavailable")
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
	fmt.Println(string(data))
}
