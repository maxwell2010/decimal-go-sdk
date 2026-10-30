package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// UnifiedNFTDelegation explicitly selects the recovery-era unified NFT route.
// Construction performs no RPC, approval, signing or transaction submission.
// Historical NFT positions remain accessible through DelegationNFT.
type UnifiedNFTDelegation struct{ *Contract }

func (c *Client) UnifiedNFTDelegation(address common.Address) (*UnifiedNFTDelegation, error) {
	if address == (common.Address{}) {
		return nil, errors.New("unified delegation address required")
	}
	a, e := delegationUpgradeABI()
	if e != nil {
		return nil, e
	}
	return &UnifiedNFTDelegation{&Contract{client: c, Address: address, Profile: "UnifiedNFTDelegation20261030", override: &a}}, nil
}

// ApprovalOperator is the actual action destination, never a legacy NFT address.
func (d *UnifiedNFTDelegation) ApprovalOperator() common.Address {
	if d == nil || d.Contract == nil {
		return common.Address{}
	}
	return d.Address
}

func (d *UnifiedNFTDelegation) validateTarget() error {
	if d == nil || d.Contract == nil || d.Address == (common.Address{}) {
		return errors.New("unified delegation target required")
	}
	return nil
}

// Build validates the selected destination before building an offline intent.
func (d *UnifiedNFTDelegation) Build(signature string, value *big.Int, args ...any) (Call, error) {
	if err := d.validateTarget(); err != nil {
		return Call{}, err
	}
	return d.Contract.Build(signature, value, args...)
}

// NFTStake is the separate official ten-field NFT tuple, not a Stake7/8.
type NFTStake struct {
	NftContract   common.Address `abi:"nftContract"`
	TokenId       *big.Int       `abi:"tokenId"`
	Amount        *big.Int       `abi:"amount"`
	NftType       uint8          `abi:"nftType"`
	Delegator     common.Address `abi:"delegator"`
	Validator     common.Address `abi:"validator"`
	ReserveToken  common.Address `abi:"reserveToken"`
	ReserveAmount *big.Int       `abi:"reserveAmount"`
	HoldTimestamp *big.Int       `abi:"holdTimestamp"`
	IsActive      bool           `abi:"isActive"`
}

func (d *UnifiedNFTDelegation) ReadNFTStakeAt(ctx context.Context, block *big.Int, nft common.Address, tokenID *big.Int, delegator, validator common.Address, holdTimestamp *big.Int) (NFTStake, error) {
	var result NFTStake
	if err := d.validateTarget(); err != nil {
		return result, err
	}
	if block == nil || block.Sign() < 0 {
		return result, errors.New("explicit snapshot block required")
	}
	values, e := d.ReadAt(ctx, block, "getNFTStake(address,uint256,address,address,uint256)", nft, tokenID, delegator, validator, holdTimestamp)
	if e != nil {
		return result, e
	}
	return *abi.ConvertType(values[0], new(NFTStake)).(*NFTStake), nil
}

func (d *UnifiedNFTDelegation) ReadNFTStakeIDAt(ctx context.Context, block *big.Int, nft common.Address, tokenID *big.Int, delegator, validator common.Address, holdTimestamp *big.Int) ([32]byte, error) {
	if err := d.validateTarget(); err != nil {
		return [32]byte{}, err
	}
	if block == nil || block.Sign() < 0 {
		return [32]byte{}, errors.New("explicit snapshot block required")
	}
	values, e := d.ReadAt(ctx, block, "getNFTStakeId(address,uint256,address,address,uint256)", nft, tokenID, delegator, validator, holdTimestamp)
	if e != nil {
		return [32]byte{}, e
	}
	return *abi.ConvertType(values[0], new([32]byte)).(*[32]byte), nil
}

// BuildDelegateNFT builds delegateNFT(address,address,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildDelegateNFT(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (Call, error) {
	return d.Build("delegateNFT(address,address,uint256)", nil, arg0, arg1, arg2)
}

// BuildDelegateNFT1155 builds delegateNFT1155(address,address,uint256,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildDelegateNFT1155(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int) (Call, error) {
	return d.Build("delegateNFT1155(address,address,uint256,uint256)", nil, arg0, arg1, arg2, arg3)
}

