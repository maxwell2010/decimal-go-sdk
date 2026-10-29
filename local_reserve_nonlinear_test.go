package decimal

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

// These are quote/execution concordance checks at evolving real runtime state,
// not independent mathematical validation of the nonlinear approximation.
func TestLocalEVMReserveNonlinearQuoteExecutionConcordance(t *testing.T) {
	for _, crr := range []uint8{10, 50, 100} {
		t.Run(fmt.Sprint(crr), func(t *testing.T) {
			e, token, owner := newLocalReserveEconomy(t, crr)
			recipient := common.HexToAddress("0x1002")
			requireEvents := func(logs []*types.Log) {
				t.Helper()
				names := map[string]bool{}
				for _, log := range logs {
					event, err := token.DecodeLog(*log)
					if err != nil {
						t.Fatal(err)
					}
					names[event.Name] = true
				}
				if !names["Transfer"] || !names["ReserveUpdated"] {
					t.Fatal("missing actual economy events", names)
				}
			}
			amount := reserveUnits(10)
			quote := reserveRead(t, token, "calculateBuyOutput(uint256)", amount)
			initialOwner := e.balance(owner)
			invalid, err := token.BuildBuyTokenForExactDEL(amount, new(big.Int).Add(quote, big.NewInt(1)), recipient)
			if err != nil {
				t.Fatal(err)
			}
			if _, logs, err := e.execute(t, owner, invalid); err == nil || len(logs) != 0 {
				t.Fatal("slippage should revert", err)
			}
			reserveAssert(t, e.balance(owner), initialOwner, "failed buy caller balance")
			reserveAssert(t, e.balance(token.Address), reserveUnits(100), "failed buy backing")
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), reserveUnits(100), "failed buy supply")
			call, err := token.BuildBuyTokenForExactDEL(amount, quote, recipient)
			if err != nil {
				t.Fatal(err)
			}
			_, logs, err := e.execute(t, owner, call)
			if err != nil {
				t.Fatal(err)
			}
			requireEvents(logs)
			reserveAssert(t, reserveRead(t, token, "balanceOf(address)", recipient), quote, "bought tokens")
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), new(big.Int).Add(reserveUnits(100), quote), "minted supply")
			reserveAssert(t, e.balance(owner), new(big.Int).Sub(initialOwner, amount), "buy debit")
			reserveAssert(t, e.balance(token.Address), reserveUnits(110), "buy native backing")
			reserveAssert(t, reserveRead(t, token, "reserve()"), e.balance(token.Address), "buy reported reserve")
			sold := new(big.Int).Quo(quote, big.NewInt(2))
			payout := reserveRead(t, token, "calculateSellOutput(uint256)", sold)
			beforeRecipient := e.balance(recipient)
			call, err = token.BuildSellExactTokensForDEL(sold, payout, recipient)
			if err != nil {
				t.Fatal(err)
			}
			_, logs, err = e.execute(t, recipient, call)
			if err != nil {
				t.Fatal(err)
			}
			requireEvents(logs)
			reserveAssert(t, reserveRead(t, token, "balanceOf(address)", recipient), new(big.Int).Sub(quote, sold), "remaining tokens")
			reserveAssert(t, reserveRead(t, token, "totalSupply()"), new(big.Int).Sub(new(big.Int).Add(reserveUnits(100), quote), sold), "burned supply")
			reserveAssert(t, e.balance(recipient), new(big.Int).Add(beforeRecipient, payout), "sell payout")
			reserveAssert(t, e.balance(token.Address), new(big.Int).Sub(reserveUnits(110), payout), "sell backing")
			reserveAssert(t, reserveRead(t, token, "reserve()"), e.balance(token.Address), "sell reported reserve")
		})
	}
}
func TestLocalEVMReserveExactOutputNonlinearConcordance(t *testing.T) {
	for _, crr := range []uint8{10, 50, 100} {
		for _, route := range []string{"buy", "sell"} {
			t.Run(fmt.Sprintf("%d/%s", crr, route), func(t *testing.T) {
				e, token, owner := newLocalReserveEconomy(t, crr)
				recipient := common.HexToAddress("0x1002")
				desired := reserveUnits(1)
				beforeOwner := e.balance(owner)
				if route == "buy" {
					charge := reserveRead(t, token, "calculateBuyInput(uint256)", desired)
					budget := new(big.Int).Add(charge, big.NewInt(2))
					call, err := token.BuildBuyExactTokenForDEL(budget, desired, recipient)
					if err != nil {
						t.Fatal(err)
					}
					if _, _, err = e.execute(t, owner, call); err != nil {
						t.Fatal(err)
					}
					reserveAssert(t, e.balance(owner), new(big.Int).Sub(beforeOwner, charge), "exact buy net charge/refund")
					reserveAssert(t, e.balance(token.Address), new(big.Int).Add(reserveUnits(100), charge), "exact buy backing")
					reserveAssert(t, reserveRead(t, token, "totalSupply()"), reserveUnits(101), "exact buy supply")
					reserveAssert(t, reserveRead(t, token, "balanceOf(address)", recipient), desired, "exact buy minted")
				} else {
					burn := reserveRead(t, token, "calculateSellInput(uint256)", desired)
					call, err := token.BuildSellTokensForExactDEL(desired, burn, recipient)
					if err != nil {
						t.Fatal(err)
					}
					if _, _, err = e.execute(t, owner, call); err != nil {
						t.Fatal(err)
					}
					reserveAssert(t, e.balance(recipient), desired, "exact sell payout")
					reserveAssert(t, e.balance(token.Address), reserveUnits(99), "exact sell backing")
					reserveAssert(t, reserveRead(t, token, "totalSupply()"), new(big.Int).Sub(reserveUnits(100), burn), "exact sell supply")
					reserveAssert(t, reserveRead(t, token, "balanceOf(address)", owner), new(big.Int).Sub(reserveUnits(100), burn), "exact sell holder burn")
				}
				reserveAssert(t, reserveRead(t, token, "reserve()"), e.balance(token.Address), "final reserve backing")
			})
		}
	}
}
