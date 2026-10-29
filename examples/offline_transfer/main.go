// Command offline_transfer builds a DEL transfer intent without RPC or signing.
package main

import (
	"fmt"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"log"
)

func main() {
	// Fixture address only. Choose and verify a real recipient before any submission.
	to, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
	if err != nil {
		log.Fatal(err)
	}
	amount, err := decimal.ParseUnits("1.25", 18)
	if err != nil {
		log.Fatal(err)
	}
	call, err := decimal.NativeTransfer(to, amount, "offline-example")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("to=%s value_wei=%s data=0x%x\n", call.To.Hex(), call.Value, call.Data)
}
