// Command messages constructs all 43 message types offline. It neither signs nor
// submits, and it creates one independent TxBody per message, not a combined tx.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"os"
)

type example struct {
	Constructor string   `json:"constructor"`
	TypeURL     string   `json:"type_url"`
	Payload     string   `json:"payload_textproto"`
	Native      bool     `json:"pinned_native_route"`
	BodyHex     string   `json:"body_hex"`
	Signers     []string `json:"signer_address_hex"`
}

func buildExamples() ([]example, error) {
	out := make([]example, 0, len(fixtures))
	for _, f := range fixtures {
		m := f.New()
		raw, e := hex.DecodeString(f.Hex)
		if e != nil {
			return nil, e
		}
		if e = proto.Unmarshal(raw, m); e != nil {
			return nil, e
		}
		if e = cosmos.ValidateBasic(m); e != nil {
			return nil, fmt.Errorf("%s: %w", f.TypeURL, e)
		}
		signers, e := cosmos.GetSigners(m)
		if e != nil {
			return nil, e
		}
		packed, e := cosmos.Pack(m)
		if e != nil {
			return nil, e
		}
		if packed.TypeUrl != f.TypeURL {
			return nil, fmt.Errorf("wrong constructor for %s", f.TypeURL)
		}
		body, e := cosmos.EncodeBody([]proto.Message{m}, "offline example", 0)
		if e != nil {
			return nil, e
		}
		payload := (prototext.MarshalOptions{Multiline: true}).Format(m)
		row := example{Constructor: f.Constructor, TypeURL: f.TypeURL, Payload: payload, Native: cosmos.PinnedNativeRoute(f.TypeURL), BodyHex: hex.EncodeToString(body)}
		for _, s := range signers {
			row.Signers = append(row.Signers, hex.EncodeToString(s))
		}
		out = append(out, row)
	}
	return out, nil
}
func main() {
	dump := flag.Bool("json", false, "print public payloads and offline encoded bodies")
	flag.Parse()
	rows, e := buildExamples()
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	if *dump {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if e = enc.Encode(rows); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		return
	}
	native := 0
	for _, r := range rows {
		if r.Native {
			native++
		}
	}
	fmt.Printf("Validated %d offline message examples: %d pinned native routes, %d historical/schema-only types. No signing or submission.\n", len(rows), native, len(rows)-native)
}
