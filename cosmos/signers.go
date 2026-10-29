package cosmos

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"reflect"
)

var ErrUnsupportedSigners = errors.New("signer extraction unsupported for this message type")

// GetSigners reproduces valid-message signer extraction from the pinned node
// GetSigners methods. It does not validate message economics, validator roles,
// oracle membership, module authority or active route admission. Invalid
// addresses return an error rather than the upstream nil/panic variations.
func GetSigners(message proto.Message) ([][]byte, error) {
	if message == nil || (reflect.ValueOf(message).Kind() == reflect.Pointer && reflect.ValueOf(message).IsNil()) || !message.ProtoReflect().IsValid() {
		return nil, errors.New("message required")
	}
	reflection := message.ProtoReflect()
	name := "/" + string(reflection.Descriptor().FullName())
	rule, ok := signerRules[name]
	if !ok {
		return nil, ErrUnsupportedSigners
	}
	pinned, e := NewMessage(name)
	if e != nil || reflection.Descriptor() != pinned.Descriptor() {
		return nil, errors.New("signer message descriptor differs from pinned schema")
	}
	field := reflection.Descriptor().Fields().ByName(protoreflect.Name(rule.field))
	if field == nil || field.Kind() != protoreflect.StringKind || field.IsList() || field.IsMap() {
		return nil, errors.New("signer field does not match pinned schema")
	}
	raw, e := DecodeAddress(reflection.Get(field).String(), rule.hrp)
	if e != nil {
		return nil, e
	}
	return [][]byte{raw}, nil
}

// TransactionSigners preserves first message occurrence and appends an explicit
// fee payer if not already present, following Cosmos SDK0.46.16 types/tx/Tx.
// This function returns address bytes, not AuthInfo public keys. The application
// must bind each required address to its authorized key before signing.
func TransactionSigners(messages []proto.Message, feePayer string) ([][]byte, error) {
	if len(messages) == 0 {
		return nil, errors.New("messages required")
	}
	seen := map[string]bool{}
	result := [][]byte{}
	for _, message := range messages {
		signers, e := GetSigners(message)
		if e != nil {
			return nil, e
		}
		for _, address := range signers {
			key := string(address)
			if !seen[key] {
				seen[key] = true
				result = append(result, append([]byte(nil), address...))
			}
		}
	}
	if feePayer != "" {
		address, e := DecodeAccountAddress(feePayer)
		if e != nil {
			return nil, e
		}
		if !seen[string(address)] {
			result = append(result, address)
		}
	}
	return result, nil
}

type signerRule struct{ field, hrp string }

var signerRules = map[string]signerRule{
	// Source-backed rules are listed individually; no generic sender fallback.
	"/decimal.coin.v1.MsgCreateCoin":                 {"sender", "d0"},
	"/decimal.coin.v1.MsgUpdateCoin":                 {"sender", "d0"},
	"/decimal.coin.v1.MsgSendCoin":                   {"sender", "d0"},
	"/decimal.coin.v1.MsgMultiSendCoin":              {"sender", "d0"},
	"/decimal.coin.v1.MsgBuyCoin":                    {"sender", "d0"},
	"/decimal.coin.v1.MsgSellCoin":                   {"sender", "d0"},
	"/decimal.coin.v1.MsgSellAllCoin":                {"sender", "d0"},
	"/decimal.coin.v1.MsgBurnCoin":                   {"sender", "d0"},
	"/decimal.coin.v1.MsgRedeemCheck":                {"sender", "d0"},
	"/decimal.legacy.v1.MsgReturnLegacy":             {"sender", "d0"},
	"/decimal.multisig.v1.MsgCreateWallet":           {"sender", "d0"},
	"/decimal.multisig.v1.MsgCreateTransaction":      {"sender", "d0"},
	"/decimal.multisig.v1.MsgSignTransaction":        {"sender", "d0"},
	"/decimal.nft.v1.MsgMintToken":                   {"sender", "d0"},
	"/decimal.nft.v1.MsgUpdateToken":                 {"sender", "d0"},
	"/decimal.nft.v1.MsgUpdateReserve":               {"sender", "d0"},
	"/decimal.nft.v1.MsgSendToken":                   {"sender", "d0"},
	"/decimal.nft.v1.MsgBurnToken":                   {"sender", "d0"},
	"/decimal.swap.v1.MsgInitializeSwap":             {"sender", "d0"},
	"/decimal.swap.v1.MsgRedeemSwap":                 {"sender", "d0"},
	"/decimal.swap.v1.MsgActivateChain":              {"sender", "d0"},
	"/decimal.swap.v1.MsgDeactivateChain":            {"sender", "d0"},
	"/decimal.validator.v1.MsgCreateValidator":       {"operator_address", "d0valoper"},
	"/decimal.validator.v1.MsgEditValidator":         {"operator_address", "d0valoper"},
	"/decimal.validator.v1.MsgSetOnline":             {"validator", "d0valoper"},
	"/decimal.validator.v1.MsgSetOffline":            {"validator", "d0valoper"},
	"/decimal.validator.v1.MsgDelegate":              {"delegator", "d0"},
	"/decimal.validator.v1.MsgDelegateNFT":           {"delegator", "d0"},
	"/decimal.validator.v1.MsgRedelegate":            {"delegator", "d0"},
	"/decimal.validator.v1.MsgRedelegateNFT":         {"delegator", "d0"},
	"/decimal.validator.v1.MsgUndelegate":            {"delegator", "d0"},
	"/decimal.validator.v1.MsgUndelegateNFT":         {"delegator", "d0"},
	"/decimal.validator.v1.MsgCancelRedelegation":    {"delegator", "d0"},
	"/decimal.validator.v1.MsgCancelRedelegationNFT": {"delegator", "d0"},
	"/decimal.validator.v1.MsgCancelUndelegation":    {"delegator", "d0"},
	"/decimal.validator.v1.MsgCancelUndelegationNFT": {"delegator", "d0"},
	"/decimal.validator.v1.MsgHaltChain":             {"sender", "d0"},
	"/decimal.validator.v1.MsgResumeChain":           {"sender", "d0"},
	"/decimal.validator.v1.MsgFreezeChain":           {"sender", "d0"},
	"/decimal.validator.v1.MsgUnfreezeChain":         {"sender", "d0"},
	"/decimal.fee.v1.MsgUpdateCoinPrices":            {"oracle", "d0"},
	"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade":     {"authority", "d0"},
	"/cosmos.upgrade.v1beta1.MsgCancelUpgrade":       {"authority", "d0"},
}
