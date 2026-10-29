package decimal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type WithdrawalIndex struct {
	Block   *big.Int
	Entries []WithdrawalIndexEntry
}
type WithdrawalIndexEntry struct {
	Contract common.Address
	Index    *big.Int
}

func jsonInteger(raw json.RawMessage) (*big.Int, error) {
	var s string
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, errors.New("missing indexer integer")
	}
	if raw[0] == '"' {
		if e := json.Unmarshal(raw, &s); e != nil {
			return nil, e
		}
	} else {
		s = string(raw)
	}
	if s == "" {
		return nil, errors.New("invalid indexer integer")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return nil, errors.New("invalid indexer integer")
		}
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok || !uint256(n) {
		return nil, errors.New("invalid indexer integer")
	}
	return n, nil
}

// WithdrawalIndex reads a user-configured authoritative indexer, never a hardcoded
// application service. Empty, unavailable, incomplete and stale are distinct.
// Pass a relative path on the API instance configured for that indexer.
func (a *API) WithdrawalIndex(ctx context.Context, path string, minBlock *big.Int) (WithdrawalIndex, error) {
	if minBlock != nil && !uint256(minBlock) {
		return WithdrawalIndex{}, errors.New("invalid minimum indexer block")
	}
	var response struct {
		Result struct {
			Authoritative *bool           `json:"withdrawals_authoritative"`
			Fresh         *bool           `json:"withdrawals_fresh"`
			Unavailable   bool            `json:"withdrawals_unavailable"`
			Block         json.RawMessage `json:"withdrawals_checked_block"`
			Entries       *[]struct {
				Contract string          `json:"contract_address"`
				Index    json.RawMessage `json:"id"`
			} `json:"withdrawals"`
		} `json:"Result"`
	}
	if e := a.Get(ctx, path, nil, &response); e != nil {
		return WithdrawalIndex{}, e
	}
	r := response.Result
	if r.Unavailable || r.Authoritative == nil || !*r.Authoritative || r.Fresh == nil || !*r.Fresh {
		return WithdrawalIndex{}, errors.New("withdrawal index unavailable, incomplete or stale")
	}
	block, e := jsonInteger(r.Block)
	if e != nil {
		return WithdrawalIndex{}, e
	}
	if minBlock != nil && block.Cmp(minBlock) < 0 {
		return WithdrawalIndex{}, errors.New("withdrawal index is older than required block")
	}
	if r.Entries == nil {
		return WithdrawalIndex{}, errors.New("withdrawal entries missing or null")
	}
	entries := *r.Entries
	if len(entries) > 10000 {
		return WithdrawalIndex{}, errors.New("withdrawal index too large")
	}
	result := WithdrawalIndex{Block: block, Entries: make([]WithdrawalIndexEntry, 0, len(entries))}
	seen := map[string]bool{}
	for _, row := range entries {
		address, e := ParseAddress(row.Contract)
		if e != nil || address == (common.Address{}) {
			return WithdrawalIndex{}, errors.New("invalid withdrawal contract")
		}
		id, e := jsonInteger(row.Index)
		if e != nil {
			return WithdrawalIndex{}, e
		}
		key := address.Hex() + ":" + id.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		result.Entries = append(result.Entries, WithdrawalIndexEntry{address, id})
	}
	return result, nil
}
