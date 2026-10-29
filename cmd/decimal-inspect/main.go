// decimal-inspect performs read-only network checks. It never loads a wallet.
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"math/big"
	"os"
	"strings"
	"time"
)

func main() {
	network := flag.String("network", "testnet", "testnet or mainnet")
	codeAddress := flag.String("code", "", "optional public contract address for pinned-block code inspection")
	expected := flag.String("expected-runtime", "", "optional exact 32-byte runtime keccak256")
	flag.Parse()
	cfg := decimal.Testnet()
	if *network == "mainnet" {
		cfg = decimal.Mainnet()
	} else if *network != "testnet" {
		fmt.Fprintln(os.Stderr, "unknown network")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, e := decimal.Dial(ctx, cfg)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	defer client.Close()
	block, e := client.BlockNumber(ctx)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Printf("network=%s chain_id=%s block=%d\n", *network, client.ChainID(), block)
	if *codeAddress != "" {
		address, e := decimal.ParseAddress(*codeAddress)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(2)
		}
		code, e := client.ContractCode(ctx, address, new(big.Int).SetUint64(block))
		if e != nil || len(code) == 0 {
			fmt.Fprintln(os.Stderr, "contract code unavailable")
			os.Exit(1)
		}
		hash := crypto.Keccak256Hash(code)
		fmt.Printf("contract=%s runtime_keccak256=%s bytes=%d\n", address.Hex(), hash.Hex(), len(code))
		if *expected != "" {
			raw, e := hex.DecodeString(strings.TrimPrefix(*expected, "0x"))
			if e != nil || len(raw) != 32 {
				fmt.Fprintln(os.Stderr, "invalid expected runtime hash")
				os.Exit(2)
			}
			if hash != common.BytesToHash(raw) {
				fmt.Fprintln(os.Stderr, "runtime hash mismatch")
				os.Exit(1)
			}
		}
	} else if *expected != "" {
		fmt.Fprintln(os.Stderr, "expected-runtime requires code address")
		os.Exit(2)
	}
}
