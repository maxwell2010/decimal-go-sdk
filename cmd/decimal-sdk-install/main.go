// Command decimal-sdk-install installs the modules advertised by the project release manifest.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const manifestURL = "https://raw.githubusercontent.com/maxwell2010/decimal-go-sdk/main/latest.json"
const module = "github.com/maxwell2010/decimal-go-sdk"
const manifestLimit = 64 << 10

type manifest struct {
	Schema      int    `json:"schema"`
	Version     string `json:"version"`
	Go          string `json:"go"`
	EVM         string `json:"evm"`
	Cosmos      string `json:"cosmos"`
	ReleaseURL  string `json:"release_url"`
	SourceZIP   string `json:"source_zip"`
	SourceTarGZ string `json:"source_tar_gz"`
	Checksums   string `json:"checksums"`
}

var versionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
var goPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func canonicalVersion(v string) bool {
	m := versionPattern.FindStringSubmatch(v)
	if m == nil {
		return false
	}
	for _, id := range strings.Split(m[4], ".") {
		if len(id) > 1 && id[0] == '0' && strings.Trim(id, "0123456789") == "" {
			return false
		}
	}
	return true
}

func (m manifest) validate() error {
	if m.Schema != 1 || !canonicalVersion(m.Version) || !goPattern.MatchString(m.Go) {
		return errors.New("unsupported manifest schema or invalid version")
	}
	// Fixed module paths support Go module majors zero and one only.
	major := versionPattern.FindStringSubmatch(m.Version)[1]
	if major != "0" && major != "1" {
		return errors.New("unsupported module major; versions 2 and later require versioned module paths")
	}
	if m.EVM != module+"@"+m.Version || m.Cosmos != module+"/cosmos@"+m.Version {
		return errors.New("unexpected SDK module paths or versions")
	}
	base := "https://github.com/maxwell2010/decimal-go-sdk/releases/"
	asset := base + "download/" + m.Version + "/"
	stem := "decimal-go-sdk-" + strings.TrimPrefix(m.Version, "v")
	if m.ReleaseURL != base+"tag/"+m.Version || m.SourceZIP != asset+stem+".zip" || m.SourceTarGZ != asset+stem+".tar.gz" || m.Checksums != asset+"SHA256SUMS" {
		return errors.New("unexpected release asset URLs")
	}
	return nil
}

func fetchManifest(ctx context.Context, client *http.Client, url string) (manifest, error) {
	var m manifest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return m, err
	}
	// Copy the client to preserve test transports while always refusing redirects.
	guarded := *client
	guarded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if guarded.Timeout <= 0 {
		guarded.Timeout = 15 * time.Second
	}
	resp, err := guarded.Do(req)
	if err != nil {
		return m, fmt.Errorf("release manifest unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return m, fmt.Errorf("release manifest HTTP status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, manifestLimit+1))
	if err != nil {
		return m, err
	}
	if len(data) > manifestLimit {
		return m, errors.New("release manifest exceeds 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&m); err != nil {
		return m, fmt.Errorf("invalid release manifest: %w", err)
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return m, errors.New("release manifest has trailing JSON")
	}
	if err = m.validate(); err != nil {
		return m, err
	}
	return m, nil
}

type runner func(context.Context, ...string) ([]byte, error)

func runGo(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "go", args...).CombinedOutput()
}

func install(ctx context.Context, args []string, out, errOut io.Writer, client *http.Client, url string, run runner) error {
	flags := flag.NewFlagSet("decimal-sdk-install", flag.ContinueOnError)
	flags.SetOutput(errOut)
	printOnly := flags.Bool("print", false, "print the selected go get command without modifying a module")
	evmOnly := flags.Bool("evm-only", false, "install only the EVM module")
	cosmosOnly := flags.Bool("cosmos-only", false, "install only the Cosmos module")
	flags.Usage = func() {
		fmt.Fprintln(errOut, "Usage: decimal-sdk-install [-print] [-evm-only | -cosmos-only]")
		flags.PrintDefaults()
		fmt.Fprintln(errOut, "Run inside an existing Go module. For a new project: go mod init example.com/myapp")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *evmOnly && *cosmosOnly {
		return errors.New("-evm-only and -cosmos-only are mutually exclusive")
	}
	m, err := fetchManifest(ctx, client, url)
	if err != nil {
		return err
	}
	command := []string{"get"}
	if !*cosmosOnly {
		command = append(command, m.EVM)
	}
	if !*evmOnly {
		command = append(command, m.Cosmos)
	}
	if *printOnly {
		fmt.Fprintln(out, "go "+strings.Join(command, " "))
		return nil
	}
	gomod, err := run(ctx, "env", "GOMOD")
	if err != nil {
		return fmt.Errorf("cannot inspect active Go module: %w", err)
	}
	path := strings.TrimSpace(string(gomod))
	if path == "" || path == os.DevNull || !filepath.IsAbs(path) || filepath.Base(path) != "go.mod" {
		return errors.New("no active Go module; create a project and run go mod init example.com/myapp first")
	}
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		return errors.New("active go.mod is unavailable; run inside an existing Go module")
	}
	fmt.Fprintf(out, "Installing Decimal SDK %s (requires Go %s or newer)\n", m.Version, m.Go)
	result, err := run(ctx, command...)
	if len(result) > 0 {
		fmt.Fprint(out, string(result))
	}
	if err != nil {
		return fmt.Errorf("go get failed: %w", err)
	}
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := install(ctx, os.Args[1:], os.Stdout, os.Stderr, &http.Client{Timeout: 15 * time.Second}, manifestURL, runGo); err != nil {
		fmt.Fprintln(os.Stderr, "decimal-sdk-install:", err)
		os.Exit(1)
	}
}
