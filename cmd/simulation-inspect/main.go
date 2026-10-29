// simulation-inspect performs bounded, block-pinned eth_call probes only.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"io"
	"math/big"
	"net/http"
	"os"
	"time"
)

var allowed = map[string]bool{"eth_chainId": true, "eth_blockNumber": true, "eth_getBlockByNumber": true, "eth_getBalance": true, "eth_getCode": true, "eth_getStorageAt": true, "eth_call": true}

type readTransport struct{ base http.RoundTripper }

func (t readTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	data, e := io.ReadAll(io.LimitReader(req.Body, 65537))
	if e != nil {
		return nil, e
	}
	req.Body.Close()
	var message struct {
		Method string `json:"method"`
	}
	if len(data) > 65536 || json.Unmarshal(data, &message) != nil || !allowed[message.Method] {
		return nil, errors.New("non-allowlisted RPC method refused")
	}
	req.Body = io.NopCloser(bytes.NewReader(data))
	return t.base.RoundTrip(req)
}

type observation struct {
	Route                                                                                                                string `json:"route"`
	Status                                                                                                               string `json:"status"`
	Reason                                                                                                               string `json:"reason,omitempty"`
	Target, From, RuntimeHash, Implementation, ImplementationHash, InputSource, Value, Data, ReturnData, ObservedBalance string
}
type report struct {
	Network, ChainID, RPC, BlockHash string
	Block                            uint64
	Consistent                       bool
	Observations                     []observation
}

func errorText(e error) string {
	var r *decimal.RevertError
	if errors.As(e, &r) {
		return r.Error() + " " + r.Reason
	}
	return e.Error()
}
func native(ctx context.Context, c *decimal.Client, r *ethclient.Client, block uint64, chain *big.Int) observation {
	row := observation{Route: "native-self-transfer-1-wei", Status: "not-tested", Reason: "no funded EOA found in at most 32 recent blocks"}
	for offset := uint64(0); offset < 32 && offset <= block; offset++ {
		b, e := r.BlockByNumber(ctx, new(big.Int).SetUint64(block-offset))
		if e != nil {
			row.Reason = "bounded block lookup failed"
			continue
		}
		for _, tx := range b.Transactions() {
			from, e := types.Sender(types.LatestSignerForChainID(chain), tx)
			if e != nil {
				continue
			}
			balance, e := r.BalanceAt(ctx, from, new(big.Int).SetUint64(block))
			if e != nil || balance.Cmp(big.NewInt(1)) < 0 {
				continue
			}
			code, e := r.CodeAt(ctx, from, new(big.Int).SetUint64(block))
			if e != nil || len(code) != 0 {
				continue
			}
			call, e := decimal.NativeTransfer(from, big.NewInt(1), "")
			if e != nil {
				row.Reason = e.Error()
				return row
			}
			row.From = from.Hex()
			row.Target = from.Hex()
			row.Value = "1"
			row.Data = "0x"
			row.ObservedBalance = balance.String()
			row.InputSource = "sender recovered from public transaction " + tx.Hash().Hex() + " at block " + b.Number().String() + "; balance and empty code re-read at snapshot; account control not claimed"
			output, e := c.Simulate(ctx, from, call, new(big.Int).SetUint64(block))
			if e != nil {
				row.Status = "failed"
				row.Reason = errorText(e)
			} else {
				row.Status = "eth_call-success"
				row.Reason = "self-transfer only; no signing, nonce admission, external-recipient flow or settlement proof"
				row.ReturnData = "0x" + common.Bytes2Hex(output)
			}
			return row
		}
	}
	return row
}
func token(ctx context.Context, c *decimal.Client, r *ethclient.Client, block uint64, profile, address, expected string) observation {
	row := observation{Route: profile + "-self-transfer-1-raw", Status: "not-tested", Target: address, Value: "0"}
	raw, e := decimal.ABIJSON(profile)
	if e != nil {
		row.Reason = e.Error()
		return row
	}
	bound, e := c.BindBeaconABI(ctx, common.HexToAddress(address), block, raw)
	if e != nil {
		row.Reason = e.Error()
		return row
	}
	row.RuntimeHash = bound.Evidence.ProxyCodeHash.Hex()
	row.Implementation = bound.Evidence.Implementation.Hex()
	row.ImplementationHash = bound.Evidence.ImplementationCodeHash.Hex()
	if row.ImplementationHash != expected {
		row.Reason = "implementation hash differs from reviewed token manifest"
		return row
	}
	getter := "owner()"
	if profile == "TestnetReserveToken20261029" {
		getter = "creator()"
	}
	values, e := bound.Contract.ReadAt(ctx, new(big.Int).SetUint64(block), getter)
	if e != nil || len(values) != 1 {
		row.Reason = getter + " getter unavailable"
		return row
	}
	owner, ok := values[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		row.Reason = "no nonzero owner"
		return row
	}
	values, e = bound.Contract.ReadAt(ctx, new(big.Int).SetUint64(block), "balanceOf(address)", owner)
	if e != nil || len(values) != 1 {
		row.Reason = "owner token balance unavailable"
		return row
	}
	balance, ok := values[0].(*big.Int)
	if !ok || balance.Sign() <= 0 {
		row.Reason = "owner has no positive token balance"
		return row
	}
	call, e := bound.Contract.Build("transfer(address,uint256)", nil, owner, big.NewInt(1))
	if e != nil {
		row.Reason = e.Error()
		return row
	}
	row.From = owner.Hex()
	row.ObservedBalance = balance.String()
	row.InputSource = getter + " and balanceOf(account) at snapshot; beacon binding and implementation hash match reviewed testnet manifest; no control of owner claimed"
	row.Data = "0x" + common.Bytes2Hex(call.Data)
	out, e := bound.Contract.Simulate(ctx, owner, call, new(big.Int).SetUint64(block))
	if e != nil {
		row.Status = "failed"
		row.Reason = errorText(e)
	} else {
		row.ReturnData = "0x" + common.Bytes2Hex(out)
		if !bytes.Equal(out, common.LeftPadBytes([]byte{1}, 32)) {
			row.Status = "unexpected-return"
			row.Reason = "transfer did not return canonical true"
		} else {
			row.Status = "eth_call-success"
			row.Reason = "positive token self-transfer only; no external recipient, allowance, permit, signing or settlement acceptance"
		}
	}
	return row
}

