package decimal

import (
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
)

// DecodeDelegationReceipt filters by this contract address and preserves the
// source order, including multiple exit events sharing one stake ID. Unknown
// topics are skipped; malformed known events fail the entire decode.
// It neither changes the receipt nor infers missing metadata or settlement.
func (c *Contract) DecodeDelegationReceipt(receipt *types.Receipt) ([]Event, error) {
	if c == nil || receipt == nil {
		return nil, errors.New("contract and receipt required")
	}
	if c.Profile != "Delegation" && c.Profile != "DelegationNFT" && c.Profile != "UnifiedNFTDelegation20261030" {
		return nil, errors.New("delegation contract profile required")
	}
	a, err := c.contractABI()
	if err != nil {
		return nil, err
	}
	result := make([]Event, 0)
	for _, log := range receipt.Logs {
		if log == nil || log.Address != c.Address || len(log.Topics) == 0 {
			continue
		}
		_, known := a.EventByID(log.Topics[0])
		if known != nil && c.override == nil && c.Profile == "Delegation" {
			current, err := delegationUpgradeABI()
			if err != nil {
				return nil, err
			}
			_, known = current.EventByID(log.Topics[0])
		}
		if known != nil {
			continue
		}
		event, err := c.DecodeLog(*log)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, nil
}
