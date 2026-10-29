package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"os"
	"testing"
)

func reserveUnits(n int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(n), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

// Installs only pinned official code and initializes through its actual ABI.
// No copied network storage, reserve formula mock, or public transaction.
func newLocalReserveEconomy(t *testing.T, crr uint8) (*localEVM, *TestnetReserveToken20261029, common.Address) {
	return newLocalReserveEconomyMint(t, crr, reserveUnits(100))
}
func newLocalReserveEconomyMint(t *testing.T, crr uint8, initialMint *big.Int) (*localEVM, *TestnetReserveToken20261029, common.Address) {
	t.Helper()
	raw, err := os.ReadFile("testdata/evm/token-runtime-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := os.ReadFile("testdata/evm/token-runtime-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		SHA   string `json:"snapshot_sha256"`
		Chain uint64 `json:"chain_id"`
		Block uint64 `json:"block"`
		Hash  string `json:"rpc_block_hash"`
	}
	if err = json.Unmarshal(meta, &m); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != m.SHA || m.Chain != 202020 || m.Block != 22477428 || m.Hash != "0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c" {
		t.Fatal("reserve provenance mismatch")
	}
	var snap struct {
		Entries []struct{ Role, Address, Runtime string }
	}
	if err = json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	e := newLocalEVM(t)
	var target common.Address
	for _, r := range snap.Entries {
		if r.Role == "reserve" {
			target = common.HexToAddress(r.Address)
			code := common.FromHex(r.Runtime)
			if target != common.HexToAddress("0x8e215123f9cf7d2c7D99457C5096BeA82a868cbE") || crypto.Keccak256Hash(code).Hex() != "0x67ef16857177e434b860586efd9ff7cf2c57c1adf4ba50bd9310f42f9c8c2cf1" {
				t.Fatal("reserve code identity")
			}
			e.state.SetCode(target, code, tracing.CodeChangeUnspecified)
		}
	}
	if target == (common.Address{}) {
		t.Fatal("reserve code missing")
	}
	token, err := e.client(t).TestnetReserveToken20261029(target)
	if err != nil {
		t.Fatal(err)
	}
	owner := common.HexToAddress("0x1001")
	e.fund(owner, reserveUnits(10000))
	call, err := token.BuildInitialize(reserveUnits(100), "LocalReserve", "LRS", owner, crr, initialMint, reserveUnits(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), "official local fixture")
	if err != nil {
		t.Fatal(err)
	}
	if out, _, err := e.execute(t, owner, call); err != nil {
		t.Fatalf("real reserve initialize failed %v %x", err, out)
	}
	return e, token, owner
}
func reserveRead(t *testing.T, token *TestnetReserveToken20261029, sig string, args ...any) *big.Int {
	t.Helper()
	v, err := token.Read(context.Background(), sig, args...)
	if err != nil || len(v) != 1 {
		t.Fatal("reserve getter", sig, v, err)
	}
	return new(big.Int).Set(v[0].(*big.Int))
}
func reserveAssert(t *testing.T, got, want *big.Int, label string) {
	t.Helper()
	if got.Cmp(want) != 0 {
		t.Fatalf("%s got %s want %s", label, got, want)
	}
}
func TestLocalEVMReserveEconomyLinearLifecycle(t *testing.T) {
	for _, route := range []string{"buy", "buyTokenForExactDEL", "buyExactTokenForDEL", "sell", "sellExactTokensForDEL", "sellTokensForExactDEL", "burn"} {
		t.Run(route, func(t *testing.T) {
			e, token, owner := newLocalReserveEconomy(t, 100)
			recipient := common.HexToAddress("0x1002")
			before := e.balance(owner)
			amount := reserveUnits(10)
			var call Call
			var err error
			switch route {
			case "buy":
				call, err = token.BuildBuy(amount, amount, recipient)
			case "buyTokenForExactDEL":
				call, err = token.BuildBuyTokenForExactDEL(amount, amount, recipient)
			case "buyExactTokenForDEL":
				call, err = token.BuildBuyExactTokenForDEL(reserveUnits(15), amount, recipient)
			case "sell":
				call, err = token.BuildSell(amount, amount, recipient)
			case "sellExactTokensForDEL":
				call, err = token.BuildSellExactTokensForDEL(amount, amount, recipient)
			case "sellTokensForExactDEL":
				call, err = token.BuildSellTokensForExactDEL(amount, reserveUnits(15), recipient)
			case "burn":
				call, err = token.BuildBurn(amount)
			}
			if err != nil {
				t.Fatal(err)
			}
			out, logs, err := e.execute(t, owner, call)
			if err != nil {
				t.Fatalf("%s failed %v %x", route, err, out)
			}
			if len(logs) == 0 {
				t.Fatal("no actual events")
			}
			transferSeen := false
			reserveEventSeen := false
			for _, log := range logs {
				ev, err := token.DecodeLog(*log)
				if err != nil {
					t.Fatal(err)
				}
				if ev.Name == "Transfer" {
					transferSeen = true
				}
				if ev.Name == "ReserveUpdated" {
					reserveEventSeen = true
					eventReserve := reserveRead(t, token, "reserve()")
					switch route {
					case "buyExactTokenForDEL":
						eventReserve = reserveUnits(115)
					case "sell", "sellExactTokensForDEL", "sellTokensForExactDEL":
						eventReserve = reserveUnits(100)
					}
					reserveAssert(t, ev.Values["newReserve"].(*big.Int), eventReserve, "event balance at emission")
					reserveAssert(t, ev.Values["newSupply"].(*big.Int), reserveRead(t, token, "totalSupply()"), "event supply")
				}
			}
			if !reserveEventSeen {
				t.Fatal("missing actual ReserveUpdated event")
			}
			if !transferSeen {
				t.Fatal("missing Transfer")
			}
			supply, reserve := reserveUnits(90), reserveUnits(90)
			switch route {
			case "buy", "buyTokenForExactDEL", "buyExactTokenForDEL":
				supply, reserve = reserveUnits(110), reserveUnits(110)
				reserveAssert(t, reserveRead(t, token, "balanceOf(address)", recipient), amount, "minted tokens")
				reserveAssert(t, e.balance(owner), new(big.Int).Sub(before, amount), "buyer native cost/refund")
			case "burn":
				reserve = reserveUnits(100)
				reserveAssert(t, e.balance(owner), before, "burn has no native payout")
			default:
				reserveAssert(t, e.balance(recipient), amount, "seller native proceeds")
			}
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), supply, "supply")
			reserveAssert(t, reserveRead(t, token, "reserve()"), reserve, "reserve")
			reserveAssert(t, e.balance(token.Address), reserve, "native backing")
		})
	}
}
func TestLocalEVMReserveEconomyRollbackAndLimits(t *testing.T) {
	for _, name := range []string{"buy-slippage", "sell-slippage", "exact-buy-budget", "exact-sell-budget", "over-balance", "below-min-supply", "zero-recipient", "unauthorized-cap", "cap", "below-min-reserve"} {
		t.Run(name, func(t *testing.T) {
			e, token, owner := newLocalReserveEconomy(t, 100)
			if name == "below-min-reserve" {
				e, token, owner = newLocalReserveEconomyMint(t, 100, reserveUnits(1000))
			}
			recipient := common.HexToAddress("0x1002")
			caller := owner
			var call Call
			var err error
			switch name {
			case "buy-slippage":
				call, err = token.BuildBuy(reserveUnits(10), reserveUnits(11), recipient)
			case "sell-slippage":
				call, err = token.BuildSell(reserveUnits(10), reserveUnits(11), recipient)
			case "exact-buy-budget":
				call, err = token.BuildBuyExactTokenForDEL(reserveUnits(9), reserveUnits(10), recipient)
			case "exact-sell-budget":
				call, err = token.BuildSellTokensForExactDEL(reserveUnits(10), reserveUnits(9), recipient)
			case "over-balance":
				call, err = token.BuildSell(reserveUnits(101), big.NewInt(0), recipient)
			case "below-min-supply":
				call, err = token.BuildSell(reserveUnits(100), big.NewInt(0), recipient)
			case "zero-recipient":
				call, err = token.BuildBuy(reserveUnits(10), big.NewInt(0), common.Address{})
			case "unauthorized-cap":
				caller = recipient
				call, err = token.BuildUpdateMaxTotalSupply(reserveUnits(120))
			case "below-min-reserve":
				call, err = token.BuildSell(reserveUnits(999), big.NewInt(0), recipient)
			case "cap":
				capCall, ce := token.BuildUpdateMaxTotalSupply(reserveUnits(105))
				if ce != nil {
					t.Fatal(ce)
				}
				if data, _, ce := e.execute(t, owner, capCall); ce != nil {
					t.Fatalf("cap update %v %x", ce, data)
				}
				call, err = token.BuildBuy(reserveUnits(10), big.NewInt(0), recipient)
			}
			if err != nil {
				t.Fatal(err)
			}
			supply := reserveRead(t, token, "totalSupply()")
			reserve := reserveRead(t, token, "reserve()")
			ownerNative := e.balance(owner)
			ownerTokens := reserveRead(t, token, "balanceOf(address)", owner)
			_, logs, err := e.execute(t, caller, call)
			if err == nil || len(logs) != 0 {
				t.Fatal("expected atomic rejection", name, err, logs)
			}
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), supply, "rollback supply")
			reserveAssert(t, reserveRead(t, token, "reserve()"), reserve, "rollback reserve")
			reserveAssert(t, e.balance(token.Address), reserve, "rollback backing")
			reserveAssert(t, e.balance(owner), ownerNative, "rollback sender")
			reserveAssert(t, reserveRead(t, token, "balanceOf(address)", owner), ownerTokens, "rollback tokens")
			reserveAssert(t, e.balance(recipient), big.NewInt(0), "rollback recipient")
		})
	}
}