// Preserve the RPC hash: Decimal's reported block hash need not equal the
// Ethereum Header.Hash recomputation of fields decoded by go-ethereum.
func rawBlockHash(ctx context.Context, client *rpc.Client, height uint64) (common.Hash, error) {
	var response *struct {
		Hash   common.Hash     `json:"hash"`
		Number *hexutil.Uint64 `json:"number"`
	}
	if e := client.CallContext(ctx, &response, "eth_getBlockByNumber", fmt.Sprintf("0x%x", height), false); e != nil {
		return common.Hash{}, e
	}
	if response == nil || response.Hash == (common.Hash{}) || response.Number == nil || uint64(*response.Number) != height {
		return common.Hash{}, errors.New("missing or mismatched canonical RPC block identity")
	}
	return response.Hash, nil
}
func main() {
	network := flag.String("network", "testnet", "testnet or mainnet")
	height := flag.Uint64("block", 0, "explicit block; zero selects current head once")
	out := flag.String("out", "", "new output file (never overwrite)")
	flag.Parse()
	cfg := decimal.Testnet()
	if *network == "mainnet" {
		cfg = decimal.Mainnet()
	} else if *network != "testnet" {
		panic("invalid network")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	rc, e := rpc.DialOptions(ctx, cfg.RPCURL, rpc.WithHTTPClient(&http.Client{Timeout: 12 * time.Second, Transport: readTransport{http.DefaultTransport}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
	if e != nil {
		panic("RPC unavailable")
	}
	defer rc.Close()
	r := ethclient.NewClient(rc)
	id, e := r.ChainID(ctx)
	if e != nil || id.Cmp(cfg.ChainID) != 0 {
		panic("chain mismatch")
	}
	block := *height
	if block == 0 {
		block, e = r.BlockNumber(ctx)
		if e != nil {
			panic("head unavailable")
		}
	}
	hash, e := rawBlockHash(ctx, rc, block)
	if e != nil {
		panic("snapshot header unavailable")
	}
	c, e := decimal.NewClient(cfg, r)
	if e != nil {
		panic(e)
	}
	result := report{Network: *network, ChainID: id.String(), RPC: cfg.RPCURL, Block: block, BlockHash: hash.Hex()}
	result.Observations = append(result.Observations, native(ctx, c, r, block, id))
	if *network == "testnet" {
		for _, spec := range [][3]string{{"TestnetReserveToken20261029", "0x37748f3e857a71e4334125a935052ce1e603b7b5", "0x67ef16857177e434b860586efd9ff7cf2c57c1adf4ba50bd9310f42f9c8c2cf1"}, {"TestnetReservelessToken20261029", "0xc310f25e3429efc5ede11a908682f0a31cfc92ce", "0x67e10916520447faad79c557c719042ba1e12b95ca57e1c18b7a5ecfaca4c1fd"}} {
			result.Observations = append(result.Observations, token(ctx, c, r, block, spec[0], spec[1], spec[2]))
		}
	} else {
		result.Observations = append(result.Observations, observation{Route: "token-write-simulations", Status: "not-tested", Reason: "testnet versioned token ABI not adopted for mainnet"})
	}
	result.Observations = append(result.Observations, observation{Route: "checks-create/redeem", Status: "not-tested", Reason: "creation nonce/funding/expiry context not established; redemption has no authorized proof; no secrets/signing or inferred nonce semantics"}, observation{Route: "privileged/bridge/NFT/staking", Status: "not-tested", Reason: "valid state, proof, role or exact deployment ABI not established by these probes"})
	again, e := rawBlockHash(ctx, rc, block)
	result.Consistent = e == nil && again == hash
	data, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		panic(e)
	}
	data = append(data, '\n')
	if *out == "" {
		fmt.Print(string(data))
	} else {
		f, e := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if e != nil {
			panic(e)
		}
		defer f.Close()
		if _, e = f.Write(data); e != nil {
			panic(e)
		}
	}
	if !result.Consistent {
		fmt.Fprintln(os.Stderr, "snapshot changed/unavailable; observations not accepted")
		os.Exit(1)
	}

}
