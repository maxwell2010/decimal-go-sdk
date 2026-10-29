package main

import (
	"encoding/json"
	cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos"
	"os"
	"strings"
	"testing"
)

func TestAllPinnedMessagesHaveValidNamedConstructorExamples(t *testing.T) {
	rows, e := buildExamples()
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 43 || len(rows) != len(cosmos.Inventory()) {
		t.Fatal("incomplete examples")
	}
	seen := map[string]bool{}
	native := 0
	// Compare exact public wire inputs, so examples cannot silently drift away
	// from fixtures executed by the official node validation reference.
	raw, e := os.ReadFile("../../schema/validation-node-corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	var wire []struct {
		Case    string `json:"case"`
		TypeURL string `json:"type_url"`
		Hex     string `json:"protobuf_hex"`
	}
	if e = json.Unmarshal(raw, &wire); e != nil {
		t.Fatal(e)
	}
	valid := map[string]string{}
	for _, v := range wire {
		if v.Case == "valid" {
			valid[v.TypeURL] = v.Hex
		}
	}
	for _, doc := range []string{"../../TRANSACTIONS.md", "../../TRANSACTIONS.ru.md"} {
		text, e := os.ReadFile(doc)
		if e != nil {
			t.Fatal(e)
		}
		for _, r := range rows {
			if !strings.Contains(string(text), r.TypeURL) || !strings.Contains(string(text), r.Constructor) {
				t.Fatal("incomplete documentation", doc, r.TypeURL)
			}
		}
	}
	for i, r := range rows {
		if seen[r.TypeURL] {
			t.Fatal("duplicate", r.TypeURL)
		}
		seen[r.TypeURL] = true
		if r.Native {
			native++
		}
		if len(r.Signers) == 0 || len(r.BodyHex) == 0 || r.Payload == "" {
			t.Fatal("incomplete row", r.TypeURL)
		}
		if valid[r.TypeURL] != fixtures[i].Hex {
			t.Fatal("wire differs from executed official reference", r.TypeURL)
		}
	}
	for _, name := range cosmos.Inventory() {
		if !seen[name] {
			t.Fatal("missing", name)
		}
	}
	if native != 8 {
		t.Fatal("admission split", native)
	}
}
