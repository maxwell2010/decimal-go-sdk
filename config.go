package decimal

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"net/url"
)

// Config is copied by NewClient. ChainID is verified against the RPC before use.
type Config struct {
	ChainID                                                         *big.Int
	RPCURL, APIURL                                                  string
	ContractCenter, Multicall, Checks, Safe, SafeFactory, MultiSend common.Address
}

// Mainnet uses the public Decimal EVM network (75); the pinned node source has a different 2020 constant.
func Mainnet() Config {
	return Config{
		ChainID: big.NewInt(75), RPCURL: "https://node.decimalchain.com/web3/", APIURL: "https://api.decimalchain.com/api/v1/",
		ContractCenter: common.HexToAddress("0xc108715a06f76caa96fa2c943ebf05159c29a87d"),
		Multicall:      common.HexToAddress("0x7b23eb47587ca6482fc16cb3d9d426ec64d4b5fc"),
		Checks:         common.HexToAddress("0x9c3326594b49AFa24db9b988Cda1258d0dcb4e40"),
		Safe:           common.HexToAddress("0x15949c33775154549D073168C1094C5f3b28b5CB"),
		SafeFactory:    common.HexToAddress("0x92466f09D5c82e8DdB8AaA7c5AdC63d43111F6c1"),
		MultiSend:      common.HexToAddress("0x72b80471AAFabd1469ed1C51453DC9ca66068bC0"),
	}
}
func Testnet() Config {
	return Config{
		ChainID: big.NewInt(202020), RPCURL: "https://testnet-val.decimalchain.com/web3/", APIURL: "https://testnet-api.decimalchain.com/api/v1/",
		ContractCenter: common.HexToAddress("0xbC96b61F137F28F0Da47Cc4Ef06e5f984B565A2E"),
		Multicall:      common.HexToAddress("0x81FD5FAe106dFD0343B0435a1bc0ef89BB14C317"),
		Checks:         common.HexToAddress("0xb6161CAA8Bd7167C2fa67b93b041FF9a94d6a735"),
		Safe:           common.HexToAddress("0xE0f30FcCAe2f9f9C7efee9af044C436841D466ee"),
		SafeFactory:    common.HexToAddress("0x4cC406B1713d3dF67e739D6f5918B6C98F614a38"),
		MultiSend:      common.HexToAddress("0xC0611a00CE349B9bCff3866351A2a5Aa9659c464"),
	}
}

func (c Config) validate() error {
	if !uint256(c.ChainID) || c.ChainID.Sign() == 0 {
		return errors.New("positive chain ID required")
	}
	return nil
}
func ParseAddress(s string) (common.Address, error) {
	if !common.IsHexAddress(s) {
		return common.Address{}, errors.New("invalid EVM address")
	}
	return common.HexToAddress(s), nil
}
func validURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Host != "" && u.User == nil && (u.Scheme == "http" || u.Scheme == "https")
}
