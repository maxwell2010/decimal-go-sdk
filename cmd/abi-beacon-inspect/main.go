// abi-beacon-inspect records standard EIP1967 beacon storage and view-call evidence.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, e := ethclient.DialContext(ctx, "https://testnet-val.decimalchain.com/web3/")
	if e != nil {
		panic("RPC unavailable")
	}
	defer c.Close()
	chain, e := c.ChainID(ctx)
	if e != nil || chain.Cmp(big.NewInt(202020)) != 0 {
		panic("wrong chain")
	}
	block := big.NewInt(22474794)
	rows := []map[string]any{}
	slot := common.HexToHash("0xa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d50")
	for role, address := range map[string]string{"reserveless": "0xc310f25e3429efc5ede11a908682f0a31cfc92ce", "reserve": "0x37748f3e857a71e4334125a935052ce1e603b7b5"} {
		target := common.HexToAddress(address)
		r := map[string]any{"role": role, "target": target.Hex(), "slot": slot.Hex()}
		stored, e := c.StorageAt(ctx, target, slot, block)
		if e != nil || len(stored) != 32 {
			r["error"] = "beacon storage unavailable"
			rows = append(rows, r)
			continue
		}
		r["storage_value"] = "0x" + common.Bytes2Hex(stored)
		beacon := common.BytesToAddress(stored)
		r["beacon"] = beacon.Hex()
		if beacon == (common.Address{}) {
			r["error"] = "standard beacon slot is zero"
			rows = append(rows, r)
			continue
		}
		code, e := c.CodeAt(ctx, beacon, block)
		if e != nil || len(code) == 0 {
			r["error"] = "beacon code unavailable"
			rows = append(rows, r)
			continue
		}
		r["beacon_code_hash"] = crypto.Keccak256Hash(code).Hex()
		output, e := c.CallContract(ctx, ethereum.CallMsg{To: &beacon, Data: common.FromHex("0x5c60da1b")}, block)
		if e != nil || len(output) != 32 {
			r["error"] = "implementation getter unavailable"
			rows = append(rows, r)
			continue
		}
		impl := common.BytesToAddress(output)
		r["implementation"] = impl.Hex()
		code, e = c.CodeAt(ctx, impl, block)
		if e != nil || len(code) == 0 {
			r["error"] = "implementation code unavailable"
		} else {
			r["implementation_code_hash"] = crypto.Keccak256Hash(code).Hex()
		}
		rows = append(rows, r)
	}
	data, _ := json.MarshalIndent(map[string]any{"chain_id": "202020", "block": block.String(), "scope": "read-only standard beacon storage/getter evidence", "entries": rows}, "", "  ")
	file, e := os.OpenFile("docs/deployments/testnet-2026-10-29-tokens/beacon-probes.json", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		panic("evidence exists or unavailable")
	}
	defer file.Close()
	_, _ = file.Write(append(data, '\n'))
	fmt.Println(string(data))
}
