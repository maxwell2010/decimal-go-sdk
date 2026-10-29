package decimal

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func decodingReceiptSeed(tb testing.TB) (common.Address, *types.Receipt) {
	tb.Helper()
	raw, e := os.ReadFile("docs/compatibility/checks-testnet-receipts.json")
	if e != nil {
		tb.Fatal(e)
	}
	var saved struct {
		Address  string
		Receipts []*types.Receipt
	}
	if e = json.Unmarshal(raw, &saved); e != nil || len(saved.Receipts) != 1 {
		tb.Fatal("invalid official receipt seed", e)
	}
	return common.HexToAddress(saved.Address), saved.Receipts[0]
}

func FuzzChecksEventDecoding(f *testing.F) {
	address, receipt := decodingReceiptSeed(f)
	abi, e := loadABI("Checks")
	if e != nil {
		f.Fatal(e)
	}
	var seeds []types.Log
	for _, log := range receipt.Logs {
		if log.Address == address && len(log.Topics) == 1 && (log.Topics[0] == abi.Events["CreatedCheck"].ID || log.Topics[0] == abi.Events["CreatedCheckDetails"].ID) {
			seeds = append(seeds, *log)
		}
	}
	if len(seeds) != 2 {
		f.Fatal("official seed events missing")
	}
	for i, seed := range seeds {
		f.Add(seed.Data, uint8(i), uint8(0))
		bad := bytes.Clone(seed.Data)
		bad[64] = 0xff
		f.Add(bad, uint8(i), uint8(0))
		f.Add(seed.Data[:len(seed.Data)-1], uint8(i), uint8(0))
	}
	f.Add([]byte{}, uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, kind, flags uint8) {
		if len(data) > 1024 {
			return
		}
		log := seeds[int(kind)%len(seeds)]
		log.Data = bytes.Clone(data)
		log.Topics = append([]common.Hash(nil), log.Topics...)
		if flags&1 != 0 {
			log.Removed = true
		}
		if flags&2 != 0 {
			log.Address = common.Address{}
		}
		if flags&4 != 0 {
			log.Topics = append(log.Topics, common.Hash{})
		}
		if flags&8 != 0 {
			log.BlockHash = common.Hash{}
		}
		before := bytes.Clone(log.Data)
		var first, second any
		var err1, err2 error
		if log.Topics[0] == abi.Events["CreatedCheck"].ID {
			first, err1 = DecodeChecksCreationLog(address, log)
			second, err2 = DecodeChecksCreationLog(address, log)
		} else {
			first, err1 = DecodeChecksCreationDetailsLog(address, log)
			second, err2 = DecodeChecksCreationDetailsLog(address, log)
		}
		if (err1 == nil) != (err2 == nil) || !reflect.DeepEqual(first, second) || !bytes.Equal(before, log.Data) {
			t.Fatal("nondeterministic decode or mutated input")
		}
		if err1 != nil {
			return
		}
		snapshot, _ := json.Marshal(first)
		for i := range log.Data {
			log.Data[i] ^= 0xff
		}
		log.Topics[0] = common.Hash{}
		after, _ := json.Marshal(first)
		if !bytes.Equal(snapshot, after) {
			t.Fatal("successful decoder borrowed input buffers")
		}
	})
}

func FuzzChecksReceiptDecoding(f *testing.F) {
	address, receipt := decodingReceiptSeed(f)
	seed, e := json.Marshal(receipt)
	if e != nil {
		f.Fatal(e)
	}
	f.Add(seed)
	f.Add([]byte(`null`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"status":"0x1","logs":[null]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 16384 {
			return
		}
		var r *types.Receipt
		if json.Unmarshal(data, &r) != nil {
			return
		}
		first, e1 := ChecksCreationsFromReceipt(address, r)
		second, e2 := ChecksCreationsFromReceipt(address, r)
		if (e1 == nil) != (e2 == nil) || !reflect.DeepEqual(first, second) {
			t.Fatal("receipt decode nondeterministic")
		}
		if e1 != nil {
			return
		}
		before, _ := json.Marshal(first)
		if r != nil {
			if r.BlockNumber != nil {
				r.BlockNumber.SetInt64(0)
			}
			for _, log := range r.Logs {
				if log != nil {
					for i := range log.Data {
						log.Data[i] ^= 0xff
					}
					for i := range log.Topics {
						log.Topics[i] = common.Hash{}
					}
				}
			}
		}
		after, _ := json.Marshal(first)
		if !bytes.Equal(before, after) {
			t.Fatal("receipt decode borrowed input memory")
		}
	})
}

func FuzzChecksSignatureRecovery(f *testing.F) {
	raw, e := os.ReadFile("docs/checks-js-vectors.json")
	if e != nil {
		f.Fatal(e)
	}
	var fixture struct {
		Vectors []struct {
			Signature string
			CheckHash string `json:"check_hash"`
			Redeemer  string
		}
	}
	if e = json.Unmarshal(raw, &fixture); e != nil || len(fixture.Vectors) == 0 {
		f.Fatal("official JS vectors unavailable", e)
	}
	for _, v := range fixture.Vectors {
		f.Add(common.FromHex(v.Signature), common.FromHex(v.CheckHash), common.FromHex(v.Redeemer))
	}
	f.Add([]byte{}, []byte{}, []byte{})
	f.Add(make([]byte, 65), make([]byte, 32), make([]byte, 20))
	f.Fuzz(func(t *testing.T, sig, hash, recipient []byte) {
		if len(sig) > 256 || len(hash) > 64 || len(recipient) > 40 {
			return
		}
		before := bytes.Clone(sig)
		h := common.BytesToHash(hash)
		a := common.BytesToAddress(recipient)
		owner, e1 := RecoverEVMCheckRedemptionSigner(h, a, sig)
		again, e2 := RecoverEVMCheckRedemptionSigner(h, a, sig)
		if (e1 == nil) != (e2 == nil) || owner != again || !bytes.Equal(before, sig) {
			t.Fatal("unstable recovery or mutated signature")
		}
		if e1 != nil {
			return
		}
		if len(sig) != 65 {
			t.Fatal("accepted invalid signature length")
		}
		canonical := bytes.Clone(sig)
		if canonical[64] >= 27 {
			canonical[64] -= 27
		}
		digest := EVMCheckRedemptionDigest(h, a)
		pub, e := crypto.SigToPub(digest[:], canonical)
		if e != nil || crypto.PubkeyToAddress(*pub) != owner || !crypto.VerifySignature(crypto.FromECDSAPub(pub), digest[:], canonical[:64]) {
			t.Fatal("recovered proof fails independent signature verification")
		}
		canonical[64] += 27
		normalized, e := RecoverEVMCheckRedemptionSigner(h, a, canonical)
		if e != nil || normalized != owner {
			t.Fatal("recovery ID forms disagree")
		}
	})
}