func TestLocalEVMReserveDustExactOutputRejectsZeroInput(t *testing.T) {
	for _, buy := range []bool{true, false} {
		t.Run(map[bool]string{true: "buy", false: "sell"}[buy], func(t *testing.T) {
			mint := reserveUnits(1000)
			if !buy {
				mint = reserveUnits(10)
			}
			e, token, owner := newLocalReserveEconomyMint(t, 100, mint)
			recipient := common.HexToAddress("0x1002")
			var call Call
			var err error
			if buy {
				q, qe := token.ReadCalculateBuyInput(context.Background(), big.NewInt(1))
				if qe != nil || q.Value0.Sign() != 0 {
					t.Fatal("expected floor-to-zero buy quote", q, qe)
				}
				call, err = token.BuildBuyExactTokenForDEL(big.NewInt(0), big.NewInt(1), recipient)
			} else {
				q, qe := token.ReadCalculateSellInput(context.Background(), big.NewInt(1))
				if qe != nil || q.Value0.Sign() != 0 {
					t.Fatal("expected floor-to-zero sell quote", q, qe)
				}
				call, err = token.BuildSellTokensForExactDEL(big.NewInt(1), big.NewInt(0), recipient)
			}
			if err != nil {
				t.Fatal(err)
			}
			native := e.balance(owner)
			out, logs, err := e.execute(t, owner, call)
			if err == nil || len(logs) != 0 {
				t.Fatal("zero-input dust trade accepted", err, logs)
			}
			parsed, pe := loadABI("TestnetReserveToken20261029")
			if pe != nil {
				t.Fatal(pe)
			}
			expectedError := "InvalidAmount"
			if buy {
				expectedError = "InvalidValue"
			}
			decoded := false
			for name, e := range parsed.Errors {
				if len(out) >= 4 && string(out[:4]) == string(e.ID[:4]) {
					if name != expectedError {
						t.Fatal("unexpected dust rejection", name)
					}
					decoded = true
				}
			}
			if !decoded {
				t.Fatal("unrecognized dust rejection", common.Bytes2Hex(out))
			}
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), mint, "dust supply rollback")
			reserveAssert(t, reserveRead(t, token, "reserve()"), reserveUnits(100), "dust reserve rollback")
			reserveAssert(t, e.balance(owner), native, "dust sender rollback")
			reserveAssert(t, e.balance(recipient), big.NewInt(0), "dust recipient rollback")
			reserveAssert(t, reserveRead(t, token, "balanceOf(address)", recipient), big.NewInt(0), "dust recipient token rollback")
		})
	}
}

