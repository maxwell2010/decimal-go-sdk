// Package cosmos implements offline codecs from pinned official Decimal and
// Cosmos SDK descriptors. It does not broadcast or assume active message routes.
package cosmos

import (
	"crypto/sha256"
	_ "embed"
	"errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"sort"
	"strings"
)

//go:embed schema/decimal.pb
var descriptorBytes []byte
var files *protoregistry.Files
var inventory []string

func init() {
	set := new(descriptorpb.FileDescriptorSet)
	if e := proto.Unmarshal(descriptorBytes, set); e != nil {
		panic(e)
	}
	var e error
	files, e = protodesc.NewFiles(set)
	if e != nil {
		panic(e)
	}
	seen := map[string]bool{}
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		s := f.Services().ByName("Msg")
		if s != nil {
			for i := 0; i < s.Methods().Len(); i++ {
				name := "/" + string(s.Methods().Get(i).Input().FullName())
				if !seen[name] {
					inventory = append(inventory, name)
					seen[name] = true
				}
			}
		}
		return true
	})
	sort.Strings(inventory)
}
func Inventory() []string { return append([]string(nil), inventory...) }
func NewMessage(typeURL string) (*dynamicpb.Message, error) {
	name := strings.TrimPrefix(typeURL, "/")
	d, e := files.FindDescriptorByName(protoreflect.FullName(name))
	if e != nil {
		return nil, errors.New("unknown pinned protobuf message")
	}
	m, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, errors.New("descriptor is not a message")
	}
	return dynamicpb.NewMessage(m), nil
}
func Marshal(message proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(message)
}
func Pack(message proto.Message) (*anypb.Any, error) {
	if message == nil {
		return nil, errors.New("message required")
	}
	b, e := Marshal(message)
	if e != nil {
		return nil, e
	}
	return &anypb.Any{TypeUrl: "/" + string(message.ProtoReflect().Descriptor().FullName()), Value: b}, nil
}
func Unpack(any *anypb.Any) (*dynamicpb.Message, error) {
	if any == nil {
		return nil, errors.New("Any required")
	}
	m, e := NewMessage(any.TypeUrl)
	if e != nil {
		return nil, e
	}
	if e = proto.Unmarshal(any.Value, m); e != nil {
		return nil, e
	}
	return m, nil
}
func Decode(typeURL string, b []byte) (*dynamicpb.Message, error) {
	m, e := NewMessage(typeURL)
	if e != nil {
		return nil, e
	}
	if e = proto.Unmarshal(b, m); e != nil {
		return nil, e
	}
	return m, nil
}
func message(name string) *dynamicpb.Message {
	m, e := NewMessage(name)
	if e != nil {
		panic(e)
	}
	return m
}
func set(m *dynamicpb.Message, name string, value protoreflect.Value) {
	m.Set(m.Descriptor().Fields().ByName(protoreflect.Name(name)), value)
}
func list(m *dynamicpb.Message, name string) protoreflect.List {
	return m.Mutable(m.Descriptor().Fields().ByName(protoreflect.Name(name))).List()
}

type Coin struct{ Denom, Amount string }

func EncodeBody(messages []proto.Message, memo string, timeoutHeight uint64) ([]byte, error) {
	m := message("cosmos.tx.v1beta1.TxBody")
	for _, msg := range messages {
		a, e := Pack(msg)
		if e != nil {
			return nil, e
		}
		list(m, "messages").Append(protoreflect.ValueOfMessage(a.ProtoReflect()))
	}
	set(m, "memo", protoreflect.ValueOfString(memo))
	set(m, "timeout_height", protoreflect.ValueOfUint64(timeoutHeight))
	return Marshal(m)
}

type SignerInfo struct {
	PublicKey []byte
	Sequence  uint64
}

