package decimal

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
)

// This is an isolated Ethereum dev chain, not Decimal consensus compatibility.
// No discovery, external dialer or public RPC URL is enabled. Tests use only
// in-process RPC; any backend-required authenticated service is restricted to
// ephemeral loopback. Ordinary HTTP/WS/IPC listeners are disabled.
func newLocalSignedChain(t *testing.T, alloc types.GenesisAlloc) *simulated.Backend {
	t.Helper()
	chain := simulated.NewBackend(alloc, func(n *node.Config, _ *ethconfig.Config) {
		n.IPCPath = ""
		n.HTTPHost = ""
		n.WSHost = ""
		n.AuthAddr = "127.0.0.1"
		n.AuthPort = 0
		n.P2P.ListenAddr = ""
		n.P2P.NoDiscovery = true
		n.P2P.NoDial = true
		n.P2P.MaxPeers = 0
	})
	t.Cleanup(func() {
		if err := chain.Close(); err != nil {
			t.Error(err)
		}
	})
	return chain
}
func TestLocalChainSignedTransactionEnvelopesAndFees(t *testing.T) {
	for _, kind := range []uint8{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType} {
		t.Run(fmt.Sprintf("type-%d", kind), func(t *testing.T) {
			signer := testSigner(t, 1)
			defer signer.Close()
			initial := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
			chain := newLocalSignedChain(t, types.GenesisAlloc{signer.Address(): {Balance: initial}})
			client, err := NewClient(Config{ChainID: big.NewInt(1337)}, chain.Client())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			recipient := common.HexToAddress("0x2002")
			value := big.NewInt(1234567)
			call, err := NativeTransfer(recipient, value, "local envelope")
			if err != nil {
				t.Fatal(err)
			}
			opts := TxOptions{TxType: kind, GasPrice: big.NewInt(2000000000)}
			if kind == types.AccessListTxType {
				opts.AccessList = types.AccessList{{Address: recipient, StorageKeys: []common.Hash{common.HexToHash("0x1")}}}
			}
			if kind == types.DynamicFeeTxType {
				header, err := chain.Client().HeaderByNumber(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				opts.GasPrice = nil
				opts.FeeCap = big.NewInt(3000000000)
				opts.TipCap = big.NewInt(1000000000)
				opts.BaseFeeEstimate = new(big.Int).Set(header.BaseFee)
			}
			sent, err := client.Send(ctx, signer, call, opts)
			if err != nil {
				t.Fatal(err)
			}
			blockHash := chain.Commit()
			mined, err := client.WaitReceipt(ctx, sent.Hash, time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			if mined.Receipt.Type != kind || mined.Receipt.BlockHash != blockHash || mined.Receipt.BlockNumber.Sign() <= 0 || mined.Receipt.Status != 1 {
				t.Fatal("mined envelope identity", mined.Receipt)
			}
			tx, pending, err := chain.Client().TransactionByHash(ctx, sent.Hash)
			if err != nil || pending {
				t.Fatal("transaction not committed", pending, err)
			}
			sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(1337)), tx)
			if err != nil || sender != signer.Address() || tx.Type() != kind || tx.ChainId().Cmp(big.NewInt(1337)) != 0 || tx.Nonce() != 0 || tx.To() == nil || *tx.To() != recipient || string(tx.Data()) != "local envelope" || tx.Value().Cmp(value) != 0 {
				t.Fatal("signed intent changed", tx, err)
			}
			if !sameAccessList(tx.AccessList(), opts.AccessList) {
				t.Fatal("access list lost")
			}
			actual := new(big.Int).Mul(new(big.Int).SetUint64(mined.Receipt.GasUsed), mined.Receipt.EffectiveGasPrice)
			if mined.ActualFee == nil || mined.ActualFee.Cmp(actual) != 0 || actual.Cmp(sent.Fee.ReservedFee) > 0 {
				t.Fatal("fee accounting", mined.ActualFee, actual, sent.Fee)
			}
			balance, err := client.Balance(ctx, recipient)
			if err != nil || balance.Cmp(value) != 0 {
				t.Fatal("recipient balance", balance, err)
			}
			wantSender := new(big.Int).Sub(new(big.Int).Sub(initial, value), actual)
			balance, err = client.Balance(ctx, signer.Address())
			if err != nil || balance.Cmp(wantSender) != 0 {
				t.Fatal("sender balance/fee", balance, wantSender, err)
			}
			nonce, err := chain.Client().PendingNonceAt(ctx, signer.Address())
			if err != nil || nonce != 1 {
				t.Fatal("mined nonce", nonce, err)
			}
		})
	}
}
func TestLocalChainRevertedReceiptConsumesNonceAndFee(t *testing.T) {
	signer := testSigner(t, 1)
	defer signer.Close()
	target := common.HexToAddress("0x3003")
	initial := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
	// A minimal REVERT program is a protocol failure fixture, not a Decimal
	// contract substitute. It cannot accept value or emit logs successfully.
	chain := newLocalSignedChain(t, types.GenesisAlloc{signer.Address(): {Balance: initial}, target: {Code: common.FromHex("0x60006000fd")}})
	client, err := NewClient(Config{ChainID: big.NewInt(1337)}, chain.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	call, err := NativeTransfer(target, big.NewInt(99), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Prepare(ctx, signer.Address(), call, TxOptions{}); err == nil {
		t.Fatal("failed EVM estimate was accepted")
	}
	nonce, err := chain.Client().PendingNonceAt(ctx, signer.Address())
	if err != nil || nonce != 0 {
		t.Fatal("failed preparation consumed nonce", nonce, err)
	}
	tx, err := signer.SignTransaction(ctx, types.NewTx(&types.LegacyTx{Nonce: 0, To: &target, Value: big.NewInt(99), Gas: 50000, GasPrice: big.NewInt(2000000000)}), big.NewInt(1337))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := client.Broadcast(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	blockHash := chain.Commit()
	mined, err := client.WaitReceipt(ctx, hash, time.Millisecond)
	var reverted *RevertedError
	if !errors.As(err, &reverted) || reverted.Hash != hash || mined == nil || mined.Receipt.Status != 0 || mined.Receipt.BlockHash != blockHash || len(mined.Receipt.Logs) != 0 || mined.ActualFee == nil || mined.ActualFee.Sign() <= 0 {
		t.Fatal("reverted mined result", mined, err)
	}
	balance, err := client.Balance(ctx, signer.Address())
	want := new(big.Int).Sub(initial, mined.ActualFee)
	if err != nil || balance.Cmp(want) != 0 {
		t.Fatal("revert fee", balance, want, err)
	}
	balance, err = client.Balance(ctx, target)
	if err != nil || balance.Sign() != 0 {
		t.Fatal("reverted value transferred", balance, err)
	}
	nonce, err = chain.Client().PendingNonceAt(ctx, signer.Address())
	if err != nil || nonce != 1 {
		t.Fatal("reverted nonce not consumed", nonce, err)
	}
}
