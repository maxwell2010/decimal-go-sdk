package decimal

import (
	_ "embed"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
)

//go:embed docs/operation-matrix.json
var referenceJSON []byte

type ReferenceOperation struct {
	Operation string   `json:"operation"`
	Methods   []string `json:"methods"`
	Profiles  []string `json:"profiles"`
	Status    string   `json:"status"`
}

func ReferenceOperations() []ReferenceOperation {
	var out []ReferenceOperation
	_ = json.Unmarshal(referenceJSON, &out)
	return out
}

func referenceProfileFamily(profile string) string {
	switch profile {
	case "TestnetContractCenter20261029":
		return "ContractCenter"
	case "TestnetTokenCenter20261029":
		return "TokenCenter"
	case "TestnetNFTCenter20261029":
		return "NFTCenter"
	case "TestnetMasterValidator20261029":
		return "MasterValidator"
	case "TestnetReserveToken20261029", "TestnetReservelessToken20261029":
		return "Token"
	}
	return profile
}

func referenceProfileAllows(spec ReferenceOperation, profile string) bool {
	if profile == "deployment" {
		return true
	}
	// Reserveless mint is a separate, reviewed interface; never authorize mint on
	// the reserve Token profile merely because both have ERC20 metadata.
	if spec.Operation == "mintTokenReserveless" {
		return profile == "TestnetReservelessToken20261029"
	}
	for _, family := range spec.Profiles {
		if family == referenceProfileFamily(profile) {
			return true
		}
	}
	return false
}

func referenceMethodNames(spec ReferenceOperation, profile string) []string {
	// JS wrapper name differs from the observed reserve implementation selector.
	// This adapter applies only to the explicitly selected, versioned interface.
	if profile == "TestnetReserveToken20261029" && spec.Operation == "updateTokenIdentity" {
		return []string{"updateIdentity"}
	}
	return spec.Methods
}

// BuildReferenceOperation adapts a named official JS workflow to an explicit
// contract ABI. Arguments are in ABI order and raw units, not JS wrapper order.
// It neither obtains approvals nor signs permits automatically. Non-pinned
// methods require a reviewed versioned profile or explicit ContractWithABI;
// missing methods fail before any RPC/signature.
func (c *Contract) BuildReferenceOperation(operation string, value *big.Int, args ...any) (Call, error) {
	if c == nil {
		return Call{}, errors.New("reference contract required")
	}
	for _, spec := range ReferenceOperations() {
		if spec.Operation != operation {
			continue
		}
		if operation == "multiSendToken" {
			return Call{}, errors.New("use BuildMultisendDEL or a sender-restricted ERC20 deployment")
		}
		if spec.Status == "native" {
			return Call{}, errors.New("use NativeTransfer for DEL transfer/burn")
		}
		if !referenceProfileAllows(spec, c.Profile) {
			return Call{}, errors.New("reference operation requires another contract profile")
		}
		var result Call
		matches := 0
		for _, name := range referenceMethodNames(spec, c.Profile) {
			call, e := c.BuildNamed(name, value, args...)
			if e == nil {
				result = call
				matches++
			}
		}
		if matches != 1 {
			return Call{}, errors.New("reference operation absent or ambiguous for this deployment ABI")
		}
		return result, nil
	}
	return Call{}, errors.New("unknown reference operation")
}

// BuildReferenceOperationSignature disambiguates an overload while preserving
// the reference operation's contract-family and method-name restrictions.
// Arguments remain ABI-order values; supplying a deployment ABI is explicit.
func (c *Contract) BuildReferenceOperationSignature(operation, signature string, value *big.Int, args ...any) (Call, error) {
	if c == nil {
		return Call{}, errors.New("reference contract required")
	}
	for _, spec := range ReferenceOperations() {
		if spec.Operation != operation {
			continue
		}
		if operation == "multiSendToken" || spec.Status == "native" {
			return Call{}, errors.New("use the explicit native/multisend workflow")
		}
		if !referenceProfileAllows(spec, c.Profile) {
			return Call{}, errors.New("reference operation requires another contract profile")
		}
		name, _, ok := strings.Cut(signature, "(")
		if !ok {
			return Call{}, errors.New("full Solidity signature required")
		}
		for _, method := range referenceMethodNames(spec, c.Profile) {
			if method == name {
				return c.Build(signature, value, args...)
			}
		}
		return Call{}, errors.New("signature is not part of this reference operation")
	}
	return Call{}, errors.New("unknown reference operation")
}
