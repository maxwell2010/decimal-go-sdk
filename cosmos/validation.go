package cosmos

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/cosmos/btcutil/bech32"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"math/big"
	"reflect"
	"regexp"
	"strings"
)

var ErrUnsupportedValidation = errors.New("message has no pinned ValidateBasic policy")

type ValidationError struct{ Message, Field, Rule string }

func (e *ValidationError) Error() string {
	return "invalid " + e.Message + " field " + e.Field + ": " + e.Rule
}

var decimalDenom = regexp.MustCompile(`^[a-z][a-z0-9]{2,9}$`)

//go:embed schema/validation-denom-whitelist.json
var validationWhitelist []byte
var sdkDenom = pinnedAppDenomPattern()

func pinnedAppDenomPattern() *regexp.Regexp {
	var values []string
	if e := json.Unmarshal(validationWhitelist, &values); e != nil {
		panic(e)
	}
	for i, v := range values {
		values[i] = regexp.QuoteMeta(v)
	}
	// Exact pinned app SetCoinDenomRegex + SDK46 anchoring; do not silently repair
	// its alternation grouping while claiming compatibility.
	return regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9/:._-]{2,127}|` + strings.Join(values, "|") + `$`)
}

var collectionDenom = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,255}$`)

type validationMessage struct {
	m    protoreflect.Message
	name string
}

func (v validationMessage) field(s string) protoreflect.FieldDescriptor {
	return v.m.Descriptor().Fields().ByName(protoreflect.Name(s))
}
func (v validationMessage) str(s string) string { return v.m.Get(v.field(s)).String() }
func (v validationMessage) sub(s string) validationMessage {
	return validationMessage{v.m.Get(v.field(s)).Message(), v.name}
}
func (v validationMessage) list(s string) protoreflect.List { return v.m.Get(v.field(s)).List() }
func (v validationMessage) uint(s string) uint64            { return v.m.Get(v.field(s)).Uint() }
func (v validationMessage) integer(s string) (*big.Int, bool) {
	n, ok := new(big.Int).SetString(v.str(s), 10)
	return n, ok && n.BitLen() <= 256
}
func (v validationMessage) decimal(s string) (*big.Int, bool) {
	n, ok := new(big.Int).SetString(v.str(s), 10)
	return n, ok && n.BitLen() <= 315
}
func (v validationMessage) fail(field, rule string) error {
	return &ValidationError{v.name, field, rule}
}
func (v validationMessage) address(field, hrp string) error {
	if _, e := DecodeAddress(v.str(field), hrp); e != nil {
		return v.fail(field, "invalid address")
	}
	return nil
}
func (v validationMessage) positive(field string) error {
	n, ok := v.integer(field)
	if !ok || n.Sign() <= 0 {
		return v.fail(field, "positive integer required")
	}
	return nil
}
func (v validationMessage) coin(field string, pattern *regexp.Regexp, positive bool) error {
	coin := v.sub(field)
	if !pattern.MatchString(coin.str("denom")) {
		return v.fail(field, "invalid denomination")
	}
	n, ok := coin.integer("amount")
	if !ok || n.Sign() < 0 || (positive && n.Sign() == 0) {
		return v.fail(field, "invalid coin amount")
	}
	return nil
}
func (v validationMessage) unique(field string, nonempty bool) error {
	l := v.list(field)
	if nonempty && l.Len() == 0 {
		return v.fail(field, "empty list")
	}
	seen := map[uint64]bool{}
	for i := 0; i < l.Len(); i++ {
		n := l.Get(i).Uint()
		if seen[n] {
			return v.fail(field, "duplicate entry")
		}
		seen[n] = true
	}
	return nil
}