func TestLocalEVMReserveDustPositiveBudgetFollowsFloorQuote(t *testing.T) {
	for _, buy := range []bool{true, false} {
		t.Run(map[bool]string{true: "buy", false: "sell"}[buy], func(t *testing.T) {
			mint := reserveUnits(1000)
			if !buy {
				mint = reserveUnits(10)
			}
			e, token, owner := newLocalReserveEconomyMint(t, 100, mint)
			recipient := common.HexToAddress("0x1002")
			before := e.balance(owner)
			var call Call
			var err error
			if buy {
				call, err = token.BuildBuyExactTokenForDEL(big.NewInt(1), big.NewInt(1), recipient)
			} else {
				call, err = token.BuildSellTokensForExactDEL(big.NewInt(1), big.NewInt(1), recipient)
			}
			if err != nil {
				t.Fatal(err)
			}
			out, logs, err := e.execute(t, owner, call)
			if err != nil || len(logs) != 2 {
				t.Fatalf("observed positive-budget dust path changed: %v %x", err, out)
			}
			reserveAssert(t, e.balance(owner), before, "dust offered budget refunded/unused")
			if buy {
				reserveAssert(t, reserveRead(t, token, "balanceOf(address)", recipient), big.NewInt(1), "dust minted output")
				reserveAssert(t, reserveRead(t, token, "totalSupply()"), new(big.Int).Add(mint, big.NewInt(1)), "dust minted supply")
				reserveAssert(t, reserveRead(t, token, "reserve()"), reserveUnits(100), "dust zero funded quote")
			} else {
				reserveAssert(t, e.balance(recipient), big.NewInt(1), "dust native output")
				reserveAssert(t, reserveRead(t, token, "totalSupply()"), mint, "dust zero burn quote")
				reserveAssert(t, reserveRead(t, token, "reserve()"), new(big.Int).Sub(reserveUnits(100), big.NewInt(1)), "dust reduced backing")
			}
			reserveAssert(t, reserveRead(t, token, "balanceOf(address)", owner), mint, "dust owner token balance unchanged")
			reserveAssert(t, e.balance(token.Address), reserveRead(t, token, "reserve()"), "dust actual backing")
		})
	}
}
