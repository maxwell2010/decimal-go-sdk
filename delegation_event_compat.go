package decimal

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

// Receipt logs retain their original topic and tuple layout. Node hook
// normalization does not change the receipt returned by eth_getTransactionReceipt.
func decodeDelegationUpgradeLog(legacy abi.ABI, log types.Log, allowUpgrade bool) (Event, error) {
	ev, err := legacy.EventByID(log.Topics[0])
	if err != nil {
		if !allowUpgrade {
			return Event{}, err
		}
		current, loadErr := delegationUpgradeABI()
		if loadErr != nil {
			return Event{}, loadErr
		}
		ev, err = current.EventByID(log.Topics[0])
		if err != nil {
			return Event{}, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range ev.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if len(log.Topics) != len(indexed)+1 {
		return Event{}, errors.New("invalid delegation event topic count")
	}
	plain := ev.Inputs.NonIndexed()
	decoded, err := plain.Unpack(log.Data)
	if err != nil {
		return Event{}, err
	}
	encoded, err := plain.Pack(decoded...)
	if err != nil || !bytes.Equal(encoded, log.Data) {
		return Event{}, errors.New("noncanonical delegation event data")
	}
	for i, arg := range indexed {
		switch arg.Type.T {
		case abi.StringTy, abi.BytesTy, abi.SliceTy, abi.ArrayTy, abi.TupleTy:
			// Indexed dynamic/aggregate values are hashes, not decodable values.
			continue
		}
		args := abi.Arguments{{Type: arg.Type}}
		value, err := args.Unpack(log.Topics[i+1][:])
		if err != nil {
			return Event{}, err
		}
		encoded, err := args.Pack(value...)
		if err != nil || !bytes.Equal(encoded, log.Topics[i+1][:]) {
			return Event{}, errors.New("noncanonical delegation indexed value")
		}
	}
	values := make(map[string]any)
	for i, arg := range plain {
		values[arg.Name] = decoded[i]
	}
	if err := abi.ParseTopicsIntoMap(values, indexed, log.Topics[1:]); err != nil {
		return Event{}, err
	}
	return Event{Name: ev.Name, Signature: ev.Sig, Values: values, Log: log}, nil
}
