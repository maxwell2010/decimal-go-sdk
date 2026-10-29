package decimal

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// BlockIdentity preserves the RPC-reported canonical block hash. Decimal's block
// hash need not equal the hash recomputed from an Ethereum types.Header.
type BlockIdentity struct {
	Number uint64      `json:"number"`
	Hash   common.Hash `json:"hash"`
}

// BlockIdentity reads an authoritative RPC block number/hash pair, not
// Header.Hash(). A nil number requests latest; explicit numbers must fit uint64.
// Custom Backends may implement this method themselves. No Ethereum-header hash
// fallback is used when a backend cannot supply the canonical identity.
func (c *Client) BlockIdentity(ctx context.Context, number *big.Int) (BlockIdentity, error) {
	if ctx == nil {
		return BlockIdentity{}, errors.New("context required")
	}
	if e := ctx.Err(); e != nil {
		return BlockIdentity{}, e
	}
	if c == nil || c.rpc == nil {
		return BlockIdentity{}, errors.New("RPC backend required")
	}
	// Capture a value before any backend method can mutate the caller's big.Int.
	explicit := number != nil
	var requested uint64
	tag := "latest"
	if explicit {
		snapshot := new(big.Int).Set(number)
		if snapshot.Sign() < 0 || !snapshot.IsUint64() {
			return BlockIdentity{}, errors.New("block number must be nonnegative uint64")
		}
		requested = snapshot.Uint64()
		tag = fmt.Sprintf("0x%x", requested)
	}
	if e := c.checkChain(ctx); e != nil {
		return BlockIdentity{}, e
	}
	if e := ctx.Err(); e != nil {
		return BlockIdentity{}, e
	}
	var result BlockIdentity
	if backend, ok := c.rpc.(interface {
		BlockIdentity(context.Context, *big.Int) (BlockIdentity, error)
	}); ok {
		var argument *big.Int
		if explicit {
			argument = new(big.Int).SetUint64(requested)
		}
		var e error
		result, e = backend.BlockIdentity(ctx, argument)
		if e != nil {
			return BlockIdentity{}, rpcFailure(ctx, "block identity unavailable")
		}
	} else if backend, ok := c.rpc.(interface{ Client() *rpc.Client }); ok {
		raw := backend.Client()
		if raw == nil {
			return BlockIdentity{}, errors.New("backend does not support canonical block identity")
		}
		var response *struct {
			Number *hexutil.Uint64 `json:"number"`
			Hash   *common.Hash    `json:"hash"`
		}
		if e := raw.CallContext(ctx, &response, "eth_getBlockByNumber", tag, false); e != nil {
			return BlockIdentity{}, rpcFailure(ctx, "block identity unavailable")
		}
		if response == nil || response.Number == nil || response.Hash == nil {
			return BlockIdentity{}, errors.New("block identity response missing")
		}
		result = BlockIdentity{Number: uint64(*response.Number), Hash: *response.Hash}
	} else {
		return BlockIdentity{}, errors.New("backend does not support canonical block identity")
	}
	if e := ctx.Err(); e != nil {
		return BlockIdentity{}, e
	}
	if result.Hash == (common.Hash{}) {
		return BlockIdentity{}, errors.New("block identity hash is zero")
	}
	if explicit && result.Number != requested {
		return BlockIdentity{}, errors.New("block identity height mismatch")
	}
	return result, nil
}
