// abi-source-inspect tests only source-documented read endpoints and GraphQL introspection.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	bridgeOnly := flag.Bool("bridge-only", false, "inspect the discovered Bridge type only")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	c := &http.Client{Timeout: 7 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	type input struct{ URL, Query, Source string }
	tasks := []input{}
	for _, address := range []string{"0x06d7ac3874adc1f773c5165e91077c8037b789ff", "0x2f29724dcf9e3fe2864db03c5310cb368aa96f09", "0xf9790e912a10b1cddc8aa3c8c2b6a8205446d4d4"} {
		tasks = append(tasks, input{"https://testnet-gate.decimalchain.com/api/evm-contracts/" + address, "", "dsc-go-sdk constants.go TestnetGate and commented endpoints.go GetEvmContract"})
	}
	tasks = append(tasks, input{"https://testnet-thegraph.decimalchain.com/subgraphs/name/bridge", `{ __type(name:"Query") { fields { name type { kind name ofType { kind name ofType { kind name } } } } } }`, "official JS Bridge subgraph"})
	if *bridgeOnly {
		tasks = []input{{"https://testnet-thegraph.decimalchain.com/subgraphs/name/bridge", `{ __type(name:"Bridge") { fields { name type { kind name } } } }`, "official JS Bridge subgraph discovered Bridge type"}}
	}
	out := make(chan map[string]any, len(tasks))
	for _, task := range tasks {
		go func(task input) {
			r := map[string]any{"url": task.URL, "source": task.Source}
			defer func() { out <- r }()
			var req *http.Request
			if task.Query != "" {
				body, _ := json.Marshal(map[string]string{"query": task.Query})
				req, _ = http.NewRequestWithContext(ctx, "POST", task.URL, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequestWithContext(ctx, "GET", task.URL, nil)
			}
			response, e := c.Do(req)
			if e != nil {
				r["error"] = "request failed"
				return
			}
			defer response.Body.Close()
			r["status"] = response.StatusCode
			if response.StatusCode != 200 {
				r["error"] = "non-success status"
				return
			}
			raw, e := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			if e != nil {
				r["error"] = "body unavailable"
				return
			}
			var decoded map[string]json.RawMessage
			if json.Unmarshal(raw, &decoded) != nil {
				r["error"] = "invalid JSON"
				return
			}
			keys := []string{}
			for k := range decoded {
				keys = append(keys, k)
			}
			r["root_keys"] = keys
			if task.Query != "" {
				r["query"] = task.Query
				if len(decoded["errors"]) > 0 {
					r["error"] = "GraphQL error"
				} else {
					r["schema_data"] = decoded["data"]
				}
			} else {
				r["error"] = "legacy detail endpoint did not provide a reviewed ABI artifact"
			}
		}(task)
	}
	rows := []map[string]any{}
	for range tasks {
		rows = append(rows, <-out)
	}
	raw, _ := json.MarshalIndent(map[string]any{"checked_utc": time.Now().UTC().Format(time.RFC3339), "scope": "documented endpoints only; read-only; no verification endpoint mutation", "results": rows}, "", "  ")
	path := "docs/deployments/testnet-2026-10-29-additional/source-discovery.json"
	if *bridgeOnly {
		path = "docs/deployments/testnet-2026-10-29-additional/bridge-schema.json"
	}
	file, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		panic("snapshot exists or unavailable")
	}
	defer file.Close()
	_, _ = file.Write(append(raw, '\n'))
	fmt.Println(string(raw))
}
