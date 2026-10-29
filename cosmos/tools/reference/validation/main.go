// Execute actual Cosmos SDK46 upgrade ValidateBasic without network or wallets.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	upgrade "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"os"
)

func main() {
	sdk.GetConfig().SetBech32PrefixForAccount("d0", "d0pub")
	address := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	rows := []map[string]any{}
	for _, height := range []int64{-1, 0, 1, 42} {
		for _, name := range []string{"", "upgrade"} {
			m := upgrade.MsgSoftwareUpgrade{Authority: address, Plan: upgrade.Plan{Name: name, Height: height}}
			b, e := m.Marshal()
			if e != nil {
				panic(e)
			}
			rows = append(rows, map[string]any{"type_url": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", "protobuf_hex": hex.EncodeToString(b), "valid": m.ValidateBasic() == nil})
		}
	}
	for _, authority := range []string{address, "", "invalid"} {
		m := upgrade.MsgCancelUpgrade{Authority: authority}
		b, e := m.Marshal()
		if e != nil {
			panic(e)
		}
		rows = append(rows, map[string]any{"type_url": "/cosmos.upgrade.v1beta1.MsgCancelUpgrade", "protobuf_hex": hex.EncodeToString(b), "valid": m.ValidateBasic() == nil})
	}
	if e := json.NewEncoder(os.Stdout).Encode(rows); e != nil {
		panic(e)
	}
}
