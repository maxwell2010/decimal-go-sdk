package main

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestUnixFSSingleBlockEncoding(t *testing.T) {
	node, e := unixFSNode([]byte("abc"))
	if e != nil {
		t.Fatal(e)
	}
	if hex.EncodeToString(node) != "0a09080212036162631803" {
		t.Fatalf("wrong DAG-PB/UnixFS wire encoding: %x", node)
	}
	if _, e = unixFSNode(nil); e == nil {
		t.Fatal("unsupported empty metadata accepted")
	}
	if _, e = unixFSNode(bytes.Repeat([]byte{1}, 256*1024+1)); e == nil {
		t.Fatal("multi-block metadata silently verified")
	}
}
func TestCIDv0EncodingKnownMultihash(t *testing.T) {
	// CIDv0 encodes the multihash prefix as well as the digest, not its hex text.
	hash, _ := hex.DecodeString("122076f8eaad2ddd373bbeddacc7622f7f53275241720c888d50bbd71a796fd9f1c7")
	if base58(hash) != "QmWM4xhHphrHDXEuPmPfauZkbZkRrHgzjuV8ZN42sYRTGA" {
		t.Fatal("CID encoding changed")
	}
}
