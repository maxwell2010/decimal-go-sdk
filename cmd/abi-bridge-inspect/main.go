// abi-bridge-inspect replays only official read-only GraphQL bridge discovery.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 7 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	type record struct {
		Endpoint string                                `json:"endpoint"`
		Status   int                                   `json:"status,omitempty"`
		Bridges  []struct{ ID, Implementation string } `json:"bridges,omitempty"`
		Error    string                                `json:"error,omitempty"`
	}
	endpoints := []string{}
	for _, network := range []string{"testnet", "mainnet"} {
		for _, name := range []string{"bridge", "bridgeETH", "bridgeBSC"} {
			endpoints = append(endpoints, "https://"+network+"-thegraph.decimalchain.com/subgraphs/name/"+name)
		}
	}
	rows := make(chan record, len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			r := record{Endpoint: endpoint}
			defer func() { rows <- r }()
			req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBufferString(`{"query":"{ bridges { id implementation } }"}`))
			req.Header.Set("Content-Type", "application/json")
			response, e := client.Do(req)
			if e != nil {
				r.Error = "official subgraph request failed"
				return
			}
			defer response.Body.Close()
			r.Status = response.StatusCode
			if r.Status != 200 {
				r.Error = "non-success HTTP status"
				return
			}
			body, e := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			if e != nil {
				r.Error = "response read failed"
				return
			}
			var result struct {
				Data struct {
					Bridges []struct{ ID, Implementation string } `json:"bridges"`
				} `json:"data"`
				Errors json.RawMessage `json:"errors"`
			}
			if json.Unmarshal(body, &result) != nil || len(result.Errors) > 0 {
				r.Error = "invalid or error GraphQL response"
				return
			}
			r.Bridges = result.Data.Bridges
			if r.Bridges == nil {
				r.Error = "bridge dataset unavailable"
			}
		}(endpoint)
	}
	var results []record
	for range endpoints {
		results = append(results, <-rows)
	}
	body, _ := json.MarshalIndent(map[string]any{"source": "https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/src/subgraph/queries.ts#lines-556", "checked_utc": time.Now().UTC().Format(time.RFC3339), "query": "{ bridges { id implementation } }", "scope": "read-only official subgraph discovery; not transactions", "results": results}, "", "  ")
	file, e := os.OpenFile("docs/deployments/testnet-2026-10-29-additional/bridge-discovery.json", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		panic("snapshot exists or unavailable")
	}
	defer file.Close()
	_, _ = file.Write(append(body, '\n'))
	fmt.Println(string(body))
}
