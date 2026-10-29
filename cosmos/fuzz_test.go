package cosmos

import (
	"google.golang.org/protobuf/proto"
	"testing"
)

func FuzzPinnedMessageDecode(f *testing.F) {
	f.Add(uint8(0), []byte{})
	f.Add(uint8(42), []byte{10, 1, 65})
	f.Add(uint8(5), []byte{255, 255, 255})
	f.Fuzz(func(t *testing.T, index uint8, raw []byte) {
		if len(raw) > 4096 {
			return
		}
		inventory := Inventory()
		if len(inventory) == 0 {
			t.Fatal("missing inventory")
		}
		name := inventory[int(index)%len(inventory)]
		decoded, e := Decode(name, raw)
		if e != nil {
			return
		}
		encoded, e := Marshal(decoded)
		if e != nil {
			t.Fatal(e)
		}
		again, e := Decode(name, encoded)
		if e != nil || !proto.Equal(decoded, again) {
			t.Fatal("protobuf roundtrip changed message")
		}
	})
}