// ValidateBasic reproduces stateless valid-message predicates in pinned node
// Msg.ValidateBasic and SDK upgrade ValidateBasic. It does not authorize roles,
// check balances, chain state, ante admission, expiry or successful execution.
// Malformed custom integer strings fail rather than reproducing upstream nil panics.
func ValidateBasic(message proto.Message) error {
	if message == nil || (reflect.ValueOf(message).Kind() == reflect.Pointer && reflect.ValueOf(message).IsNil()) || !message.ProtoReflect().IsValid() {
		return errors.New("message required")
	}
	m := message.ProtoReflect()
	name := "/" + string(m.Descriptor().FullName())
	if _, ok := signerRules[name]; !ok {
		return ErrUnsupportedValidation
	}
	expected, e := NewMessage(name)
	if e != nil {
		return ErrUnsupportedValidation
	}
	if m.Descriptor() != expected.Descriptor() {
		return errors.New("message descriptor differs from pinned schema")
	}
	v := validationMessage{m, name}
	short := string(m.Descriptor().Name())
	pkg := string(m.Descriptor().ParentFile().Package())
	// Each request has exactly one source-backed signer field. This checks the
	// correct operator/account HRP; no role membership is inferred.
	rule := signerRules[name]
	if e = v.address(rule.field, rule.hrp); e != nil {
		return e
	}
	switch pkg {
	case "decimal.coin.v1":
		return validateCoinMessage(v, short)
	case "decimal.nft.v1":
		if short == "MsgMintToken" {
			if e = v.address("recipient", AccountHRP); e != nil {
				return e
			}
			if !collectionDenom.MatchString(v.str("denom")) || !collectionDenom.MatchString(v.str("token_id")) {
				return v.fail("denom/token_id", "invalid collection identifier")
			}
			if v.uint("quantity") == 0 {
				return v.fail("quantity", "positive quantity required")
			}
			return v.sub("reserve").positive("amount")
		}
		if strings.TrimSpace(v.str("token_id")) == "" {
			return v.fail("token_id", "empty identifier")
		}
		if short != "MsgUpdateToken" {
			if e = v.unique("sub_token_ids", false); e != nil {
				return e
			}
		}
		if short == "MsgUpdateReserve" {
			n, ok := v.sub("reserve").integer("amount")
			if !ok || n.Sign() == 0 {
				return v.fail("reserve", "nonzero amount required")
			}
		}
		if short == "MsgSendToken" {
			if e = v.address("recipient", AccountHRP); e != nil {
				return e
			}
			a, _ := DecodeAccountAddress(v.str("sender"))
			b, _ := DecodeAccountAddress(v.str("recipient"))
			if bytes.Equal(a, b) {
				return v.fail("recipient", "self transfer")
			}
		}
		return nil
	case "decimal.validator.v1":
		return validateValidatorMessage(v, short)
	case "decimal.legacy.v1":
		if len(v.m.Get(v.field("public_key")).Bytes()) != 33 {
			return v.fail("public_key", "33 byte public key required")
		}
		return nil
	case "decimal.fee.v1":
		l := v.list("prices")
		if l.Len() == 0 {
			return v.fail("prices", "empty price list")
		}
		type pricePair struct{ denom, quote string }
		seen := map[pricePair]bool{}
		for i := 0; i < l.Len(); i++ {
			p := validationMessage{l.Get(i).Message(), name}
			price, valid := p.decimal("price")
			if !valid || price.Sign() <= 0 {
				return v.fail("prices", "positive SDK decimal required")
			}
			key := pricePair{p.str("denom"), p.str("quote")}
			if seen[key] {
				return v.fail("prices", "duplicate pair")
			}
			seen[key] = true
		}
		return nil
	case "decimal.multisig.v1":
		switch short {
		case "MsgCreateWallet":
			owners, weights := v.list("owners"), v.list("weights")
			if owners.Len() < 2 || owners.Len() > 16 || owners.Len() != weights.Len() {
				return v.fail("owners/weights", "invalid count")
			}
			seen := map[string]bool{}
			var sum uint64
			for i := 0; i < owners.Len(); i++ {
				owner := owners.Get(i).String()
				if _, e = DecodeAccountAddress(owner); e != nil || seen[owner] {
					return v.fail("owners", "invalid or duplicate owner")
				}
				seen[owner] = true
				w := weights.Get(i).Uint()
				if w < 1 || w > 1024 {
					return v.fail("weights", "weight outside 1..1024")
				}
				sum += w
			}
			if sum < v.uint("threshold") {
				return v.fail("threshold", "exceeds weight sum")
			}
		case "MsgCreateTransaction":
			return v.address("wallet", AccountHRP)
		case "MsgSignTransaction":
			prefix, words, e := bech32.Decode(v.str("id"), 1023)
			if e == nil {
				_, e = bech32.ConvertBits(words, 5, 8, false)
			}
			if e != nil || prefix != "d0mstx" {
				return v.fail("id", "invalid multisig transaction identifier")
			}
		}
		return nil
	case "decimal.swap.v1":
		if short == "MsgActivateChain" || short == "MsgDeactivateChain" {
			if v.uint("id") == 0 {
				return v.fail("id", "zero chain number")
			}
			if short == "MsgActivateChain" && v.str("name") == "" {
				return v.fail("name", "empty chain name")
			}
			return nil
		}
		if e = v.positive("amount"); e != nil {
			return e
		}
		if v.uint("from_chain") == 0 || v.uint("dest_chain") == 0 || v.uint("from_chain") == v.uint("dest_chain") {
			return v.fail("chain", "invalid chain numbers")
		}
		if _, ok := v.integer("transaction_number"); !ok {
			return v.fail("transaction_number", "invalid integer")
		}
		if short == "MsgRedeemSwap" {
			if e = v.address("recipient", AccountHRP); e != nil {
				return e
			}
			for _, field := range []string{"r", "s"} {
				b, e := hex.DecodeString(v.str(field))
				if e != nil || len(b) != 32 {
					return v.fail(field, "32 byte hex required")
				}
			}
		}
		return nil
	case "cosmos.upgrade.v1beta1":
		if short == "MsgCancelUpgrade" {
			return nil
		}
		plan := v.sub("plan")
		if plan.str("name") == "" || plan.m.Get(plan.field("height")).Int() <= 0 {
			return v.fail("plan", "name and positive height required")
		}
		if plan.m.Has(plan.field("upgraded_client_state")) {
			return v.fail("plan", "IBC client state deprecated")
		}
		if plan.m.Has(plan.field("time")) {
			timestamp := plan.sub("time")
			if timestamp.m.Get(timestamp.field("seconds")).Int() != -62135596800 || timestamp.m.Get(timestamp.field("nanos")).Int() != 0 {
				return v.fail("plan.time", "time-based upgrade deprecated")
			}
		}
		return nil
	}
	return ErrUnsupportedValidation
}

