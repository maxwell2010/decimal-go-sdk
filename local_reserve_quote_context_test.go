package decimal

import (
	"fmt"
	"math/big"
	"testing"
)

// Independent integer-rational selected price vectors, including changed
// supply after a real burn. This does not certify arbitrary overflow extremes.
func TestLocalEVMReserveCurrentPriceRationalVectors(t *testing.T) {
	for _, crr := range []uint8{10, 50, 100} {
		t.Run(fmt.Sprint(crr), func(t *testing.T) {
			e, token, owner := newLocalReserveEconomy(t, crr)
			check := func() {
				t.Helper()
				supply := reserveRead(t, token, "totalSupply()")
				backing := e.balance(token.Address)
				numerator := new(big.Int).Mul(new(big.Int).Mul(backing, big.NewInt(100)), reserveUnits(1))
				denominator := new(big.Int).Mul(supply, big.NewInt(int64(crr)))
				want := new(big.Int).Quo(numerator, denominator)
				reserveAssert(t, reserveRead(t, token, "getCurrentPrice()"), want, "selected rational price")
			}
			check()
			call, err := token.BuildBurn(reserveUnits(1))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = e.execute(t, owner, call); err != nil {
				t.Fatal(err)
			}
			check()
		})
	}
}
func TestLocalEVMReserveQuoteOverloadsMatchState(t *testing.T) {
	for _, crr := range []uint8{10, 50, 100} {
		t.Run(fmt.Sprint(crr), func(t *testing.T) {
			e, token, owner := newLocalReserveEconomy(t, crr)
			for phase := 0; phase < 2; phase++ {
				supply := reserveRead(t, token, "totalSupply()")
				backing := reserveRead(t, token, "reserve()")
				for _, name := range []string{"calculateBuyInput", "calculateBuyOutput", "calculateSellInput", "calculateSellOutput"} {
					expected := reserveRead(t, token, name+"(uint256,uint256,uint256,uint256)", supply, backing, big.NewInt(int64(crr)), reserveUnits(10))
					reserveAssert(t, reserveRead(t, token, name+"(uint256)", reserveUnits(10)), expected, "state/stateless quote "+name)
				}
				if phase == 0 {
					call, err := token.BuildBurn(reserveUnits(1))
					if err != nil {
						t.Fatal(err)
					}
					if _, _, err = e.execute(t, owner, call); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

// Native funding is an explicit local EVM balance credit, not guessed contract
// storage or a claim that transferring to a deployed token is side-effect free.
// The Before views remove the specified incoming-value context from balance.
func TestLocalEVMReserveBeforeQuotesCreditedBalanceContext(t *testing.T) {
	for _, crr := range []uint8{10, 50, 100} {
		t.Run(fmt.Sprint(crr), func(t *testing.T) {
			e, token, _ := newLocalReserveEconomy(t, crr)
			amount := reserveUnits(10)
			original := e.balance(token.Address)
			supply := reserveRead(t, token, "totalSupply()")
			wantedOutput := reserveRead(t, token, "calculateBuyOutput(uint256,uint256,uint256,uint256)", supply, original, big.NewInt(int64(crr)), amount)
			e.fund(token.Address, amount)
			reserveAssert(t, reserveRead(t, token, "reserve()"), reserveUnits(110), "credited native balance")
			reserveAssert(t, reserveRead(t, token, "calculateBuyOutputBefore(uint256)", amount), wantedOutput, "before output removes incoming amount")
			e.fund(token.Address, reserveUnits(5))
			credited := reserveUnits(15)
			wantedInput := reserveRead(t, token, "calculateBuyInput(uint256,uint256,uint256,uint256)", supply, original, big.NewInt(int64(crr)), amount)
			reserveAssert(t, reserveRead(t, token, "calculateBuyInputBefore(uint256,uint256)", amount, credited), wantedInput, "before input removes explicit incoming value")
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), supply, "quote leaves supply unchanged")
			reserveAssert(t, e.balance(token.Address), reserveUnits(115), "quote leaves credited backing unchanged")
		})
	}
}
