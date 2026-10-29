package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type captureTransport struct {
	calls int
	body  string
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.calls++
	b, _ := io.ReadAll(r.Body)
	c.body = string(b)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
}
func TestReadOnlyTransportRejectsWritesBeforeNetwork(t *testing.T) {
	for _, body := range []string{`{"method":"eth_sendRawTransaction","params":["0x1234"]}`, `{"method":"eth_sendTransaction"}`, `{"method":"personal_sign"}`, `[{"method":"eth_call"},{"method":"eth_sendTransaction"}]`, `{"method":"evm_mine"}`, `{`, strings.Repeat("x", 65537)} {
		base := &captureTransport{}
		req, _ := http.NewRequest("POST", "https://example.invalid", strings.NewReader(body))
		_, e := (readTransport{base}).RoundTrip(req)
		if e == nil || base.calls != 0 {
			t.Fatal("unsafe request reached network")
		}
	}
}
func TestReadOnlyTransportPreservesAllowedRequest(t *testing.T) {
	for method := range allowed {
		body := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":["0x1"]}`
		base := &captureTransport{}
		req, _ := http.NewRequest("POST", "https://example.invalid", strings.NewReader(body))
		res, e := (readTransport{base}).RoundTrip(req)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if base.calls != 1 || base.body != body {
			t.Fatal("request changed")
		}
	}
}

func TestRawBlockHashPreservesNodeIdentity(t *testing.T) {
	expected := common.HexToHash("0x123456")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":{"hash":%q,"number":"0x4d"}}`, expected.Hex())
	}))
	defer server.Close()
	client, e := rpc.DialHTTP(server.URL)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	actual, e := rawBlockHash(context.Background(), client, 77)
	if e != nil || actual != expected {
		t.Fatal("node hash lost", e)
	}
}

func TestRawBlockHashRejectsMissingWrongOrInvalidHeight(t *testing.T) {
	hash := common.HexToHash("0x123456").Hex()
	for _, number := range []string{"", `,"number":null`, `,"number":"0x4e"`, `,"number":"0x10000000000000000"`, `,"number":"77"`, `,"number":"0x04d"`, `,"number":77`} {
		t.Run(number, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":{"hash":%q%s}}`, hash, number)
			}))
			defer server.Close()
			client, e := rpc.DialHTTP(server.URL)
			if e != nil {
				t.Fatal(e)
			}
			defer client.Close()
			if _, e = rawBlockHash(context.Background(), client, 77); e == nil {
				t.Fatal("unbound block hash accepted")
			}
		})
	}
}