func validateCoinMessage(v validationMessage, name string) error {
	switch name {
	case "MsgCreateCoin", "MsgUpdateCoin":
		if !decimalDenom.MatchString(v.str("denom")) {
			return v.fail("denom", "invalid denomination")
		}
		limit, ok := v.integer("limit_volume")
		max := new(big.Int).Exp(big.NewInt(10), big.NewInt(33), nil)
		min := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
		if !ok || limit.Cmp(max) > 0 {
			return v.fail("limit_volume", "exceeds maximum")
		}
		if s := v.str("min_volume"); s != "" && s != "0" {
			n, ok := v.integer("min_volume")
			if !ok || n.Sign() != 0 && (n.Cmp(min) < 0 || n.Cmp(max) > 0 || n.Cmp(limit) > 0) {
				return v.fail("min_volume", "invalid minimum emission")
			}
		}
		if name == "MsgCreateCoin" {
			if len(v.str("title")) > 64 {
				return v.fail("title", "title exceeds 64 bytes")
			}
			crr := v.uint("crr")
			if crr < 10 || crr > 100 {
				return v.fail("crr", "outside 10..100")
			}
			initial, ok := v.integer("initial_volume")
			if !ok || initial.Cmp(min) < 0 || initial.Cmp(max) > 0 || initial.Cmp(limit) > 0 {
				return v.fail("initial_volume", "invalid supply")
			}
			reserve, ok := v.integer("initial_reserve")
			if !ok || reserve.Cmp(min) < 0 {
				return v.fail("initial_reserve", "below reserve minimum")
			}
		}
		return nil
	case "MsgRedeemCheck":
		return nil // Upstream only checks sender, then TODO.
	case "MsgSendCoin":
		if e := v.address("recipient", AccountHRP); e != nil {
			return e
		}
		if v.str("sender") == v.str("recipient") {
			return v.fail("recipient", "self transfer")
		}
		return v.coin("coin", decimalDenom, true)
	case "MsgBurnCoin":
		return v.coin("coin", decimalDenom, true)
	case "MsgMultiSendCoin":
		l := v.list("sends")
		if l.Len() == 0 {
			return v.fail("sends", "empty send list")
		}
		for i := 0; i < l.Len(); i++ {
			send := validationMessage{l.Get(i).Message(), v.name}
			if e := send.address("recipient", AccountHRP); e != nil {
				return e
			}
			if v.str("sender") == send.str("recipient") {
				return v.fail("recipient", "self transfer")
			}
			if e := send.coin("coin", decimalDenom, true); e != nil {
				return e
			}
		}
		return nil
	case "MsgBuyCoin":
		if v.sub("coin_to_buy").str("denom") == v.sub("max_coin_to_sell").str("denom") {
			return v.fail("denom", "same coin")
		}
		if e := v.coin("coin_to_buy", decimalDenom, true); e != nil {
			return e
		}
		return v.coin("max_coin_to_sell", decimalDenom, true)
	case "MsgSellCoin":
		if v.sub("coin_to_sell").str("denom") == v.sub("min_coin_to_buy").str("denom") {
			return v.fail("denom", "same coin")
		}
		if e := v.coin("coin_to_sell", decimalDenom, true); e != nil {
			return e
		}
		if !decimalDenom.MatchString(v.sub("min_coin_to_buy").str("denom")) {
			return v.fail("min_coin_to_buy", "invalid denomination")
		}
		return nil
	case "MsgSellAllCoin":
		if !decimalDenom.MatchString(v.str("coin_denom_to_sell")) || v.str("coin_denom_to_sell") == v.sub("min_coin_to_buy").str("denom") {
			return v.fail("coin_denom_to_sell", "invalid or same coin")
		}
		return v.coin("min_coin_to_buy", decimalDenom, true)
	}
	return ErrUnsupportedValidation
}
func validateValidatorMessage(v validationMessage, name string) error {
	if name == "MsgHaltChain" {
		if v.m.Get(v.field("height")).Int() < 0 {
			return v.fail("height", "negative halt height")
		}
		return nil
	}
	if name == "MsgResumeChain" || name == "MsgFreezeChain" || name == "MsgUnfreezeChain" || name == "MsgSetOnline" || name == "MsgSetOffline" {
		return nil
	}
	if name == "MsgCreateValidator" || name == "MsgEditValidator" {
		if e := v.address("reward_address", AccountHRP); e != nil {
			return e
		}
		desc := v.sub("description")
		nonempty := false
		desc.m.Range(func(_ protoreflect.FieldDescriptor, _ protoreflect.Value) bool { nonempty = true; return false })
		if !nonempty {
			return v.fail("description", "empty description")
		}
		if name == "MsgCreateValidator" {
			if len(v.sub("consensus_pubkey").m.Get(v.sub("consensus_pubkey").field("value")).Bytes()) == 0 {
				return v.fail("consensus_pubkey", "empty public key")
			}
			n, ok := v.decimal("commission")
			if !ok || n.Sign() < 0 {
				return v.fail("commission", "invalid commission")
			}
			return v.coin("stake", sdkDenom, true)
		}
		return nil
	}
	if strings.Contains(name, "Redelegation") || strings.Contains(name, "Redelegate") {
		for _, f := range []string{"validator_src", "validator_dst"} {
			if e := v.address(f, ValidatorHRP); e != nil {
				return e
			}
		}
		if !strings.HasPrefix(name, "MsgCancel") && v.str("validator_src") == v.str("validator_dst") {
			return v.fail("validator_dst", "self redelegation")
		}
	} else if e := v.address("validator", ValidatorHRP); e != nil {
		return e
	}
	if strings.HasPrefix(name, "MsgCancel") && v.m.Get(v.field("creation_height")).Int() <= 0 {
		return v.fail("creation_height", "positive height required")
	}
	if strings.HasSuffix(name, "NFT") {
		if v.str("token_id") == "" {
			return v.fail("token_id", "empty identifier")
		}
		return v.unique("sub_token_ids", true)
	}
	return v.coin("coin", sdkDenom, true)
}
