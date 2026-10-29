// Execute pinned official Msg.ValidateBasic methods against public wire fixtures.
// No app, keeper, network client, environment configuration or wallet is created.
package main

import (
	_ "bitbucket.org/decimalteam/go-smart-node/x/coin/types"
	_ "bitbucket.org/decimalteam/go-smart-node/x/fee/types"
	_ "bitbucket.org/decimalteam/go-smart-node/x/legacy/types"
	_ "bitbucket.org/decimalteam/go-smart-node/x/multisig/types"
	_ "bitbucket.org/decimalteam/go-smart-node/x/nft/types"
	_ "bitbucket.org/decimalteam/go-smart-node/x/swap/types"
	_ "bitbucket.org/decimalteam/go-smart-node/x/validator/types"
	"encoding/hex"
	"encoding/json"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	proto "github.com/gogo/protobuf/proto"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type row struct {
	TypeURL     string `json:"type_url"`
	Case        string `json:"case"`
	Wire        string `json:"protobuf_hex"`
	PortValid   bool   `json:"port_valid"`
	Valid       bool   `json:"valid"`
	Panic       bool   `json:"panic,omitempty"`
	DecodeError bool   `json:"decode_error,omitempty"`
}

func evaluate(r *row) {
	defer func() {
		if recover() != nil {
			r.Panic = true
			r.Valid = false
		}
	}()
	typ := proto.MessageType(strings.TrimPrefix(r.TypeURL, "/"))
	if typ == nil {
		panic("unregistered message")
	}
	message := reflect.New(typ.Elem()).Interface().(proto.Message)
	raw, e := hex.DecodeString(r.Wire)
	if e != nil {
		panic(e)
	}
	if e = proto.Unmarshal(raw, message); e != nil {
		r.DecodeError = true
		return
	}
	validator := message.(interface{ ValidateBasic() error })
	r.Valid = validator.ValidateBasic() == nil
}
func main() {
	sdk.GetConfig().SetBech32PrefixForAccount("d0", "d0pub")
	sdk.GetConfig().SetBech32PrefixForValidator("d0valoper", "d0valoperpub")
	b, e := os.ReadFile("../../schema/validation-denom-whitelist.json")
	if e != nil {
		panic(e)
	}
	var whitelist []string
	if e = json.Unmarshal(b, &whitelist); e != nil {
		panic(e)
	}
	for i, v := range whitelist {
		whitelist[i] = regexp.QuoteMeta(v)
	}
	sdk.SetCoinDenomRegex(func() string { return `[a-zA-Z][a-zA-Z0-9/:._-]{2,127}|` + strings.Join(whitelist, "|") })
	b, e = os.ReadFile("../../schema/validation-node-corpus.json")
	if e != nil {
		panic(e)
	}
	var rows []row
	if e = json.Unmarshal(b, &rows); e != nil {
		panic(e)
	}
	mismatches := 0
	for i := range rows {
		evaluate(&rows[i])
		if rows[i].Valid != rows[i].PortValid {
			mismatches++
			fmt.Fprintf(os.Stderr, "mismatch %s %s port=%v node=%v panic=%v decode=%v\n", rows[i].TypeURL, rows[i].Case, rows[i].PortValid, rows[i].Valid, rows[i].Panic, rows[i].DecodeError)
		}
	}
	if e = json.NewEncoder(os.Stdout).Encode(rows); e != nil {
		panic(e)
	}
	if mismatches > 0 {
		os.Exit(1)
	}
}
