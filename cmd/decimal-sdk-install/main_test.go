package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func validManifest() manifest {
	v := "v0.1.0-preview.2"
	base := "https://github.com/maxwell2010/decimal-go-sdk/releases/"
	asset := base + "download/" + v + "/decimal-go-sdk-0.1.0-preview.2"
	return manifest{1, v, "1.26.10", module + "@" + v, module + "/cosmos@" + v, base + "tag/" + v, asset + ".zip", asset + ".tar.gz", base + "download/" + v + "/SHA256SUMS"}
}
func manifestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
	t.Cleanup(s.Close)
	return s
}
func encodedManifest(t *testing.T) string {
	t.Helper()
	b, e := json.Marshal(validManifest())
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func TestManifestValidation(t *testing.T) {
	for name, mutate := range map[string]func(*manifest){
		"schema":             func(m *manifest) { m.Schema = 2 },
		"injection":          func(m *manifest) { m.Version = "v0.1.0;echo injected" },
		"noncanonical":       func(m *manifest) { m.Version = "v01.1.0" },
		"numeric prerelease": func(m *manifest) { m.Version = "v0.1.0-preview.02" },
		"evm":                func(m *manifest) { m.EVM = "evil.example/sdk@" + m.Version },
		"cosmos":             func(m *manifest) { m.Cosmos = module + "/cosmos@v0.1.0" },
		"url":                func(m *manifest) { m.SourceZIP += "?download=1" },
		"go":                 func(m *manifest) { m.Go = "1.26.10 -exec=bad" },
	} {
		t.Run(name, func(t *testing.T) {
			m := validManifest()
			mutate(&m)
			if m.validate() == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
	for _, v := range []string{"v0.1.0-preview.2", "v1.0.0", "v1.2.3-rc.0"} {
		if !canonicalVersion(v) {
			t.Fatal(v)
		}
	}
}

func TestFetchBoundsTrailingAndRedirect(t *testing.T) {
	for name, body := range map[string]string{"oversize": strings.Repeat(" ", manifestLimit+1), "trailing": encodedManifest(t) + " {}", "malformed": "{", "unknown": strings.TrimSuffix(encodedManifest(t), "}") + `,"extra":1}`} {
		t.Run(name, func(t *testing.T) {
			s := manifestServer(t, body)
			if _, e := fetchManifest(context.Background(), s.Client(), s.URL); e == nil {
				t.Fatal("accepted invalid response")
			}
		})
	}
	hits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; io.WriteString(w, encodedManifest(t)) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	if _, e := fetchManifest(context.Background(), redirect.Client(), redirect.URL); e == nil {
		t.Fatal("accepted redirect")
	}
	if hits != 0 {
		t.Fatal("followed redirect")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := fetchManifest(ctx, target.Client(), target.URL); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

func TestInstallerCommandAndFlags(t *testing.T) {
	s := manifestServer(t, encodedManifest(t))
	mod := filepath.Join(t.TempDir(), "go.mod")
	if e := os.WriteFile(mod, []byte("module example.invalid/app\n"), 0600); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ args, want []string }{{nil, []string{"get", validManifest().EVM, validManifest().Cosmos}}, {[]string{"-evm-only"}, []string{"get", validManifest().EVM}}, {[]string{"-cosmos-only"}, []string{"get", validManifest().Cosmos}}} {
		var calls [][]string
		run := func(ctx context.Context, args ...string) ([]byte, error) {
			calls = append(calls, append([]string(nil), args...))
			if len(calls) == 1 {
				return []byte(mod + "\n"), nil
			}
			return []byte("installed fixture\n"), nil
		}
		var out bytes.Buffer
		if e := install(context.Background(), tc.args, &out, io.Discard, s.Client(), s.URL, run); e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(calls, [][]string{{"env", "GOMOD"}, tc.want}) {
			t.Fatalf("arguments %v", calls)
		}
	}
	never := func(context.Context, ...string) ([]byte, error) {
		t.Fatal("print/help/invalid flags invoked runner")
		return nil, nil
	}
	var out bytes.Buffer
	if e := install(context.Background(), []string{"-print"}, &out, io.Discard, s.Client(), s.URL, never); e != nil {
		t.Fatal(e)
	}
	if out.String() != "go get "+validManifest().EVM+" "+validManifest().Cosmos+"\n" {
		t.Fatal(out.String())
	}
	if e := install(context.Background(), []string{"-help"}, io.Discard, io.Discard, s.Client(), "bad URL", never); e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]string{{"-evm-only", "-cosmos-only"}, {"-url", "evil"}, {"unexpected"}} {
		if e := install(context.Background(), args, io.Discard, io.Discard, s.Client(), "bad URL", never); e == nil {
			t.Fatal("invalid flags accepted")
		}
	}
}

func TestInstallerRejectsMissingModuleAndGoFailure(t *testing.T) {
	s := manifestServer(t, encodedManifest(t))
	for _, mod := range []string{"", os.DevNull, "relative/go.mod", filepath.Join(t.TempDir(), "go.mod")} {
		calls := 0
		run := func(context.Context, ...string) ([]byte, error) { calls++; return []byte(mod), nil }
		if e := install(context.Background(), nil, io.Discard, io.Discard, s.Client(), s.URL, run); e == nil {
			t.Fatal("missing module accepted")
		}
		if calls != 1 {
			t.Fatal("go get invoked without module")
		}
	}
	run := func(context.Context, ...string) ([]byte, error) { return nil, errors.New("no go executable") }
	if e := install(context.Background(), nil, io.Discard, io.Discard, s.Client(), s.URL, run); e == nil {
		t.Fatal("runner failure swallowed")
	}
}

func TestManifestRejectsUnsupportedModuleMajor(t *testing.T) {
	for _, v := range []string{"v2.0.0", "v12.3.4-preview.1", "v999999999999999999999.0.0"} {
		m := validManifest()
		old := m.Version
		m.Version = v
		m.EVM = module + "@" + v
		m.Cosmos = module + "/cosmos@" + v
		m.ReleaseURL = strings.ReplaceAll(m.ReleaseURL, old, v)
		m.SourceZIP = strings.ReplaceAll(strings.ReplaceAll(m.SourceZIP, old, v), strings.TrimPrefix(old, "v"), strings.TrimPrefix(v, "v"))
		m.SourceTarGZ = strings.ReplaceAll(strings.ReplaceAll(m.SourceTarGZ, old, v), strings.TrimPrefix(old, "v"), strings.TrimPrefix(v, "v"))
		m.Checksums = strings.ReplaceAll(m.Checksums, old, v)
		if err := m.validate(); err == nil || !strings.Contains(err.Error(), "unsupported module major") {
			t.Fatalf("major %s: %v", v, err)
		}
	}
}

func TestManifestFetchBypassesMovingChannelCache(t *testing.T) {
	expected := validManifest()
	body := encodedManifest(t)
	requests := make(chan *http.Request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(context.Background())
		if r.URL.Query().Get("_sdk_update") == "" || r.Header.Get("Cache-Control") != "no-cache" || r.Header.Get("Pragma") != "no-cache" {
			http.Error(w, "stale cache path", http.StatusPreconditionFailed)
			return
		}
		io.WriteString(w, body)
	}))
	defer server.Close()
	var previous string
	for range 2 {
		actual, err := fetchManifest(context.Background(), server.Client(), server.URL+"/channel/latest.json?existing=kept")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("manifest changed: %+v", actual)
		}
		request := <-requests
		if request.URL.Path != "/channel/latest.json" || request.Host != strings.TrimPrefix(server.URL, "http://") || request.URL.Query().Get("existing") != "kept" {
			t.Fatalf("trusted request target changed: %s", request.URL)
		}
		query := request.URL.Query()
		if len(query) != 2 || len(query["_sdk_update"]) != 1 {
			t.Fatalf("unexpected query: %v", query)
		}
		nonce := query.Get("_sdk_update")
		if decoded, err := hex.DecodeString(nonce); err != nil || len(decoded) != 16 {
			t.Fatalf("invalid freshness nonce: %q", nonce)
		}
		if nonce == previous {
			t.Fatal("freshness nonce reused")
		}
		previous = nonce
	}
}
