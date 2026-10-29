// Independent reference using actual pinned Cosmos/Ethermint implementations.
// Only public deterministic test material is used; no network operations.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tx "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	eth "github.com/decimalteam/ethermint/crypto/ethsecp256k1"
	"os"
)

func main() {
	key := eth.PrivKey{Key: bytes.Repeat([]byte{1}, 32)}
	pub := key.PubKey()
	any, e := codectypes.NewAnyWithValue(pub)
	if e != nil {
		panic(e)
	}
	payload, e := hex.DecodeString("0a08643073656e646572120b6430726563697069656e741a0a0a0364656c1203313233")
	if e != nil {
		panic(e)
	}
	body := tx.TxBody{Messages: []*codectypes.Any{{TypeUrl: "/decimal.coin.v1.MsgSendCoin", Value: payload}}, Memo: "reference", TimeoutHeight: 7}
	bodyBytes, e := body.Marshal()
	if e != nil {
		panic(e)
	}
	auth := tx.AuthInfo{SignerInfos: []*tx.SignerInfo{{PublicKey: any, Sequence: 9, ModeInfo: &tx.ModeInfo{Sum: &tx.ModeInfo_Single_{Single: &tx.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT}}}}}, Fee: &tx.Fee{Amount: sdk.NewCoins(sdk.NewInt64Coin("del", 7)), GasLimit: 200000}}
	authBytes, e := auth.Marshal()
	if e != nil {
		panic(e)
	}
	doc := tx.SignDoc{BodyBytes: bodyBytes, AuthInfoBytes: authBytes, ChainId: "decimal_test-1", AccountNumber: 11}
	signBytes, e := doc.Marshal()
	if e != nil {
		panic(e)
	}
	sig, e := key.Sign(signBytes)
	if e != nil {
		panic(e)
	}
	raw := tx.TxRaw{BodyBytes: bodyBytes, AuthInfoBytes: authBytes, Signatures: [][]byte{sig}}
	rawBytes, e := raw.Marshal()
	if e != nil {
		panic(e)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"body": hex.EncodeToString(bodyBytes), "auth": hex.EncodeToString(authBytes), "sign_doc": hex.EncodeToString(signBytes), "signature": hex.EncodeToString(sig), "tx_raw": hex.EncodeToString(rawBytes), "public_key": hex.EncodeToString(pub.Bytes())})
}
