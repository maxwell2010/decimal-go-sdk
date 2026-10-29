// Command checks-inspect reads public historical Checks events; it never signs or sends.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"math/big"
	"os"
	"time"
)

func main() {
	network := flag.String("network", "testnet", "testnet or mainnet")
	address := flag.String("address", "", "explicit Checks address; default selected network configuration")
	to := flag.Uint64("to", 0, "last block (0 resolves once to current head)")
	blocks := flag.Uint64("blocks", 1000, "window size 1..2000")
	flag.Parse()
	cfg := decimal.Testnet()
	if *network == "mainnet" {
		cfg = decimal.Mainnet()
	} else if *network != "testnet" {
		fmt.Fprintln(os.Stderr, "invalid network")
		os.Exit(1)
	}
	if *blocks < 1 || *blocks > 2000 {
		fmt.Fprintln(os.Stderr, "blocks must be 1..2000")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	c, e := decimal.Dial(ctx, cfg)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	defer c.Close()
	target := cfg.Checks
	if *address != "" {
		target, e = decimal.ParseAddress(*address)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	end := *to
	if end == 0 {
		end, e = c.BlockNumber(ctx)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	start := uint64(0)
	if end >= *blocks-1 {
		start = end - *blocks + 1
	}
	raw, e := rpc.DialContext(ctx, cfg.RPCURL)
	if e != nil {
		fmt.Fprintln(os.Stderr, "raw RPC unavailable")
		os.Exit(1)
	}
	defer raw.Close()
	var rawChain hexutil.Big
	if e = raw.CallContext(ctx, &rawChain, "eth_chainId"); e != nil || (*big.Int)(&rawChain).Cmp(cfg.ChainID) != 0 {
		fmt.Fprintln(os.Stderr, "raw RPC chain mismatch")
		os.Exit(1)
	}
	header, e := rawHeader(ctx, raw, end)
	if e != nil || header == nil {
		fmt.Fprintln(os.Stderr, "snapshot header unavailable")
		os.Exit(1)
	}
	rows, queryErr := c.ChecksCreationHistory(ctx, target, start, end)
	again, e := rawHeader(ctx, raw, end)
	if e != nil || again == nil || again.Hash != header.Hash {
		fmt.Fprintln(os.Stderr, "snapshot changed during event query")
		os.Exit(1)
	}
	out := map[string]any{"network": *network, "chain_id": c.ChainID().String(), "rpc": cfg.RPCURL, "address": target.Hex(), "from_block": start, "to_block": end, "to_block_hash": header.Hash.Hex(), "events": rows, "event_count": len(rows), "scope": "historical event evidence only; no current state or finality certification"}
	if queryErr != nil {
		out["error"] = queryErr.Error()
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e = enc.Encode(out); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	if queryErr != nil {
		os.Exit(1)
	}
}

type observedHeader struct {
	Hash   common.Hash
	Number *hexutil.Uint64
}

func rawHeader(ctx context.Context, r *rpc.Client, height uint64) (*observedHeader, error) {
	var h *observedHeader
	e := r.CallContext(ctx, &h, "eth_getBlockByNumber", hexutil.EncodeUint64(height), false)
	if e != nil {
		return nil, e
	}
	if h == nil || h.Number == nil || uint64(*h.Number) != height || h.Hash == (common.Hash{}) {
		return nil, fmt.Errorf("invalid raw block identity")
	}
	return h, nil
}
