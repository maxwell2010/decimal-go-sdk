// Command checks_history decodes saved public receipt evidence offline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	decimal "github.com/maxwell2010/decimal-go-sdk"
	"os"
)

func main() {
	file := flag.String("file", "docs/compatibility/checks-testnet-receipts.json", "saved public receipt evidence")
	flag.Parse()
	raw, e := os.ReadFile(*file)
	if e != nil {
		panic(e)
	}
	var evidence struct {
		Address  string
		Receipts []*types.Receipt
	}
	if e = json.Unmarshal(raw, &evidence); e != nil {
		panic(e)
	}
	address, e := decimal.ParseAddress(evidence.Address)
	if e != nil {
		panic(e)
	}
	count := 0
	for _, r := range evidence.Receipts {
		rows, e := decimal.ChecksCreationsFromReceipt(address, r)
		if e != nil {
			panic(e)
		}
		for _, row := range rows {
			fmt.Printf("check_hash=%s signer=%s block=%d tx=%s\n", row.CheckHash.Hex(), row.Signer.Hex(), row.Log.BlockNumber, row.Log.TxHash.Hex())
			count++
		}
	}
	fmt.Printf("%d historical creation events; current spendability is unknown\n", count)
}
