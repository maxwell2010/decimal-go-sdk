// Command token_transfer builds ERC20 calldata using fixture addresses only.
package main

import (
	"fmt"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"log"
)

func main() {
	// Syntactically valid fixtures, not verified deployed token/recipient addresses.
	tokenAddress, err := decimal.ParseAddress("0x2000000000000000000000000000000000000002")
	if err != nil {
		log.Fatal(err)
	}
	recipient, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
	if err != nil {
		log.Fatal(err)
	}
	// The zero-value client supports offline builders. It has no RPC connection.
	client := new(decimal.Client)
	token, err := client.Token(tokenAddress)
	if err != nil {
		log.Fatal(err)
	}
	// Fixture assumes 18 decimals. Read and verify actual token decimals separately.
	amount, err := decimal.ParseUnits("2.5", 18)
	if err != nil {
		log.Fatal(err)
	}
	call, err := token.BuildTransfer(recipient, amount)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("to=%s value_wei=%s signature=%s data=0x%x\n", call.To.Hex(), call.Value, call.Signature, call.Data)
}
