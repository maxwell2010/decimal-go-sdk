package main

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"testing"
)

func TestPreparedEnvelopeFixtures(t *testing.T) {
	prepared, err := prepareExamples(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 3 {
		t.Fatal(len(prepared))
	}
	for i, p := range prepared {
		tx := p.Transaction
		if tx.Type() != uint8(i) || tx.Nonce() != 7 || tx.Gas() != 23100 || tx.Value().Int64() != 123 {
			t.Fatalf("unexpected envelope %d", i)
		}
		v, r, s := tx.RawSignatureValues()
		if v.Sign() != 0 || r.Sign() != 0 || s.Sign() != 0 {
			t.Fatal("example signed a transaction")
		}
		wantReserved, wantEstimate := int64(231000), int64(210000)
		if i == 2 {
			wantReserved, wantEstimate = 462000, 147000
			if tx.GasFeeCap().Int64() != 20 || tx.GasTipCap().Int64() != 2 {
				t.Fatal("dynamic caps")
			}
		}
		if p.Fee.ReservedFee.Int64() != wantReserved || p.Fee.EstimatedFee.Int64() != wantEstimate || p.Fee.RequiredBalance.Int64() != wantReserved+123 {
			t.Fatalf("fee quote %d", i)
		}
		if i > 0 && (tx.ChainId().Int64() != 202020 || len(tx.AccessList()) != 1) {
			t.Fatal("typed envelope fields")
		}
		raw, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var decoded types.Transaction
		if err = decoded.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		if decoded.Type() != tx.Type() || decoded.Gas() != tx.Gas() {
			t.Fatal("wire roundtrip")
		}
	}
}
