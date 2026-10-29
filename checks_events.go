package decimal

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"sort"
)

// ChecksCreation is historical event evidence, not current spendable check state.
type ChecksCreation struct {
	CheckHash   common.Hash
	DetailsHash common.Hash
	Signer      common.Address
	Status      uint8
	Log         types.Log
}

// ChecksCreationDetails preserves the explicit CreatedCheckDetails payload.
// It does not infer how DetailsHash was originally computed.
type ChecksCreationDetails struct {
	DetailsHash      common.Hash
	Type             uint8
	Amount, DueBlock *big.Int
	Token, Creator   common.Address
	Log              types.Log
}

func checksEventLog(address common.Address, log types.Log, name string, size int) (types.Log, error) {
	a, e := loadABI("Checks")
	if e != nil {
		return types.Log{}, e
	}
	if address == (common.Address{}) || log.Address != address || log.Removed || len(log.Topics) != 1 || log.Topics[0] != a.Events[name].ID || len(log.Data) != size {
		return types.Log{}, errors.New("invalid Checks creation event address, topic, removal flag or length")
	}
	if log.BlockNumber == 0 || log.BlockHash == (common.Hash{}) || log.TxHash == (common.Hash{}) {
		return types.Log{}, errors.New("mined event identity required")
	}
	values, e := a.Events[name].Inputs.Unpack(log.Data)
	if e != nil {
		return types.Log{}, e
	}
	canonical, e := a.Events[name].Inputs.Pack(values...)
	if e != nil || !bytes.Equal(canonical, log.Data) {
		return types.Log{}, errors.New("noncanonical Checks event data")
	}
	log.Data = append([]byte(nil), log.Data...)
	log.Topics = append([]common.Hash(nil), log.Topics...)
	return log, nil
}

// DecodeChecksCreationLog decodes the pinned JS CreatedCheck event exactly.
func DecodeChecksCreationLog(address common.Address, log types.Log) (ChecksCreation, error) {
	l, e := checksEventLog(address, log, "CreatedCheck", 128)
	if e != nil {
		return ChecksCreation{}, e
	}
	// ABI canonical validation above includes address and uint8 padding.
	return ChecksCreation{common.BytesToHash(l.Data[:32]), common.BytesToHash(l.Data[32:64]), common.BytesToAddress(l.Data[76:96]), l.Data[127], l}, nil
}

// DecodeChecksCreationDetailsLog decodes the corresponding detail event.
func DecodeChecksCreationDetailsLog(address common.Address, log types.Log) (ChecksCreationDetails, error) {
	l, e := checksEventLog(address, log, "CreatedCheckDetails", 192)
	if e != nil {
		return ChecksCreationDetails{}, e
	}
	return ChecksCreationDetails{common.BytesToHash(l.Data[:32]), l.Data[63], new(big.Int).SetBytes(l.Data[64:96]), new(big.Int).SetBytes(l.Data[96:128]), common.BytesToAddress(l.Data[140:160]), common.BytesToAddress(l.Data[172:192]), l}, nil
}

// ChecksCreationsFromReceipt extracts only matching CreatedCheck events from a
// successful mined receipt. It rejects inconsistent log/receipt identities.
// A receipt alone is not a confirmation-depth or current-state guarantee.
func ChecksCreationsFromReceipt(address common.Address, receipt *types.Receipt) ([]ChecksCreation, error) {
	if address == (common.Address{}) || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful || receipt.BlockNumber == nil || !receipt.BlockNumber.IsUint64() || receipt.BlockNumber.Sign() <= 0 || receipt.BlockHash == (common.Hash{}) || receipt.TxHash == (common.Hash{}) {
		return nil, errors.New("successful mined receipt required")
	}
	a, e := loadABI("Checks")
	if e != nil {
		return nil, e
	}
	var logs []types.Log
	for _, l := range receipt.Logs {
		if l == nil {
			return nil, errors.New("nil receipt log")
		}
		if l.Address != address || len(l.Topics) == 0 || l.Topics[0] != a.Events["CreatedCheck"].ID {
			continue
		}
		if l.BlockNumber != receipt.BlockNumber.Uint64() || l.BlockHash != receipt.BlockHash || l.TxHash != receipt.TxHash || l.TxIndex != receipt.TransactionIndex {
			return nil, errors.New("receipt event identity mismatch")
		}
		logs = append(logs, *l)
	}
	return checksCreationRows(address, logs)
}

