package decimal_test

import (
	"fmt"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"math/big"
)

func ExampleParseUnits() {
	amount, err := decimal.ParseUnits("1.25", 18)
	if err != nil {
		panic(err)
	}
	fmt.Println(amount)
	fmt.Println(decimal.FormatUnits(amount, 18))
	// Output:
	// 1250000000000000000
	// 1.25
}
func ExampleNativeTransfer() {
	to, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
	if err != nil {
		panic(err)
	}
	call, err := decimal.NativeTransfer(to, big.NewInt(5), "invoice-123")
	if err != nil {
		panic(err)
	}
	fmt.Println(call.Value, string(call.Data))
	// Output: 5 invoice-123
}
