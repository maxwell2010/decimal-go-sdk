package decimal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	evmuint256 "github.com/holiman/uint256"
)

// localEVM executes actual CREATE/CALL bytecode in memory. It is deliberately
// not a mined chain, Decimal consensus emulator, or public transaction backend.
type localEVM struct {
	state    *state.StateDB
	cfg      *runtime.Config
	sequence uint64
}

func newLocalEVM(t *testing.T) *localEVM {
	t.Helper()
	db, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	rules := *params.AllDevChainProtocolChanges
	rules.ChainID = big.NewInt(202020)
	return &localEVM{state: db, cfg: &runtime.Config{State: db, ChainConfig: &rules, BlockNumber: big.NewInt(1), Time: 1, GasLimit: 30000000}}
}
func (e *localEVM) fund(address common.Address, amount *big.Int) {
	e.state.AddBalance(address, evmuint256.MustFromBig(amount), tracing.BalanceChangeUnspecified)
}
func (e *localEVM) balance(address common.Address) *big.Int {
	return e.state.GetBalance(address).ToBig()
}
func (e *localEVM) block(height uint64) { e.cfg.BlockNumber = new(big.Int).SetUint64(height) }
func (e *localEVM) next(from common.Address, value *big.Int) common.Hash {
	e.sequence++
	hash := common.BigToHash(new(big.Int).SetUint64(e.sequence))
	e.state.SetTxContext(hash, int(e.sequence), 0)
	e.cfg.Origin = from
	e.cfg.Value = copyInt(value)
	return hash
}
func (e *localEVM) deploy(t *testing.T, from common.Address, rawABI, creation []byte, args ...any) common.Address {
	t.Helper()
	parsed, err := abi.JSON(bytes.NewReader(rawABI))
	if err != nil {
		t.Fatal(err)
	}
	constructor, err := parsed.Pack("", args...)
	if err != nil {
		t.Fatal(err)
	}
	e.next(from, big.NewInt(0))
	code, address, _, err := runtime.Create(append(append([]byte(nil), creation...), constructor...), e.cfg)
	if err != nil {
		t.Fatalf("local CREATE failed: %v", err)
	}
	if len(code) == 0 {
		t.Fatal("local CREATE installed no code")
	}
	e.state.Finalise(e.cfg.ChainConfig.Rules(e.cfg.BlockNumber, true, e.cfg.Time))
	return address
}
func (e *localEVM) execute(t *testing.T, from common.Address, call Call) ([]byte, []*types.Log, error) {
	t.Helper()
	hash := e.next(from, call.Value)
	data, _, err := runtime.Call(call.To, append([]byte(nil), call.Data...), e.cfg)
	logs := e.state.GetLogs(hash, 0, common.Hash{}, 0)
	e.state.Finalise(e.cfg.ChainConfig.Rules(e.cfg.BlockNumber, true, e.cfg.Time))
	return data, logs, err
}

// The SDK's read transport executes the same bytecode on a state copy, so view
// calls cannot mutate persistent fixture state or the next operation's caller.
type localEVMReadBackend struct {
	*fakeRPC
	evm *localEVM
}

