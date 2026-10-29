package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// API is an explicit read-only HTTP API. No API key or application service URL
// is embedded in this SDK. The default HTTP client has a 15 second timeout.
type API struct {
	base *url.URL
	http *http.Client
}

func NewAPI(base string, client *http.Client) (*API, error) {
	if !validURL(base) {
		return nil, errors.New("invalid API base URL")
	}
	u, _ := url.Parse(strings.TrimRight(base, "/") + "/")
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("API base cannot contain query or fragment")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &API{u, &copyClient}, nil
}

// Get supports additional official REST routes without baking in undocumented
// response shapes. Bodies are bounded; HTTP failures do not expose response data.
func (a *API) Get(ctx context.Context, path string, query url.Values, out any) error {
	u, e := url.Parse(path)
	if e != nil || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(path, "\\") || strings.HasPrefix(path, "/") {
		return errors.New("API path must be relative")
	}
	for _, segment := range strings.Split(u.Path, "/") {
		if segment == ".." || segment == "." {
			return errors.New("invalid API path")
		}
	}
	u = a.base.ResolveReference(u)
	u.RawQuery = query.Encode()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if e != nil {
		return errors.New("cannot construct API request")
	}
	res, e := a.http.Do(req)
	if e != nil {
		return errors.New("API request failed")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New("API returned non-success HTTP status")
	}
	const limit = 8 << 20
	b, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil || len(b) > limit {
		return errors.New("invalid API response size")
	}
	if e = json.Unmarshal(b, out); e != nil {
		return errors.New("API returned invalid JSON")
	}
	return nil
}

type Page struct{ Limit, Offset int }

func (p Page) Query() (url.Values, error) {
	if p.Limit < 1 || p.Limit > 500 || p.Offset < 0 {
		return nil, errors.New("pagination requires limit 1..500 and nonnegative offset")
	}
	v := url.Values{}
	v.Set("limit", itoa(p.Limit))
	v.Set("offset", itoa(p.Offset))
	return v, nil
}
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	b := make([]byte, 0, 20)
	for i > 0 {
		b = append(b, byte(i%10)+'0')
		i /= 10
	}
	for l, r := 0, len(b)-1; l < r; l, r = l+1, r-1 {
		b[l], b[r] = b[r], b[l]
	}
	return string(b)
}
func (a *API) WalletStakes(ctx context.Context, wallet common.Address, out any) error {
	return a.Get(ctx, "validators/wallet/"+strings.ToLower(wallet.Hex())+"/stakes/coins", nil, out)
}
func (a *API) WalletUnstakes(ctx context.Context, wallet common.Address, out any) error {
	return a.Get(ctx, "validators/"+strings.ToLower(wallet.Hex())+"/unstakes/coins", nil, out)
}
func (a *API) Validators(ctx context.Context, out any) error {
	return a.Get(ctx, "validators", nil, out)
}
func (a *API) Transaction(ctx context.Context, hash common.Hash, out any) error {
	return a.Get(ctx, "txs/"+hash.Hex(), nil, out)
}

// DeploymentABI reads the official compact-contract response; using the result
// remains an explicit ContractWithABI decision by the caller.
func (a *API) DeploymentABI(ctx context.Context, address common.Address) (json.RawMessage, error) {
	var response struct {
		Result struct {
			ABI json.RawMessage `json:"abi"`
		} `json:"Result"`
	}
	if e := a.Get(ctx, "contracts/compact/"+strings.ToLower(address.Hex()), nil, &response); e != nil {
		return nil, e
	}
	raw := response.Result.ABI
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("deployment ABI unavailable")
	}
	if raw[0] == '"' {
		var s string
		if e := json.Unmarshal(raw, &s); e != nil {
			return nil, e
		}
		raw = json.RawMessage(s)
	}
	return raw, nil
}

// LoadDeploymentContract explicitly opts into the ABI returned by this API.
// Chain ID and target code are checked; API provenance is not a proof that a
// proxy's implementation will remain unchanged. Pin/review artifacts when needed.
func (c *Client) LoadDeploymentContract(ctx context.Context, api *API, address common.Address) (*Contract, error) {
	if api == nil {
		return nil, errors.New("explicit ABI API required")
	}
	code, e := c.ContractCode(ctx, address, nil)
	if e != nil {
		return nil, e
	}
	if len(code) == 0 {
		return nil, errors.New("deployment has no code")
	}
	raw, e := api.DeploymentABI(ctx, address)
	if e != nil {
		return nil, e
	}
	contract, e := c.ContractWithABI(address, raw)
	if e != nil {
		return nil, e
	}
	parsed, e := contract.contractABI()
	if e != nil {
		return nil, e
	}
	if len(parsed.Methods) == 0 {
		return nil, errors.New("deployment ABI has no callable methods; proxy implementation ABI must be verified and supplied explicitly")
	}
	return contract, nil
}
