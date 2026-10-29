package decimal

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"os"
	"testing"
)

func reserveOracleContract(t *testing.T) *TestnetReserveToken20261029 {
	t.Helper()
	data, err := os.ReadFile("testdata/evm/token-runtime-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Entries []struct{ Role, Address, Runtime string }
	}
	if err = json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	e := newLocalEVM(t)
	for _, entry := range snapshot.Entries {
		if entry.Role == "reserve" {
			code := common.FromHex(entry.Runtime)
			if crypto.Keccak256Hash(code).Hex() != "0x67ef16857177e434b860586efd9ff7cf2c57c1adf4ba50bd9310f42f9c8c2cf1" {
				t.Fatal("reserve runtime changed")
			}
			address := common.HexToAddress(entry.Address)
			e.state.SetCode(address, code, tracing.CodeChangeUnspecified)
			token, err := e.client(t).TestnetReserveToken20261029(address)
			if err != nil {
				t.Fatal(err)
			}
			return token
		}
	}
	t.Fatal("missing reserve runtime")
	return nil
}

// CRR 100 is the linear boundary of a constant-reserve-ratio curve. The oracle
// uses integer rational arithmetic, not another contract quote or float64. It
// checks only the pinned runtime's stateless branch, not economic feasibility.
func TestLocalReserveOracleCRR100IntegerRounding(t *testing.T) {
	token := reserveOracleContract(t)
	ctx := context.Background()
	methods := []struct {
		sig     string
		inverse bool
	}{
		{"calculateBuyOutput(uint256,uint256,uint256,uint256)", false},
		{"calculateSellInput(uint256,uint256,uint256,uint256)", false},
		{"calculateBuyInput(uint256,uint256,uint256,uint256)", true},
		{"calculateSellOutput(uint256,uint256,uint256,uint256)", true},
	}
	for _, m := range methods {
		t.Run(m.sig, func(t *testing.T) {
			for i := int64(0); i < 96; i++ {
				supply := big.NewInt(1000 + 31*i)
				reserve := big.NewInt(100 + 17*i)
				amount := big.NewInt(i*i + 1)
				if i == 0 {
					amount.SetInt64(0)
				}
				num, den := supply, reserve
				if m.inverse {
					num, den = reserve, supply
				}
				product := new(big.Int).Mul(num, amount)
				want, remainder := new(big.Int), new(big.Int)
				want.QuoRem(product, den, remainder)
				values, err := token.Read(ctx, m.sig, supply, reserve, big.NewInt(100), amount)
				if err != nil {
					t.Fatal(err)
				}
				got := values[0].(*big.Int)
				if got.Cmp(want) != 0 {
					t.Fatalf("S=%v R=%v amount=%v got=%v wantfloor=%v remainder=%v", supply, reserve, amount, got, want, remainder)
				}
				lower := new(big.Int).Mul(got, den)
				upper := new(big.Int).Mul(new(big.Int).Add(got, big.NewInt(1)), den)
				if lower.Cmp(product) > 0 || upper.Cmp(product) <= 0 {
					t.Fatal("rational floor bounds violated")
				}
			}
		})
	}
	// Quotes use floor even for exact-output input estimators. A quote is not a
	// promise that a write can mint/burn the requested amount or stay above minima.
	got, err := token.ReadCalculateBuyInput1(ctx, big.NewInt(1000), big.NewInt(100), big.NewInt(100), big.NewInt(1))
	if err != nil || got.Value0.Sign() != 0 {
		t.Fatal("dust inverse quote changed", got, err)
	}
	oversized, err := token.ReadCalculateSellInput1(ctx, big.NewInt(1000), big.NewInt(100), big.NewInt(100), big.NewInt(101))
	if err != nil || oversized.Value0.Cmp(big.NewInt(1010)) != 0 {
		t.Fatal("stateless quote unexpectedly enforces available supply", oversized, err)
	}
}

func TestLocalReserveOracleCRR50PerfectSquareVectors(t *testing.T) {
	token := reserveOracleContract(t)
	ctx := context.Background()
	for _, v := range []struct {
		method       string
		amount, want int64
	}{
		{"calculateBuyInput", 100, 300}, {"calculateBuyOutput", 300, 100},
		{"calculateSellOutput", 50, 75}, {"calculateSellInput", 75, 50},
	} {
		t.Run(v.method, func(t *testing.T) {
			values, err := token.Read(ctx, v.method+"(uint256,uint256,uint256,uint256)", reserveUnits(100), reserveUnits(100), big.NewInt(50), reserveUnits(v.amount))
			if err != nil {
				t.Fatal(err)
			}
			if values[0].(*big.Int).Cmp(reserveUnits(v.want)) != 0 {
				t.Fatalf("perfect-square vector got %s want %s", values[0], reserveUnits(v.want))
			}
		})
	}
}
