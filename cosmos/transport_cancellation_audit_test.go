package cosmos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc"
)

type cancellationAuditConn struct {
	grpc.ClientConnInterface
	cancel  context.CancelFunc
	failure bool
}

func (c *cancellationAuditConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.failure {
		return errors.New("private transport credential")
	}
	return nil
}
func TestGRPCCancellationDominatesSuccessfulTransport(t *testing.T) {
	rawFixture, err := os.ReadFile("schema/reference-tx.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector map[string]string
	if err := json.Unmarshal(rawFixture, &vector); err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(vector["tx_raw"])
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []bool{false, true} {
		for _, simulate := range []bool{false, true} {
			ctx, cancel := context.WithCancel(context.Background())
			c, _ := NewGRPC(&cancellationAuditConn{cancel: cancel, failure: failure})
			if simulate {
				gas, err := c.Simulate(ctx, raw)
				if !errors.Is(err, context.Canceled) || gas != (GasInfo{}) {
					t.Fatal("simulation cancellation lost", gas, err)
				}
			} else {
				req, err := NewQueryRequest("cosmos.tx.v1beta1.Service", "Simulate")
				if err != nil {
					t.Fatal(err)
				}
				response, err := c.Query(ctx, "cosmos.tx.v1beta1.Service", "Simulate", req)
				if !errors.Is(err, context.Canceled) || response != nil {
					t.Fatal("query cancellation lost", response, err)
				}
			}
			cancel()
		}
	}
	c, _ := NewGRPC(&cancellationAuditConn{failure: true})
	req, _ := NewQueryRequest("cosmos.tx.v1beta1.Service", "Simulate")
	if _, err := c.Query(context.Background(), "cosmos.tx.v1beta1.Service", "Simulate", req); err == nil || strings.Contains(err.Error(), "credential") {
		t.Fatal("unredacted transport error", err)
	}
	if _, err := c.Query(nil, "cosmos.tx.v1beta1.Service", "Simulate", req); err == nil {
		t.Fatal("nil context accepted")
	}
}

type cancellationAuditRoundTripper func(*http.Request) (*http.Response, error)

func (f cancellationAuditRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type cancellationAuditBody struct {
	io.Reader
	cancel  context.CancelFunc
	failure bool
	closed  bool
}

func (b *cancellationAuditBody) Read(p []byte) (int, error) {
	if b.cancel != nil {
		b.cancel()
	}
	if b.failure {
		return 0, errors.New("private body credential")
	}
	return b.Reader.Read(p)
}
func (b *cancellationAuditBody) Close() error { b.closed = true; return nil }

type cancellationAuditJSON struct {
	cancel  context.CancelFunc
	failure bool
}

func (j *cancellationAuditJSON) UnmarshalJSON([]byte) error {
	j.cancel()
	if j.failure {
		return errors.New("private JSON credential")
	}
	return nil
}

func TestRESTCancellationDominatesResponseBoundaries(t *testing.T) {
	for _, stage := range []string{"response", "body", "decode"} {
		for _, failure := range []bool{false, true} {
			t.Run(stage+map[bool]string{false: "/success", true: "/failure"}[failure], func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				body := &cancellationAuditBody{Reader: strings.NewReader(`{"account":{"address":"d0fixture","account_number":"7","sequence":"9"}}`)}
				if stage == "body" {
					body.cancel = cancel
					body.failure = failure
				}
				h := &http.Client{Transport: cancellationAuditRoundTripper(func(r *http.Request) (*http.Response, error) {
					status := 200
					if stage == "response" {
						cancel()
						if failure {
							status = 503
						}
					}
					return &http.Response{StatusCode: status, Body: body, Header: make(http.Header), Request: r}, nil
				})}
				c, err := NewREST("https://offline.invalid/", h)
				if err != nil {
					t.Fatal(err)
				}
				if stage == "decode" {
					output := &cancellationAuditJSON{cancel: cancel, failure: failure}
					err = c.request(ctx, http.MethodGet, "fixture", nil, nil, output)
				} else {
					account, e := c.Account(ctx, "d0fixture")
					err = e
					if account != (Account{}) {
						t.Fatal("partial account after cancellation", account)
					}
				}
				if !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation lost", err)
				}
				if !body.closed {
					t.Fatal("response body leaked")
				}
			})
		}
	}
}
func TestRESTTransportAndBodyErrorsRemainRedacted(t *testing.T) {
	for _, bodyFailure := range []bool{false, true} {
		body := &cancellationAuditBody{Reader: strings.NewReader(""), failure: true}
		c, _ := NewREST("https://offline.invalid/", &http.Client{Transport: cancellationAuditRoundTripper(func(r *http.Request) (*http.Response, error) {
			if !bodyFailure {
				return nil, errors.New("private transport credential")
			}
			return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header), Request: r}, nil
		})})
		_, err := c.Account(context.Background(), "d0fixture")
		if err == nil || strings.Contains(err.Error(), "credential") {
			t.Fatal("unredacted error", err)
		}
		if bodyFailure && !body.closed {
			t.Fatal("response body leaked")
		}
	}
}
