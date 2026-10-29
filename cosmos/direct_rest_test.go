package cosmos

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fixtureTx(t *testing.T) DirectTx {
	t.Helper()
	body, e := EncodeBody([]proto.Message{NewMsgSendCoin()}, "original", 0)
	if e != nil {
		t.Fatal(e)
	}
	key, e := crypto.HexToECDSA(strings.Repeat("01", 32))
	if e != nil {
		t.Fatal(e)
	}
	pub := crypto.CompressPubkey(&key.PublicKey)
	return DirectTx{Body: body, ChainID: "decimal_test-1", Fee: []Coin{{"del", "1"}}, Gas: 1000, SignerOrder: [][]byte{pub}, Signers: []DirectSigner{{PublicKey: pub, AccountNumber: 7, Sequence: 9, Sign: func(_ context.Context, h [32]byte) ([]byte, error) { return crypto.Sign(h[:], key) }}}}
}
func TestDirectSignerOrderMutationAndRecovery(t *testing.T) {
	tx := fixtureTx(t)
	original := append([]byte(nil), tx.Body...)
	first := tx.Signers[0]
	key, _ := crypto.HexToECDSA(strings.Repeat("02", 32))
	secondPub := crypto.CompressPubkey(&key.PublicKey)
	tx.Signers = append(tx.Signers, DirectSigner{PublicKey: secondPub, AccountNumber: 8, Sequence: 10, Sign: func(_ context.Context, h [32]byte) ([]byte, error) { tx.Body[0] ^= 1; return crypto.Sign(h[:], key) }})
	tx.SignerOrder = [][]byte{secondPub, first.PublicKey}
	result, e := SignDirectTx(context.Background(), tx)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(result.Body, original) {
		t.Fatal("callback changed signed intent")
	}
	auth, _ := Decode("cosmos.tx.v1beta1.AuthInfo", result.AuthInfo)
	info := list(auth, "signer_infos")
	if info.Get(0).Message().Get(info.Get(0).Message().Descriptor().Fields().ByName("sequence")).Uint() != 10 {
		t.Fatal("signer ordering ignored")
	}
	bad := fixtureTx(t)
	bad.Signers[0].Sign = func(_ context.Context, h [32]byte) ([]byte, error) { return crypto.Sign(h[:], key) }
	if _, e = SignDirectTx(context.Background(), bad); e == nil {
		t.Fatal("wrong key accepted")
	}
	duplicate := fixtureTx(t)
	duplicate.SignerOrder = append(duplicate.SignerOrder, duplicate.SignerOrder[0])
	duplicate.Signers = append(duplicate.Signers, duplicate.Signers[0])
	if _, e = SignDirectTx(context.Background(), duplicate); e == nil {
		t.Fatal("duplicate signer")
	}
	ctx, cancel := context.WithCancel(context.Background())
	canceled := fixtureTx(t)
	sign := canceled.Signers[0].Sign
	canceled.Signers[0].Sign = func(ctx context.Context, h [32]byte) ([]byte, error) { s, e := sign(ctx, h); cancel(); return s, e }
	if _, e = SignDirectTx(ctx, canceled); !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
}
func TestRESTQueriesSimulationAndSingleSubmission(t *testing.T) {
	signed, e := SignDirectTx(context.Background(), fixtureTx(t))
	if e != nil {
		t.Fatal(e)
	}
	broadcasts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "accounts/d0fixture"):
			_, _ = w.Write([]byte(`{"account":{"base_account":{"address":"d0fixture","account_number":"7","sequence":"9"}}}`))
		case strings.HasSuffix(r.URL.Path, "calculate_commission"):
			if r.URL.Query().Get("tx_bytes") != "0102" {
				t.Error("commission expects HEX")
			}
			_, _ = w.Write([]byte(`{"commission":"99"}`))
		case strings.HasSuffix(r.URL.Path, "simulate"):
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			raw, e := base64.StdEncoding.DecodeString(payload["tx_bytes"])
			if e != nil || !bytes.Equal(raw, signed.Raw) {
				t.Error("simulation bytes")
			}
			_, _ = w.Write([]byte(`{"gas_info":{"gas_wanted":"1000","gas_used":"900"}}`))
		case r.Method == http.MethodPost:
			broadcasts++
			http.Error(w, "private backend details", 500)
		default:
			_, _ = w.Write([]byte(`{"tx_response":{"txhash":"` + hex.EncodeToString(signed.Hash[:]) + `","height":"5","code":0}}`))
		}
	}))
	defer server.Close()
	client, _ := NewREST(server.URL, nil)
	account, e := client.Account(context.Background(), "d0fixture")
	if e != nil || account.Number != 7 || account.Sequence != 9 {
		t.Fatal("account", e)
	}
	fee, e := client.Commission(context.Background(), []byte{1, 2}, "del")
	if e != nil || fee != "99" {
		t.Fatal("commission", e)
	}
	gas, e := client.Simulate(context.Background(), signed.Raw)
	if e != nil || gas.Used != "900" {
		t.Fatal("simulate", e)
	}
	_, e = client.Broadcast(context.Background(), signed.Raw)
	var unknown *SubmissionUnknownError
	if !errors.As(e, &unknown) || unknown.Hash != signed.Hash || broadcasts != 1 {
		t.Fatal("ambiguous submission retried or hash lost")
	}
	if _, e = client.Transaction(context.Background(), signed.Hash); e != nil {
		t.Fatal(e)
	}
}