func (b *localEVMReadBackend) CodeAt(ctx context.Context, address common.Address, _ *big.Int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), b.evm.state.GetCode(address)...), nil
}
func (b *localEVMReadBackend) CallContract(ctx context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg := *b.evm.cfg
	cfg.State = b.evm.state.Copy()
	cfg.Origin = msg.From
	cfg.Value = copyInt(msg.Value)
	data, _, err := runtime.Call(*msg.To, append([]byte(nil), msg.Data...), &cfg)
	return data, err
}
func (e *localEVM) client(t *testing.T) *Client {
	t.Helper()
	c, err := NewClient(Testnet(), &localEVMReadBackend{newFake(), e})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type localContractArtifact struct {
	ABI              json.RawMessage `json:"abi"`
	Bytecode         string          `json:"bytecode"`
	DeployedBytecode string          `json:"deployedBytecode"`
}

func loadLocalArtifact(t *testing.T, name string) localContractArtifact {
	t.Helper()
	data, err := os.ReadFile("testdata/evm/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var artifact localContractArtifact
	if err = json.Unmarshal(data, &artifact); err != nil {
		t.Fatal(err)
	}
	if len(artifact.ABI) == 0 || len(common.FromHex(artifact.Bytecode)) == 0 {
		t.Fatal("missing executable official artifact")
	}
	return artifact
}
func TestLocalEVMOfficialHistoricalTokenLifecycle(t *testing.T) {
	ctx := context.Background()
	e := newLocalEVM(t)
	artifact := loadLocalArtifact(t, "NewToken.compiled.json")
	owner := common.HexToAddress("0x1001")
	recipient := common.HexToAddress("0x1002")
	spender := common.HexToAddress("0x1003")
	address := e.deploy(t, owner, artifact.ABI, common.FromHex(artifact.Bytecode), "Local official token", "LOT", big.NewInt(100))
	if !bytes.Equal(e.state.GetCode(address), common.FromHex(artifact.DeployedBytecode)) {
		t.Fatal("deployed runtime differs from compiled source")
	}
	client := e.client(t)
	token, err := client.Token(address)
	if err != nil {
		t.Fatal(err)
	}
	historical, err := client.ContractWithABI(address, artifact.ABI)
	if err != nil {
		t.Fatal(err)
	}
	checkBalance := func(who common.Address, want *big.Int) {
		t.Helper()
		got, err := token.ReadBalanceOf(ctx, who)
		if err != nil || got.Value0.Cmp(want) != 0 {
			t.Fatalf("balance %s got %v error %v want %v", who, got, err, want)
		}
	}
	checkSupply := func(want *big.Int) {
		t.Helper()
		got, err := token.ReadTotalSupply(ctx)
		if err != nil || got.Value0.Cmp(want) != 0 {
			t.Fatalf("supply got %v error %v want %v", got, err, want)
		}
	}
	run := func(from common.Address, call Call, buildErr error, fail bool) {
		t.Helper()
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		_, logs, err := e.execute(t, from, call)
		if (err != nil) != fail {
			t.Fatalf("execution error %v expected failure %v", err, fail)
		}
		if fail && len(logs) != 0 {
			t.Fatal("revert retained logs")
		}
		if !fail {
			if len(logs) < 1 || len(logs) > 2 {
				t.Fatalf("expected genuine token events: %d", len(logs))
			}
			for _, log := range logs {
				event, decodeErr := historical.DecodeLog(*log)
				if decodeErr != nil {
					t.Fatal(decodeErr)
				}
				if event.Name != "Transfer" && event.Name != "Approval" {
					t.Fatal("unexpected token event", event.Name)
				}
			}
		}
	}
	initial := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	checkBalance(owner, initial)
	checkSupply(initial)
	call, err := token.BuildTransfer(recipient, big.NewInt(10))
	run(owner, call, err, false)
	checkBalance(recipient, big.NewInt(10))
	ownerBalance := new(big.Int).Sub(new(big.Int).Set(initial), big.NewInt(10))
	checkBalance(owner, ownerBalance)
	call, err = token.BuildApprove(spender, big.NewInt(7))
	run(owner, call, err, false)
	allowance, err := token.ReadAllowance(ctx, owner, spender)
	if err != nil || allowance.Value0.Cmp(big.NewInt(7)) != 0 {
		t.Fatal("approval not recorded", allowance, err)
	}
	call, err = token.BuildTransferFrom(owner, recipient, big.NewInt(5))
	run(spender, call, err, false)
	checkBalance(recipient, big.NewInt(15))
	ownerBalance.Sub(ownerBalance, big.NewInt(5))
	checkBalance(owner, ownerBalance)
	allowance, err = token.ReadAllowance(ctx, owner, spender)
	if err != nil || allowance.Value0.Cmp(big.NewInt(2)) != 0 {
		t.Fatal("allowance not consumed", allowance, err)
	}
	call, err = token.BuildTransferFrom(owner, recipient, big.NewInt(3))
	run(spender, call, err, true)
	checkBalance(recipient, big.NewInt(15))
	checkBalance(owner, ownerBalance)
	call, err = historical.Build("mint(address,uint256)", nil, recipient, big.NewInt(9))
	run(spender, call, err, true)
	checkSupply(initial)
	call, err = historical.Build("mint(address,uint256)", nil, recipient, big.NewInt(9))
	run(owner, call, err, false)
	checkBalance(recipient, big.NewInt(24))
	supply := new(big.Int).Add(new(big.Int).Set(initial), big.NewInt(9))
	checkSupply(supply)
	call, err = token.BuildBurn(big.NewInt(4))
	run(recipient, call, err, false)
	checkBalance(recipient, big.NewInt(20))
	supply.Sub(supply, big.NewInt(4))
	checkSupply(supply)
	call, err = token.BuildBurn(big.NewInt(21))
	run(recipient, call, err, true)
	checkBalance(recipient, big.NewInt(20))
	checkSupply(supply)

	call, err = token.BuildApprove(spender, big.NewInt(6))
	run(recipient, call, err, false)
	call, err = historical.Build("burnFrom(address,uint256)", nil, recipient, big.NewInt(4))
	run(spender, call, err, false)
	checkBalance(recipient, big.NewInt(16))
	supply.Sub(supply, big.NewInt(4))
	checkSupply(supply)
	allowance, err = token.ReadAllowance(ctx, recipient, spender)
	if err != nil || allowance.Value0.Cmp(big.NewInt(2)) != 0 {
		t.Fatal("burn allowance not consumed", allowance, err)
	}
	call, err = historical.Build("burnFrom(address,uint256)", nil, recipient, big.NewInt(3))
	run(spender, call, err, true)
	checkBalance(recipient, big.NewInt(16))
	checkSupply(supply)
	allowance, err = token.ReadAllowance(ctx, recipient, spender)
	if err != nil || allowance.Value0.Cmp(big.NewInt(2)) != 0 {
		t.Fatal("revert changed allowance", allowance, err)
	}
	call, err = token.BuildTransfer(common.Address{}, big.NewInt(1))
	run(owner, call, err, true)
	checkBalance(owner, ownerBalance)
	t.Log(fmt.Sprintf("executed official historical NewToken CREATE and token state transitions at %s; no public transaction or current deployment parity", address))
}
