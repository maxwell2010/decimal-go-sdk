package cosmos

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type REST struct {
	base *url.URL
	http *http.Client
}

func NewREST(base string, h *http.Client) (*REST, error) {
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("invalid Cosmos REST URL")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/"
	if h == nil {
		h = &http.Client{Timeout: 15 * time.Second}
	}
	client := *h
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &REST{u, &client}, nil
}
func (r *REST) request(ctx context.Context, method, path string, q url.Values, body, out any) error {
	u := *r.base
	u.Path += path
	u.RawQuery = q.Encode()
	var b []byte
	var e error
	if body != nil {
		b, e = json.Marshal(body)
		if e != nil {
			return e
		}
	}
	req, e := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(b))
	if e != nil {
		return errors.New("invalid REST request")
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := r.http.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("Cosmos REST request failed")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New("Cosmos REST non-success HTTP status")
	}
	const limit = 4 << 20
	raw, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil || len(raw) > limit {
		return errors.New("Cosmos REST response exceeds limit")
	}
	if e = json.Unmarshal(raw, out); e != nil {
		return errors.New("invalid Cosmos REST JSON")
	}
	return nil
}

type Account struct {
	Address          string
	Number, Sequence uint64
}

func (r *REST) Account(ctx context.Context, address string) (Account, error) {
	if address == "" || strings.ContainsAny(address, "/\\?#%") || address == "." || address == ".." {
		return Account{}, errors.New("invalid account path")
	}
	var response struct {
		Account json.RawMessage `json:"account"`
	}
	if e := r.request(ctx, http.MethodGet, "cosmos/auth/v1beta1/accounts/"+address, nil, nil, &response); e != nil {
		return Account{}, e
	}
	raw := response.Account
	for depth := 0; depth < 8; depth++ {
		var fields map[string]json.RawMessage
		if e := json.Unmarshal(raw, &fields); e != nil {
			return Account{}, errors.New("invalid account payload")
		}
		if n, ok := fields["account_number"]; ok {
			var number, sequence, actual string
			if json.Unmarshal(n, &number) != nil || json.Unmarshal(fields["sequence"], &sequence) != nil || json.Unmarshal(fields["address"], &actual) != nil || actual != address {
				return Account{}, errors.New("account response identity mismatch")
			}
			accountNumber, e := strconv.ParseUint(number, 10, 64)
			if e != nil {
				return Account{}, e
			}
			seq, e := strconv.ParseUint(sequence, 10, 64)
			if e != nil {
				return Account{}, e
			}
			return Account{actual, accountNumber, seq}, nil
		}
		if v, ok := fields["base_account"]; ok {
			raw = v
		} else if v, ok := fields["base_vesting_account"]; ok {
			raw = v
		} else {
			break
		}
	}
	return Account{}, errors.New("unsupported account wrapper")
}

type GasInfo struct {
	Wanted string `json:"gas_wanted"`
	Used   string `json:"gas_used"`
}

func (r *REST) Simulate(ctx context.Context, raw []byte) (GasInfo, error) {
	var result struct {
		GasInfo GasInfo `json:"gas_info"`
	}
	if _, e := Decode("cosmos.tx.v1beta1.TxRaw", raw); e != nil {
		return GasInfo{}, e
	}
	e := r.request(ctx, http.MethodPost, "cosmos/tx/v1beta1/simulate", nil, map[string]string{"tx_bytes": base64.StdEncoding.EncodeToString(raw)}, &result)
	if e != nil {
		return GasInfo{}, e
	}
	if _, e = strconv.ParseUint(result.GasInfo.Used, 10, 64); e != nil {
		return GasInfo{}, errors.New("missing or invalid simulated gas")
	}
	return result.GasInfo, nil
}

// Commission follows the node QueryCalculateCommissionRequest: tx_bytes is HEX,
// unlike Cosmos tx simulate/broadcast JSON where bytes are Base64 encoded.
func (r *REST) Commission(ctx context.Context, raw []byte, denom string) (string, error) {
	if len(raw) == 0 || denom == "" {
		return "", errors.New("transaction and denomination required")
	}
	var result struct {
		Commission string `json:"commission"`
	}
	e := r.request(ctx, http.MethodGet, "decimal/fee/v1/calculate_commission", url.Values{"tx_bytes": {hex.EncodeToString(raw)}, "denom": {denom}}, nil, &result)
	if e != nil {
		return "", e
	}
	if result.Commission == "" {
		return "", errors.New("missing commission")
	}
	for _, c := range result.Commission {
		if c < '0' || c > '9' {
			return "", errors.New("invalid commission")
		}
	}
	return result.Commission, nil
}

type BroadcastResult struct {
	Hash   string `json:"txhash"`
	Code   uint32 `json:"code"`
	Height string `json:"height"`
}
type SubmissionUnknownError struct{ Hash [32]byte }

func (e *SubmissionUnknownError) Error() string {
	return "Cosmos submission outcome unknown; reconcile hash " + hex.EncodeToString(e.Hash[:])
}

// Broadcast submits once using SYNC. Code0 only means CheckTx acceptance, not
// DeliverTx success; query confirmation independently. No retry/failover occurs.
func (r *REST) Broadcast(ctx context.Context, raw []byte) (BroadcastResult, error) {
	snapshot := append([]byte(nil), raw...)
	hash := TransactionHash(snapshot)
	if e := ValidateTxRaw(snapshot); e != nil {
		return BroadcastResult{}, e
	}
	if e := ctx.Err(); e != nil {
		return BroadcastResult{}, e
	}
	var result struct {
		Response *BroadcastResult `json:"tx_response"`
	}
	e := r.request(ctx, http.MethodPost, "cosmos/tx/v1beta1/txs", nil, map[string]string{"tx_bytes": base64.StdEncoding.EncodeToString(snapshot), "mode": "BROADCAST_MODE_SYNC"}, &result)
	if e != nil || result.Response == nil {
		return BroadcastResult{}, &SubmissionUnknownError{hash}
	}
	if !strings.EqualFold(result.Response.Hash, hex.EncodeToString(hash[:])) {
		return *result.Response, &SubmissionUnknownError{hash}
	}
	if result.Response.Code != 0 {
		return *result.Response, errors.New("Cosmos CheckTx rejected transaction")
	}
	return *result.Response, nil
}
func (r *REST) Transaction(ctx context.Context, hash [32]byte) (BroadcastResult, error) {
	var result struct {
		Response *BroadcastResult `json:"tx_response"`
	}
	if e := r.request(ctx, http.MethodGet, "cosmos/tx/v1beta1/txs/"+strings.ToUpper(hex.EncodeToString(hash[:])), nil, nil, &result); e != nil {
		return BroadcastResult{}, e
	}
	if result.Response == nil || !strings.EqualFold(result.Response.Hash, hex.EncodeToString(hash[:])) {
		return BroadcastResult{}, errors.New("transaction confirmation hash mismatch")
	}
	if result.Response.Code != 0 {
		return *result.Response, errors.New("Cosmos DeliverTx failed")
	}
	if e := validateConfirmationHeight(result.Response.Height); e != nil {
		return *result.Response, e
	}
	return *result.Response, nil
}

// A CheckTx-style zero height is not evidence of inclusion in a committed block.
func validateConfirmationHeight(height string) error {
	n, e := strconv.ParseInt(height, 10, 64)
	if e != nil || n <= 0 {
		return errors.New("transaction confirmation requires positive block height")
	}
	return nil
}