func EncodeAuthInfo(signers []SignerInfo, fee []Coin, gas uint64) ([]byte, error) {
	if len(signers) == 0 {
		return nil, errors.New("signers required")
	}
	m := message("cosmos.tx.v1beta1.AuthInfo")
	for _, s := range signers {
		if len(s.PublicKey) != 33 {
			return nil, errors.New("compressed Ethermint public key required")
		}
		pub := message("ethermint.crypto.v1.ethsecp256k1.PubKey")
		set(pub, "key", protoreflect.ValueOfBytes(append([]byte(nil), s.PublicKey...)))
		a, e := Pack(pub)
		if e != nil {
			return nil, e
		}
		info := message("cosmos.tx.v1beta1.SignerInfo")
		set(info, "public_key", protoreflect.ValueOfMessage(a.ProtoReflect()))
		mode := message("cosmos.tx.v1beta1.ModeInfo")
		single := message("cosmos.tx.v1beta1.ModeInfo.Single")
		set(single, "mode", protoreflect.ValueOfEnum(1))
		set(mode, "single", protoreflect.ValueOfMessage(single))
		set(info, "mode_info", protoreflect.ValueOfMessage(mode))
		set(info, "sequence", protoreflect.ValueOfUint64(s.Sequence))
		list(m, "signer_infos").Append(protoreflect.ValueOfMessage(info))
	}
	f := message("cosmos.tx.v1beta1.Fee")
	for _, coin := range fee {
		if coin.Denom == "" || coin.Amount == "" {
			return nil, errors.New("fee denom and amount required")
		}
		for _, r := range coin.Amount {
			if r < '0' || r > '9' {
				return nil, errors.New("invalid fee amount")
			}
		}
		c := message("cosmos.base.v1beta1.Coin")
		set(c, "denom", protoreflect.ValueOfString(coin.Denom))
		set(c, "amount", protoreflect.ValueOfString(coin.Amount))
		list(f, "amount").Append(protoreflect.ValueOfMessage(c))
	}
	set(f, "gas_limit", protoreflect.ValueOfUint64(gas))
	set(m, "fee", protoreflect.ValueOfMessage(f))
	return Marshal(m)
}
func EncodeSignDoc(body, auth []byte, chainID string, accountNumber uint64) ([]byte, error) {
	if chainID == "" {
		return nil, errors.New("string Cosmos chain ID required")
	}
	if _, e := Decode("cosmos.tx.v1beta1.TxBody", body); e != nil {
		return nil, e
	}
	if _, e := Decode("cosmos.tx.v1beta1.AuthInfo", auth); e != nil {
		return nil, e
	}
	m := message("cosmos.tx.v1beta1.SignDoc")
	set(m, "body_bytes", protoreflect.ValueOfBytes(body))
	set(m, "auth_info_bytes", protoreflect.ValueOfBytes(auth))
	set(m, "chain_id", protoreflect.ValueOfString(chainID))
	set(m, "account_number", protoreflect.ValueOfUint64(accountNumber))
	return Marshal(m)
}
func EncodeTxRaw(body, auth []byte, signatures [][]byte) ([]byte, error) {
	if len(signatures) == 0 {
		return nil, errors.New("signatures required")
	}
	a, e := Decode("cosmos.tx.v1beta1.AuthInfo", auth)
	if e != nil {
		return nil, e
	}
	if list(a, "signer_infos").Len() != len(signatures) {
		return nil, errors.New("signature/signer count mismatch")
	}
	if _, e = Decode("cosmos.tx.v1beta1.TxBody", body); e != nil {
		return nil, e
	}
	m := message("cosmos.tx.v1beta1.TxRaw")
	set(m, "body_bytes", protoreflect.ValueOfBytes(body))
	set(m, "auth_info_bytes", protoreflect.ValueOfBytes(auth))
	for _, s := range signatures {
		if len(s) != 65 {
			return nil, errors.New("Ethermint signature requires65 bytes")
		}
		list(m, "signatures").Append(protoreflect.ValueOfBytes(append([]byte(nil), s...)))
	}
	return Marshal(m)
}
func TransactionHash(raw []byte) [32]byte { return sha256.Sum256(raw) }

// ValidateTxRaw checks the signed envelope shape before transport. It does not
// replace Msg business validation or cryptographic verification against SignDoc.
func ValidateTxRaw(raw []byte) error {
	tx, e := Decode("cosmos.tx.v1beta1.TxRaw", raw)
	if e != nil {
		return e
	}
	body, e := Decode("cosmos.tx.v1beta1.TxBody", tx.Get(tx.Descriptor().Fields().ByName("body_bytes")).Bytes())
	if e != nil {
		return e
	}
	if list(body, "messages").Len() == 0 {
		return errors.New("transaction messages required")
	}
	auth, e := Decode("cosmos.tx.v1beta1.AuthInfo", tx.Get(tx.Descriptor().Fields().ByName("auth_info_bytes")).Bytes())
	if e != nil {
		return e
	}
	sigs := list(tx, "signatures")
	if sigs.Len() == 0 || sigs.Len() != list(auth, "signer_infos").Len() {
		return errors.New("signature/signer count mismatch")
	}
	for i := 0; i < sigs.Len(); i++ {
		b := sigs.Get(i).Bytes()
		if len(b) != 65 || b[64] > 1 {
			return errors.New("invalid Ethermint signature shape")
		}
	}
	return nil
}