// BuildDelegateNFTHold builds delegateNFTHold(address,address,uint256,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildDelegateNFTHold(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int) (Call, error) {
	return d.Build("delegateNFTHold(address,address,uint256,uint256)", nil, arg0, arg1, arg2, arg3)
}

// BuildDelegateNFT1155Hold builds delegateNFT1155Hold(address,address,uint256,uint256,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildDelegateNFT1155Hold(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 *big.Int) (Call, error) {
	return d.Build("delegateNFT1155Hold(address,address,uint256,uint256,uint256)", nil, arg0, arg1, arg2, arg3, arg4)
}

// BuildDelegateNFTByPermit builds delegateNFTByPermit(address,address,uint256,uint256,uint8,bytes32,bytes32) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildDelegateNFTByPermit(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 uint8, arg5 [32]byte, arg6 [32]byte) (Call, error) {
	return d.Build("delegateNFTByPermit(address,address,uint256,uint256,uint8,bytes32,bytes32)", nil, arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// BuildWithdrawNFT builds withdrawNFT(address,address,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildWithdrawNFT(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (Call, error) {
	return d.Build("withdrawNFT(address,address,uint256)", nil, arg0, arg1, arg2)
}

// BuildWithdrawNFT1155 builds withdrawNFT1155(address,address,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildWithdrawNFT1155(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (Call, error) {
	return d.Build("withdrawNFT1155(address,address,uint256)", nil, arg0, arg1, arg2)
}

// BuildWithdrawNFTHold builds withdrawNFTHold(address,address,uint256,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildWithdrawNFTHold(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int) (Call, error) {
	return d.Build("withdrawNFTHold(address,address,uint256,uint256)", nil, arg0, arg1, arg2, arg3)
}

// BuildWithdrawNFT1155Hold builds withdrawNFT1155Hold(address,address,uint256,uint256) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildWithdrawNFT1155Hold(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int) (Call, error) {
	return d.Build("withdrawNFT1155Hold(address,address,uint256,uint256)", nil, arg0, arg1, arg2, arg3)
}

// BuildTransferNFT builds transferNFT(address,address,uint256,address) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildTransferNFT(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 common.Address) (Call, error) {
	return d.Build("transferNFT(address,address,uint256,address)", nil, arg0, arg1, arg2, arg3)
}

// BuildTransferNFT1155 builds transferNFT1155(address,address,uint256,address) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildTransferNFT1155(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 common.Address) (Call, error) {
	return d.Build("transferNFT1155(address,address,uint256,address)", nil, arg0, arg1, arg2, arg3)
}

// BuildTransferNFTHold builds transferNFTHold(address,address,uint256,uint256,address) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildTransferNFTHold(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (Call, error) {
	return d.Build("transferNFTHold(address,address,uint256,uint256,address)", nil, arg0, arg1, arg2, arg3, arg4)
}

// BuildTransferNFT1155Hold builds transferNFT1155Hold(address,address,uint256,uint256,address) offline; it does not approve, sign or send.
func (d *UnifiedNFTDelegation) BuildTransferNFT1155Hold(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 common.Address) (Call, error) {
	return d.Build("transferNFT1155Hold(address,address,uint256,uint256,address)", nil, arg0, arg1, arg2, arg3, arg4)
}

// BuildERC721Approval builds a token-specific approval for this unified operator.
func (d *UnifiedNFTDelegation) BuildERC721Approval(collection common.Address, tokenID *big.Int) (Call, error) {
	if err := d.validateTarget(); err != nil {
		return Call{}, err
	}
	token, e := d.client.ERC721(collection)
	if e != nil {
		return Call{}, e
	}
	return token.BuildApprove(d.ApprovalOperator(), tokenID)
}

// BuildERC1155ApprovalForAll builds a collection-wide grant to this unified operator.
// Review this broader approval separately from the action. Nothing is submitted.
func (d *UnifiedNFTDelegation) BuildERC1155ApprovalForAll(collection common.Address) (Call, error) {
	if err := d.validateTarget(); err != nil {
		return Call{}, err
	}
	token, e := d.client.ERC1155(collection)
	if e != nil {
		return Call{}, e
	}
	return token.BuildSetApprovalForAll(d.ApprovalOperator(), true)
}
