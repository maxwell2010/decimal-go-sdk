package cosmos

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"google.golang.org/protobuf/reflect/protoreflect"
	"os"
	"testing"
)

var exportValidationCorpus = flag.Bool("export-validation-corpus", false, "write deterministic public offline node reference fixtures")

func TestExportNodeValidationCorpus(t *testing.T) {
	if !*exportValidationCorpus {
		t.Skip("explicit fixture generation only")
	}
	rows := []map[string]any{}
	for _, name := range Inventory() {
		for _, variant := range []string{"valid", "empty", "bad-signer"} {
			m, _ := NewMessage(name)
			if variant != "empty" {
				fillValidationFixture(m.ProtoReflect())
			}
			if variant == "bad-signer" {
				rule := signerRules[name]
				m.Set(m.Descriptor().Fields().ByName(protoreflect.Name(rule.field)), protoreflect.ValueOfString("invalid"))
			}
			raw, e := Marshal(m)
			if e != nil {
				t.Fatal(e)
			}
			rows = append(rows, map[string]any{"type_url": name, "case": variant, "protobuf_hex": hex.EncodeToString(raw), "port_valid": ValidateBasic(m) == nil})
		}
	}
	raw, e := json.MarshalIndent(rows, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile("schema/validation-node-corpus.json", append(raw, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
