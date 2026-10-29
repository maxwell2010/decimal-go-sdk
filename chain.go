package decimal

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	if e := c.checkChain(ctx); e != nil {
		return 0, e
	}
	r, ok := c.rpc.(interface {
		BlockNumber(context.Context) (uint64, error)
	})
	if !ok {
		return 0, errors.New("backend does not support block numbers")
	}
	n, e := r.BlockNumber(ctx)
	if e != nil {
		return 0, rpcFailure(ctx, "block number unavailable")
	}
	return n, nil
}

// Header returns Ethereum-shaped fields. Header.Hash() recomputes an Ethereum
// hash and may differ from Decimal canonical RPC identity; use BlockIdentity.
func (c *Client) Header(ctx context.Context, number *big.Int) (*types.Header, error) {
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	r, ok := c.rpc.(interface {
		HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	})
	if !ok {
		return nil, errors.New("backend does not support block headers")
	}
	h, e := r.HeaderByNumber(ctx, number)
	if e != nil {
		return nil, rpcFailure(ctx, "block header unavailable")
	}
	return h, nil
}
func (c *Client) ContractCode(ctx context.Context, address common.Address, block *big.Int) ([]byte, error) {
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	b, e := c.rpc.CodeAt(ctx, address, block)
	if e != nil {
		return nil, rpcFailure(ctx, "contract code unavailable")
	}
	return b, nil
}
func (c *Client) Logs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	r, ok := c.rpc.(interface {
		FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
	})
	if !ok {
		return nil, errors.New("backend does not support log filtering")
	}
	logs, e := r.FilterLogs(ctx, query)
	if e != nil {
		return nil, rpcFailure(ctx, "log query failed")
	}
	return logs, nil
}

// SubscribeLogs works with a WebSocket/IPC Backend supplied through NewClient.
// The caller owns the subscription and must Unsubscribe or cancel its context.
func (c *Client) SubscribeLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	r, ok := c.rpc.(interface {
		SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error)
	})
	if !ok {
		return nil, errors.New("backend does not support subscriptions")
	}
	s, e := r.SubscribeFilterLogs(ctx, query, ch)
	if e != nil {
		return nil, rpcFailure(ctx, "log subscription failed")
	}
	return s, nil
}
func (c *Client) SubscribeHeads(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	r, ok := c.rpc.(interface {
		SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error)
	})
	if !ok {
		return nil, errors.New("backend does not support subscriptions")
	}
	s, e := r.SubscribeNewHead(ctx, ch)
	if e != nil {
		return nil, rpcFailure(ctx, "header subscription failed")
	}
	return s, nil
}
