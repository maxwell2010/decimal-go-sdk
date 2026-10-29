// Command read_balance reads a public testnet DEL balance without a signer.
package main

import (
	"context"
	"fmt"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"log"
	"os"
	"time"
)

func main() {
	address, err := decimal.ParseAddress(os.Getenv("WALLET_ADDRESS"))
	if err != nil {
		log.Fatal("set WALLET_ADDRESS to a valid public EVM address")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := decimal.Dial(ctx, decimal.Testnet())
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	balance, err := client.Balance(ctx, address)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("testnet address=%s balance=%s DEL\n", address.Hex(), decimal.FormatUnits(balance, 18))
}
