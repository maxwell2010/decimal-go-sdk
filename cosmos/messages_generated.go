// Code generated from pinned official descriptors; DO NOT EDIT.
package cosmos

import "google.golang.org/protobuf/types/dynamicpb"

func NewMsgCreateCoin() *dynamicpb.Message    { return message("decimal.coin.v1.MsgCreateCoin") }
func NewMsgUpdateCoin() *dynamicpb.Message    { return message("decimal.coin.v1.MsgUpdateCoin") }
func NewMsgSendCoin() *dynamicpb.Message      { return message("decimal.coin.v1.MsgSendCoin") }
func NewMsgMultiSendCoin() *dynamicpb.Message { return message("decimal.coin.v1.MsgMultiSendCoin") }
func NewMsgBuyCoin() *dynamicpb.Message       { return message("decimal.coin.v1.MsgBuyCoin") }
func NewMsgSellCoin() *dynamicpb.Message      { return message("decimal.coin.v1.MsgSellCoin") }
func NewMsgSellAllCoin() *dynamicpb.Message   { return message("decimal.coin.v1.MsgSellAllCoin") }
func NewMsgBurnCoin() *dynamicpb.Message      { return message("decimal.coin.v1.MsgBurnCoin") }
func NewMsgRedeemCheck() *dynamicpb.Message   { return message("decimal.coin.v1.MsgRedeemCheck") }
func NewMsgUpdateCoinPrices() *dynamicpb.Message {
	return message("decimal.fee.v1.MsgUpdateCoinPrices")
}
func NewMsgReturnLegacy() *dynamicpb.Message { return message("decimal.legacy.v1.MsgReturnLegacy") }
func NewMsgCreateWallet() *dynamicpb.Message { return message("decimal.multisig.v1.MsgCreateWallet") }
func NewMsgCreateTransaction() *dynamicpb.Message {
	return message("decimal.multisig.v1.MsgCreateTransaction")
}
func NewMsgSignTransaction() *dynamicpb.Message {
	return message("decimal.multisig.v1.MsgSignTransaction")
}
func NewMsgMintToken() *dynamicpb.Message       { return message("decimal.nft.v1.MsgMintToken") }
func NewMsgUpdateToken() *dynamicpb.Message     { return message("decimal.nft.v1.MsgUpdateToken") }
func NewMsgUpdateReserve() *dynamicpb.Message   { return message("decimal.nft.v1.MsgUpdateReserve") }
func NewMsgSendToken() *dynamicpb.Message       { return message("decimal.nft.v1.MsgSendToken") }
func NewMsgBurnToken() *dynamicpb.Message       { return message("decimal.nft.v1.MsgBurnToken") }
func NewMsgActivateChain() *dynamicpb.Message   { return message("decimal.swap.v1.MsgActivateChain") }
func NewMsgDeactivateChain() *dynamicpb.Message { return message("decimal.swap.v1.MsgDeactivateChain") }
func NewMsgInitializeSwap() *dynamicpb.Message  { return message("decimal.swap.v1.MsgInitializeSwap") }
func NewMsgRedeemSwap() *dynamicpb.Message      { return message("decimal.swap.v1.MsgRedeemSwap") }
func NewMsgCreateValidator() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgCreateValidator")
}
func NewMsgEditValidator() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgEditValidator")
}
func NewMsgSetOnline() *dynamicpb.Message   { return message("decimal.validator.v1.MsgSetOnline") }
func NewMsgSetOffline() *dynamicpb.Message  { return message("decimal.validator.v1.MsgSetOffline") }
func NewMsgDelegate() *dynamicpb.Message    { return message("decimal.validator.v1.MsgDelegate") }
func NewMsgDelegateNFT() *dynamicpb.Message { return message("decimal.validator.v1.MsgDelegateNFT") }
func NewMsgRedelegate() *dynamicpb.Message  { return message("decimal.validator.v1.MsgRedelegate") }
func NewMsgRedelegateNFT() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgRedelegateNFT")
}
func NewMsgUndelegate() *dynamicpb.Message { return message("decimal.validator.v1.MsgUndelegate") }
func NewMsgUndelegateNFT() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgUndelegateNFT")
}
func NewMsgCancelRedelegation() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgCancelRedelegation")
}
func NewMsgCancelRedelegationNFT() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgCancelRedelegationNFT")
}
func NewMsgCancelUndelegation() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgCancelUndelegation")
}
func NewMsgCancelUndelegationNFT() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgCancelUndelegationNFT")
}
func NewMsgHaltChain() *dynamicpb.Message   { return message("decimal.validator.v1.MsgHaltChain") }
func NewMsgResumeChain() *dynamicpb.Message { return message("decimal.validator.v1.MsgResumeChain") }
func NewMsgFreezeChain() *dynamicpb.Message { return message("decimal.validator.v1.MsgFreezeChain") }
func NewMsgUnfreezeChain() *dynamicpb.Message {
	return message("decimal.validator.v1.MsgUnfreezeChain")
}
func NewMsgSoftwareUpgrade() *dynamicpb.Message {
	return message("cosmos.upgrade.v1beta1.MsgSoftwareUpgrade")
}
func NewMsgCancelUpgrade() *dynamicpb.Message {
	return message("cosmos.upgrade.v1beta1.MsgCancelUpgrade")
}