func checksCreationRows(address common.Address, logs []types.Log) ([]ChecksCreation, error) {
	rows := make([]ChecksCreation, 0, len(logs))
	positions := map[[2]uint64]bool{}
	blocks := map[uint64]common.Hash{}
	type coordinate struct {
		Block uint64
		Hash  common.Hash
		Index uint
	}
	transactions := map[common.Hash]coordinate{}
	transactionPositions := map[[2]uint64]common.Hash{}
	for _, l := range logs {
		row, e := DecodeChecksCreationLog(address, l)
		if e != nil {
			return nil, e
		}
		position := [2]uint64{l.BlockNumber, uint64(l.Index)}
		if positions[position] {
			return nil, errors.New("duplicate Checks log position")
		}
		if h, ok := blocks[l.BlockNumber]; ok && h != l.BlockHash {
			return nil, errors.New("inconsistent event block hashes")
		}
		txPosition := [2]uint64{l.BlockNumber, uint64(l.TxIndex)}
		if tx, ok := transactionPositions[txPosition]; ok && tx != l.TxHash {
			return nil, errors.New("inconsistent transaction hash at block transaction index")
		}
		point := coordinate{l.BlockNumber, l.BlockHash, l.TxIndex}
		if previous, ok := transactions[l.TxHash]; ok && previous != point {
			return nil, errors.New("transaction hash has inconsistent mined coordinates")
		}
		transactionPositions[txPosition] = l.TxHash
		transactions[l.TxHash] = point
		blocks[l.BlockNumber] = l.BlockHash
		positions[position] = true
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i].Log, rows[j].Log
		if a.BlockNumber != b.BlockNumber {
			return a.BlockNumber < b.BlockNumber
		}
		if a.TxIndex != b.TxIndex {
			return a.TxIndex < b.TxIndex
		}
		return a.Index < b.Index
	})
	for i := 1; i < len(rows); i++ {
		a, b := rows[i-1].Log, rows[i].Log
		if a.BlockNumber == b.BlockNumber && a.Index >= b.Index {
			return nil, errors.New("global log order contradicts transaction indices")
		}
	}
	return rows, nil
}

// ChecksCreationHistory reads at most 2000 explicitly selected blocks and 10000
// matching logs. Empty output means no matching event in this window only.
func (c *Client) ChecksCreationHistory(ctx context.Context, address common.Address, from, to uint64) ([]ChecksCreation, error) {
	if c == nil || c.rpc == nil || ctx == nil {
		return nil, errors.New("client and context required")
	}
	if address == (common.Address{}) || from > to || to-from >= 2000 {
		return nil, errors.New("Checks history needs nonzero address and range of 1..2000 blocks")
	}
	a, e := loadABI("Checks")
	if e != nil {
		return nil, e
	}
	if e = c.checkChain(ctx); e != nil {
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	r, ok := c.rpc.(interface {
		FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
	})
	if !ok {
		return nil, errors.New("backend does not support log filtering")
	}
	logs, e := r.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: new(big.Int).SetUint64(from), ToBlock: new(big.Int).SetUint64(to), Addresses: []common.Address{address}, Topics: [][]common.Hash{{a.Events["CreatedCheck"].ID}}})
	if e != nil {
		return nil, rpcFailure(ctx, "log query failed")
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	if len(logs) > 10000 {
		return nil, errors.New("Checks history exceeds 10000 events; narrow the range")
	}
	for _, l := range logs {
		if l.BlockNumber < from || l.BlockNumber > to {
			return nil, errors.New("Checks event outside requested block range")
		}
	}
	rows, e := checksCreationRows(address, logs)
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	return rows, e
}

// ChecksCreationReceipt reads one existing receipt and validates the requested
// transaction hash before extracting historical creation events.
func (c *Client) ChecksCreationReceipt(ctx context.Context, address common.Address, tx common.Hash) ([]ChecksCreation, error) {
	if c == nil || c.rpc == nil || ctx == nil || address == (common.Address{}) || tx == (common.Hash{}) {
		return nil, errors.New("client, context, Checks address and transaction hash required")
	}
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	r, e := c.rpc.TransactionReceipt(ctx, tx)
	if e != nil {
		return nil, rpcFailure(ctx, "Checks receipt unavailable")
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	if r == nil || r.TxHash != tx {
		return nil, errors.New("Checks receipt transaction hash mismatch")
	}
	rows, e := ChecksCreationsFromReceipt(address, r)
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	return rows, e
}
