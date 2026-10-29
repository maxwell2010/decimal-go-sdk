// abi-metadata-inspect reads deployed Solidity metadata and checks content hashes.
// It never executes fetched code, signs, or broadcasts.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

// CIDv0 is DAG-PB/UnixFS, not SHA256(file). This verifies the single-block file
// encoding specified by ipfs/specs UNIXFS.md; larger DAGs are deliberately refused.
func unixFSNode(body []byte) ([]byte, error) {
	if len(body) == 0 || len(body) > 256*1024 {
		return nil, errors.New("single-block verification requires 1..256KiB")
	}
	length := func(n int) []byte {
		var b [10]byte
		k := binary.PutUvarint(b[:], uint64(n))
		return append([]byte(nil), b[:k]...)
	}
	data := []byte{0x08, 0x02, 0x12}
	data = append(data, length(len(body))...)
	data = append(data, body...)
	data = append(data, 0x18)
	data = append(data, length(len(body))...)
	node := []byte{0x0a}
	node = append(node, length(len(data))...)
	return append(node, data...), nil
}

type fetched struct {
	url  string
	body []byte
}

func fetchAll(ctx context.Context, c *http.Client, gateways []string, cid string, multihash []byte) []fetched {
	result := make(chan fetched, len(gateways))
	for _, g := range gateways {
		go func(g string) {
			out := fetched{url: g + cid}
			if strings.Contains(g, "{cidv1}") {
				v1 := append([]byte{1, 0x70}, multihash...)
				encoded := "b" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(v1))
				out.url = strings.ReplaceAll(g, "{cidv1}", encoded)
			}
			defer func() { result <- out }()
			req, e := http.NewRequestWithContext(ctx, "GET", out.url, nil)
			if e != nil {
				return
			}
			res, e := c.Do(req)
			if e != nil {
				return
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				return
			}
			b, e := io.ReadAll(io.LimitReader(res.Body, 256*1024+1))
			if e == nil && len(b) <= 256*1024 {
				out.body = b
			}
		}(g)
	}
	var outputs []fetched
	for range gateways {
		r := <-result
		if len(r.body) > 0 {
			outputs = append(outputs, r)
		}
	}
	return outputs
}

func base58(b []byte) string {
	alphabet := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	n := new(big.Int).SetBytes(b)
	radix := big.NewInt(58)
	var result []byte
	for n.Sign() > 0 {
		q, r := new(big.Int), new(big.Int)
		q.QuoRem(n, radix, r)
		result = append(result, alphabet[r.Int64()])
		n = q
	}
	for _, v := range b {
		if v != 0 {
			break
		}
		result = append(result, '1')
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}
func main() {
	output := flag.String("out", "docs/deployments/testnet-2026-10-29-metadata", "new evidence output directory")
	gatewayFlag := flag.String("gateway", "", "single reviewed metadata gateway prefix; empty uses public defaults")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
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
	at := big.NewInt(22474794)
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	entries := []map[string]any{}
	dir := *output
	if e = os.MkdirAll(dir, 0755); e != nil {
		panic("output unavailable")
	}
	if _, e = os.Stat(dir + "/metadata.json"); e == nil {
		panic("snapshot exists")
	}
	for role, address := range map[string]string{"delegation": "0x2f29724DCf9e3Fe2864dB03c5310cB368Aa96f09", "delegation-nft": "0xf9790e912a10B1cDDC8Aa3C8c2B6A8205446D4d4", "bridge": "0x06d7ac3874adc1f773c5165e91077c8037b789ff"} {
		row := map[string]any{"role": role, "implementation": address}
		code, e := c.CodeAt(ctx, common.HexToAddress(address), at)
		if e != nil || len(code) < 2 {
			row["error"] = "runtime unavailable"
			entries = append(entries, row)
			continue
		}
		row["code_hash"] = crypto.Keccak256Hash(code).Hex()
		size := int(binary.BigEndian.Uint16(code[len(code)-2:]))
		if size > len(code)-2 || size < 8 {
			row["error"] = "no bounded Solidity metadata trailer"
			entries = append(entries, row)
			continue
		}
		meta := code[len(code)-size-2 : len(code)-2]
		row["cbor_hex"] = hex.EncodeToString(meta)
		prefix := []byte{0x64, 'i', 'p', 'f', 's', 0x58, 0x22}
		i := bytes.Index(meta, prefix)
		if i < 0 || bytes.Count(meta, prefix) != 1 || i+len(prefix)+34 > len(meta) {
			row["error"] = "no unique CIDv0 metadata entry"
			entries = append(entries, row)
			continue
		}
		hash := meta[i+len(prefix) : i+len(prefix)+34]
		if hash[0] != 0x12 || hash[1] != 0x20 {
			row["error"] = "unsupported metadata hash"
			entries = append(entries, row)
			continue
		}
		cid := base58(hash)
		row["cid"] = cid
		gateways := []string{"https://testnet-nft-ipfs.decimalchain.com/ipfs/", "https://{cidv1}.ipfs.dweb.link/", "https://{cidv1}.ipfs.w3s.link/", "https://{cidv1}.ipfs.nftstorage.link/"}
		if *gatewayFlag != "" {
			gateways = []string{*gatewayFlag}
		}
		for _, fetched := range fetchAll(ctx, client, gateways, cid, hash) {
			url, body := fetched.url, fetched.body
			node, e := unixFSNode(body)
			if e != nil {
				row["error"] = "metadata requires unsupported multi-block verification"
				continue
			}
			sum := sha256.Sum256(node)
			if !bytes.Equal(sum[:], hash[2:]) {
				row["error"] = "metadata CID content hash mismatch"
				continue
			}
			var artifact struct {
				Output struct {
					ABI json.RawMessage `json:"abi"`
				} `json:"output"`
			}
			if json.Unmarshal(body, &artifact) != nil || len(artifact.Output.ABI) == 0 {
				row["error"] = "hashed metadata has no ABI"
				continue
			}
			row["gateway"] = url
			row["verified_sha256"] = hex.EncodeToString(sum[:])
			row["verification"] = "CIDv0 single-block UnixFS DAG-PB root hash"
			_ = os.WriteFile(dir+"/"+role+".metadata.json", body, 0644)
			_ = os.WriteFile(dir+"/"+role+".abi.json", artifact.Output.ABI, 0644)
			delete(row, "error")
			break
		}
		if row["verified_sha256"] == nil && row["error"] == nil {
			row["error"] = "CID metadata unavailable from bounded public gateways"
		}
		entries = append(entries, row)
	}
	body, _ := json.MarshalIndent(map[string]any{"chain_id": "202020", "block": at.String(), "entries": entries, "scope": "read-only bytecode metadata content-hash evidence, not execution proof"}, "", "  ")
	_ = os.WriteFile(dir+"/metadata.json", append(body, '\n'), 0644)
	fmt.Println(string(body))
}
