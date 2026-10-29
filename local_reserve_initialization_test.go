package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"math/big"
	"testing"
)

func TestLocalEVMReserveInitializationBoundaries(t *testing.T) {
	for _, name := range []string{"valid", "crr-zero", "crr-above-max", "reserve-below-min", "mint-below-min", "min-above-mint", "max-below-mint", "max-above-global", "min-below-global"} {
		t.Run(name, func(t *testing.T) {
			fixture, source, owner := newLocalReserveEconomy(t, 100)
			e := newLocalEVM(t)
			e.state.SetCode(source.Address, append([]byte(nil), fixture.state.GetCode(source.Address)...), tracing.CodeChangeUnspecified)
			e.fund(owner, reserveUnits(1000))
			token, err := e.client(t).TestnetReserveToken20261029(source.Address)
			if err != nil {
				t.Fatal(err)
			}
			crr := uint8(100)
			reserve, mint, min, max := reserveUnits(100), reserveUnits(100), reserveUnits(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
			switch name {
			case "crr-zero":
				crr = 0
			case "crr-above-max":
				crr = 101
			case "reserve-below-min":
				reserve.Sub(reserveUnits(1), big.NewInt(1))
			case "mint-below-min":
				mint.Sub(reserveUnits(1), big.NewInt(1))
			case "min-above-mint":
				min = reserveUnits(101)
			case "max-below-mint":
				max = reserveUnits(99)
			case "max-above-global":
				max.Add(max, big.NewInt(1))
			case "min-below-global":
				min.Sub(reserveUnits(1), big.NewInt(1))
			}
			call, err := token.BuildInitialize(reserve, "InitReserve", "IRS", owner, crr, mint, min, max, "official local")
			if err != nil {
				t.Fatal(err)
			}
			beforeNative := e.balance(owner)
			out, logs, err := e.execute(t, owner, call)
			allowed := name == "valid" || name == "crr-above-max" || name == "min-above-mint" || name == "max-above-global"
			if (err == nil) != allowed {
				t.Fatalf("observed initialization boundary changed: %s %v %x", name, err, out)
			}
			if allowed {
				if len(logs) != 3 {
					t.Fatal("actual initialization events missing", len(logs))
				}
				reserveAssert(t, reserveRead(t, token, "totalSupply()"), mint, "initialized mint")
				reserveAssert(t, reserveRead(t, token, "minTotalSupply()"), min, "initialized minimum")
				reserveAssert(t, reserveRead(t, token, "maxTotalSupply()"), max, "initialized maximum")
				rate, re := token.ReadCrr(context.Background())
				if re != nil || rate.Value0.Cmp(new(big.Int).SetUint64(uint64(crr))) != 0 {
					t.Fatal("initialized crr", rate, re)
				}
				reserveAssert(t, e.balance(token.Address), reserve, "initialized backing")
				reserveAssert(t, e.balance(owner), new(big.Int).Sub(beforeNative, reserve), "initializer payment")
			} else {
				if len(logs) != 0 {
					t.Fatal("failed initializer retained logs")
				}
				if name == "crr-zero" {
					if len(out) != 36 || common.Bytes2Hex(out[:4]) != "4e487b71" || new(big.Int).SetBytes(out[4:]).Uint64() != 0x12 {
						t.Fatal("expected division-by-zero panic", common.Bytes2Hex(out))
					}
				} else {
					reserveBoundaryKnownError(t, out)
				}
				reserveAssert(t, reserveRead(t, token, "totalSupply()"), big.NewInt(0), "failed initializer supply")
				reserveAssert(t, e.balance(token.Address), big.NewInt(0), "failed initializer backing")
				reserveAssert(t, e.balance(owner), beforeNative, "failed initializer refunded value")
			}
		})
	}
}
func TestLocalEVMReserveAdministrationBoundaries(t *testing.T) {
	for _, name := range []string{"valid-min", "valid-max", "min-above-supply", "max-below-supply", "max-above-global", "min-below-global", "outsider-min", "outsider-max", "reinitialize"} {
		t.Run(name, func(t *testing.T) {
			e, token, owner := newLocalReserveEconomy(t, 100)
			caller := owner
			var call Call
			var err error
			switch name {
			case "valid-min":
				call, err = token.BuildUpdateMinTotalSupply(reserveUnits(2))
			case "valid-max":
				call, err = token.BuildUpdateMaxTotalSupply(reserveUnits(200))
			case "min-above-supply":
				call, err = token.BuildUpdateMinTotalSupply(reserveUnits(101))
			case "max-below-supply":
				call, err = token.BuildUpdateMaxTotalSupply(reserveUnits(99))
			case "max-above-global":
				call, err = token.BuildUpdateMaxTotalSupply(new(big.Int).Add(new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), big.NewInt(1)))
			case "min-below-global":
				call, err = token.BuildUpdateMinTotalSupply(new(big.Int).Sub(reserveUnits(1), big.NewInt(1)))
			case "outsider-min":
				caller = common.HexToAddress("0x1002")
				call, err = token.BuildUpdateMinTotalSupply(reserveUnits(2))
			case "outsider-max":
				caller = common.HexToAddress("0x1002")
				call, err = token.BuildUpdateMaxTotalSupply(reserveUnits(200))
			case "reinitialize":
				call, err = token.BuildInitialize(reserveUnits(100), "Again", "AGA", owner, 100, reserveUnits(100), reserveUnits(1), reserveUnits(200), "again")
			}
			if err != nil {
				t.Fatal(err)
			}
			beforeMin := reserveRead(t, token, "minTotalSupply()")
			beforeMax := reserveRead(t, token, "maxTotalSupply()")
			beforeNative := e.balance(owner)
			out, logs, err := e.execute(t, caller, call)
			allowed := name == "valid-min" || name == "valid-max" || name == "min-above-supply" || name == "max-below-supply" || name == "max-above-global"
			if (err == nil) != allowed {
				t.Fatalf("observed admin boundary changed: %s %v %x", name, err, out)
			}
			if allowed {
				if len(logs) != 1 {
					t.Fatal("admin event missing")
				}
				ev, de := token.DecodeLog(*logs[0])
				if de != nil {
					t.Fatal(de)
				}
				if name == "valid-min" || name == "min-above-supply" {
					want := reserveUnits(2)
					if name == "min-above-supply" {
						want = reserveUnits(101)
					}
					reserveAssert(t, reserveRead(t, token, "minTotalSupply()"), want, "updated minimum")
					if ev.Name != "MinTotalSupplyUpdated" {
						t.Fatal("min event", ev.Name)
					}
					reserveAssert(t, ev.Values["newMinTotalSupply"].(*big.Int), want, "min event value")
				} else {
					want := reserveUnits(200)
					if name == "max-below-supply" {
						want = reserveUnits(99)
					}
					if name == "max-above-global" {
						want.Add(new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), big.NewInt(1))
					}
					reserveAssert(t, reserveRead(t, token, "maxTotalSupply()"), want, "updated maximum")
					if ev.Name != "MaxTotalSupplyUpdated" {
						t.Fatal("max event", ev.Name)
					}
					reserveAssert(t, ev.Values["newMaxTotalSupply"].(*big.Int), want, "max event value")
				}
			} else {
				if len(logs) != 0 {
					t.Fatal("failed admin retained logs")
				}
				reserveBoundaryKnownError(t, out)
				reserveAssert(t, reserveRead(t, token, "minTotalSupply()"), beforeMin, "failed admin minimum")
				reserveAssert(t, reserveRead(t, token, "maxTotalSupply()"), beforeMax, "failed admin maximum")
			}
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), reserveUnits(100), "admin unchanged supply")
			reserveAssert(t, e.balance(token.Address), reserveUnits(100), "admin unchanged backing")
			reserveAssert(t, e.balance(owner), beforeNative, "admin unchanged sender")
		})
	}
}

func reserveBoundaryKnownError(t *testing.T, data []byte) {
	t.Helper()
	parsed, err := loadABI("TestnetReserveToken20261029")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range parsed.Errors {
		if len(data) >= 4 && string(data[:4]) == string(e.ID[:4]) {
			return
		}
	}
	t.Fatal("unknown boundary error", common.Bytes2Hex(data))
}
