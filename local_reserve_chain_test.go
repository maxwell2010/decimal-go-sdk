package decimal

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Actual signed/mined economic flows on an isolated Ethereum dev chain. Code
// comes from the pinned reserve fixture, but public proxy/Decimal consensus and
// network fees are not reproduced. The explicit ABI is used on chain 1337.
func TestLocalReserveSignedChainEconomyAndActualFees(t *testing.T) {
	fixture, source, _ := newLocalReserveEconomy(t, 100)
	code := append([]byte(nil), fixture.state.GetCode(source.Address)...)
	signer := testSigner(t, 1)
	defer signer.Close()
	target := common.HexToAddress("0x4004")
	initialNative := reserveUnits(10000)
	chain := newLocalSignedChain(t, types.GenesisAlloc{signer.Address(): {Balance: new(big.Int).Set(initialNative)}, target: {Code: code}})
	client, err := NewClient(Config{ChainID: big.NewInt(1337)}, chain.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.TestnetReserveToken20261029(target); err == nil {
		t.Fatal("testnet profile silently adopted on unrelated chain")
	}
	raw, err := ABIJSON("TestnetReserveToken20261029")
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := client.ContractWithABI(target, raw)
	if err != nil {
		t.Fatal(err)
	}
	token := &TestnetReserveToken20261029{Contract: explicit}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	fees := new(big.Int)
	netCost := new(big.Int)
	send := func(call Call, buildErr error, wantNativeCost *big.Int) {
		t.Helper()
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		submitted, err := client.Send(ctx, signer, call, TxOptions{GasPrice: big.NewInt(2000000000)})
		if err != nil {
			t.Fatal(err)
		}
		blockHash := chain.Commit()
		mined, err := client.WaitReceipt(ctx, submitted.Hash, time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		if mined.Receipt.BlockHash != blockHash || mined.ActualFee == nil || mined.ActualFee.Sign() <= 0 || mined.ActualFee.Cmp(submitted.Fee.ReservedFee) > 0 {
			t.Fatal("mined fee/identity", mined)
		}
		fees.Add(fees, mined.ActualFee)
		netCost.Add(netCost, wantNativeCost)
		balance, err := client.Balance(ctx, signer.Address())
		want := new(big.Int).Sub(new(big.Int).Sub(initialNative, netCost), fees)
		if err != nil || balance.Cmp(want) != 0 {
			t.Fatal("native settlement/refund/fee delta", balance, want, err)
		}
	}
	check := func(wantSupply, wantReserve *big.Int) {
		t.Helper()
		supply, err := token.ReadTotalSupply(ctx)
		if err != nil || supply.Value0.Cmp(wantSupply) != 0 {
			t.Fatal("supply", supply, err)
		}
		reserve, err := token.ReadReserve(ctx)
		if err != nil || reserve.Value0.Cmp(wantReserve) != 0 {
			t.Fatal("reserve getter", reserve, err)
		}
		native, err := client.Balance(ctx, target)
		if err != nil || native.Cmp(wantReserve) != 0 {
			t.Fatal("settled reserve balance", native, err)
		}
		tokens, err := token.ReadBalanceOf(ctx, signer.Address())
		if err != nil || tokens.Value0.Cmp(wantSupply) != 0 {
			t.Fatal("token balance", tokens, err)
		}
	}
	init, err := token.BuildInitialize(reserveUnits(100), "SignedReserve", "SRS", signer.Address(), 100, reserveUnits(100), reserveUnits(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), "official local code")
	send(init, err, reserveUnits(100))
	check(reserveUnits(100), reserveUnits(100))
	// Attach 15 native units but buy exactly 10 tokens. The actual mined sender
	// delta must account for the 5-unit refund as well as the charged gas fee.
	buy, err := token.BuildBuyExactTokenForDEL(reserveUnits(15), reserveUnits(10), signer.Address())
	send(buy, err, reserveUnits(10))
	check(reserveUnits(110), reserveUnits(110))
	sell, err := token.BuildSellExactTokensForDEL(reserveUnits(10), reserveUnits(10), signer.Address())
	send(sell, err, new(big.Int).Neg(reserveUnits(10)))
	check(reserveUnits(100), reserveUnits(100))
	burn, err := token.BuildBurn(reserveUnits(5))
	send(burn, err, new(big.Int))
	check(reserveUnits(95), reserveUnits(100))
	before, err := client.Balance(ctx, signer.Address())
	if err != nil {
		t.Fatal(err)
	}
	bad, err := token.BuildBuy(reserveUnits(1), reserveUnits(1000), signer.Address())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Prepare(ctx, signer.Address(), bad, TxOptions{GasPrice: big.NewInt(2000000000)}); err == nil {
		t.Fatal("impossible slippage estimate succeeded")
	}
	after, err := client.Balance(ctx, signer.Address())
	if err != nil || after.Cmp(before) != 0 {
		t.Fatal("failed preparation charged value or fee", after, before, err)
	}
	nonce, err := chain.Client().PendingNonceAt(ctx, signer.Address())
	if err != nil || nonce != 4 {
		t.Fatal("unexpected committed nonce", nonce, err)
	}
	check(reserveUnits(95), reserveUnits(100))
}
