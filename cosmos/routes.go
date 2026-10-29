package cosmos

// PinnedNativeRoute reports the native Cosmos ante admission found in node
// commit9e6c6d7. It is source evidence, not a live deployment/role guarantee.
func PinnedNativeRoute(typeURL string) bool {
	switch typeURL {
	case "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", "/cosmos.upgrade.v1beta1.MsgCancelUpgrade", "/decimal.validator.v1.MsgSetOnline", "/decimal.validator.v1.MsgHaltChain", "/decimal.validator.v1.MsgResumeChain", "/decimal.validator.v1.MsgFreezeChain", "/decimal.validator.v1.MsgUnfreezeChain", "/decimal.fee.v1.MsgUpdateCoinPrices":
		return true
	}
	return false
}
