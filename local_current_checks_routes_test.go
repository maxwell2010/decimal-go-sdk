package decimal

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"os"
)

// Execute the sampled current Checks code with the historical token fixture.
// No public deployment state or acceptance is inferred.
type currentRouteChecks struct {
	e               *localEVM
	token           *Token
	checks          *Checks
	parsed          abi.ABI
	owner, receiver common.Address
	secrets         []*KeySigner
	supply          *big.Int
}

func newCurrentRouteChecks(t *testing.T) *currentRouteChecks {
	t.Helper()
	e := newLocalEVM(t)
	owner := common.HexToAddress("0x1101")
	receiver := common.HexToAddress("0x1102")
	tokenArtifact := loadLocalArtifact(t, "NewToken.compiled.json")
	checksArtifact := loadLocalArtifact(t, "Checks.official.json")
	tokenAddress := e.deploy(t, owner, tokenArtifact.ABI, common.FromHex(tokenArtifact.Bytecode), "Historical token", "HIS", big.NewInt(1))
	raw, err := os.ReadFile("testdata/evm/checks-current-runtime.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct{ Address, Runtime string }
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	code := common.FromHex(snapshot.Runtime)
	if len(code) != 6318 || crypto.Keccak256Hash(code).Hex() != "0x78da094992fb84bf17517277dcfc9c67ab36ba535f8879b5fd5b5614ac397bf4" {
		t.Fatal("current runtime identity changed")
	}
	checksAddress := common.HexToAddress(snapshot.Address)
	e.state.SetCode(checksAddress, code, tracing.CodeChangeUnspecified)
	client := e.client(t)
	token, err := client.Token(tokenAddress)
	if err != nil {
		t.Fatal(err)
	}
	checks, err := client.Checks(checksAddress)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(checksArtifact.ABI)))
	if err != nil {
		t.Fatal(err)
	}
	s := &currentRouteChecks{e: e, token: token, checks: checks, parsed: parsed, owner: owner, receiver: receiver, supply: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)}
	for _, password := range []string{"public token Checks fixture A", "public token Checks fixture B"} {
		key, err := EVMCheckSignerFromPassword(password)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(key.Close)
		s.secrets = append(s.secrets, key)
	}
	return s
}
func (s *currentRouteChecks) balance(t *testing.T, address common.Address, want *big.Int) {
	t.Helper()
	r, err := s.token.ReadBalanceOf(context.Background(), address)
	if err != nil || r.Value0.Cmp(want) != 0 {
		t.Fatalf("token balance %s got %v error %v want %v", address, r, err, want)
	}
}
func (s *currentRouteChecks) allowance(t *testing.T, want *big.Int) {
	t.Helper()
	r, err := s.token.ReadAllowance(context.Background(), s.owner, s.checks.Address)
	if err != nil || r.Value0.Cmp(want) != 0 {
		t.Fatal("allowance", r, err, want)
	}
}
func (s *currentRouteChecks) approve(t *testing.T, amount int64) {
	t.Helper()
	call, err := s.token.BuildApprove(s.checks.Address, big.NewInt(amount))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.e.execute(t, s.owner, call); err != nil {
		t.Fatal(err)
	}
}
func (s *currentRouteChecks) create(t *testing.T) Call {
	t.Helper()
	nonce, err := s.checks.ReadFromAt(context.Background(), s.owner, nil, "nonces()")
	if err != nil {
		t.Fatal(err)
	}
	call, err := s.checks.BuildCreateChecksToken([]common.Address{s.secrets[0].Address(), s.secrets[1].Address()}, big.NewInt(100), big.NewInt(1000), nonce[0].(*big.Int), s.token.Address)
	if err != nil {
		t.Fatal(err)
	}
	return call
}
func (s *currentRouteChecks) signatures(t *testing.T, hashes [][32]byte) [][]byte {
	t.Helper()
	var out [][]byte
	for i, h := range hashes {
		sig, err := SignEVMCheckRedemption(context.Background(), s.secrets[i], common.Hash(h), s.receiver)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sig)
	}
	return out
}

func TestLocalEVMCurrentChecksTokenAllowanceAndFundingRollback(t *testing.T) {
	for _, scenario := range []string{"no-allowance", "partial-allowance", "insufficient-balance"} {
		t.Run(scenario, func(t *testing.T) {
			s := newCurrentRouteChecks(t)
			wantAllowance := int64(0)
			ownerBalance := new(big.Int).Set(s.supply)
			if scenario == "partial-allowance" {
				wantAllowance = 199
				s.approve(t, 199)
			}
			if scenario == "insufficient-balance" {
				wantAllowance = 200
				s.approve(t, 200)
				burn, err := s.token.BuildBurn(new(big.Int).Sub(s.supply, big.NewInt(150)))
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err = s.e.execute(t, s.owner, burn); err != nil {
					t.Fatal(err)
				}
				ownerBalance.SetInt64(150)
			}
			beforeNonce, err := s.checks.ReadFromAt(context.Background(), s.owner, nil, "nonces()")
			if err != nil {
				t.Fatal(err)
			}
			_, logs, err := s.e.execute(t, s.owner, s.create(t))
			if err == nil || len(logs) != 0 {
				t.Fatal("invalid funding accepted or rollback retained logs", err)
			}
			s.balance(t, s.owner, ownerBalance)
			s.balance(t, s.checks.Address, big.NewInt(0))
			s.balance(t, s.receiver, big.NewInt(0))
			s.allowance(t, big.NewInt(wantAllowance))
			afterNonce, err := s.checks.ReadFromAt(context.Background(), s.owner, nil, "nonces()")
			if err != nil || beforeNonce[0].(*big.Int).Cmp(afterNonce[0].(*big.Int)) != 0 {
				t.Fatal("failed creation changed caller nonce", beforeNonce, afterNonce, err)
			}
		})
	}
}
func TestLocalEVMCurrentChecksTokenBatchRedemptionAtomicity(t *testing.T) {
	s := newCurrentRouteChecks(t)
	s.approve(t, 200)
	data, logs, err := s.e.execute(t, s.owner, s.create(t))
	if err != nil {
		t.Fatal("create token Checks", err)
	}
	values, err := s.parsed.Unpack("createChecksToken", data)
	if err != nil {
		t.Fatal(err)
	}
	hashes := values[0].([][32]byte)
	if len(hashes) != 2 || hashes[0] == hashes[1] {
		t.Fatal("invalid returned batch", hashes)
	}
	found := map[[32]byte]bool{}
	for _, log := range logs {
		if log.Address == s.checks.Address && len(log.Topics) > 0 && log.Topics[0] == s.parsed.Events["CreatedCheck"].ID {
			fields, err := s.parsed.Events["CreatedCheck"].Inputs.Unpack(log.Data)
			if err != nil {
				t.Fatal(err)
			}
			found[fields[0].([32]byte)] = true
		}
	}
	if len(found) != 2 || !found[hashes[0]] || !found[hashes[1]] {
		t.Fatal("creation events do not match return values")
	}
	s.balance(t, s.checks.Address, big.NewInt(200))
	s.balance(t, s.owner, new(big.Int).Sub(s.supply, big.NewInt(200)))
	s.allowance(t, big.NewInt(0))
	signatures := s.signatures(t, hashes)
	bad := [][]byte{append([]byte(nil), signatures[0]...), append([]byte(nil), signatures[1]...)}
	bad[1][0] ^= 1
	redeem, err := s.checks.BuildRedeemChecks(bad, hashes)
	if err != nil {
		t.Fatal(err)
	}
	_, logs, err = s.e.execute(t, s.receiver, redeem)
	if err == nil || len(logs) != 0 {
		t.Fatal("invalid second proof did not revert whole batch", err)
	}
	s.balance(t, s.checks.Address, big.NewInt(200))
	s.balance(t, s.receiver, big.NewInt(0))
	redeem, err = s.checks.BuildRedeemChecks(signatures, hashes)
	if err != nil {
		t.Fatal(err)
	}
	_, logs, err = s.e.execute(t, s.receiver, redeem)
	if err != nil {
		t.Fatal("current token redemption", err)
	}
	redeemed := map[[32]byte]bool{}
	for _, log := range logs {
		if log.Address == s.checks.Address && len(log.Topics) > 0 && log.Topics[0] == s.parsed.Events["RedeemCheck"].ID {
			fields, err := s.parsed.Events["RedeemCheck"].Inputs.Unpack(log.Data)
			if err != nil {
				t.Fatal(err)
			}
			if fields[1].(uint8) != 1 {
				t.Fatal("wrong redeemed status")
			}
			redeemed[fields[0].([32]byte)] = true
		}
	}
	if len(redeemed) != 2 || !redeemed[hashes[0]] || !redeemed[hashes[1]] {
		t.Fatal("missing redemption events")
	}
	s.balance(t, s.receiver, big.NewInt(200))
	s.balance(t, s.checks.Address, big.NewInt(0))
	s.allowance(t, big.NewInt(0))
	_, logs, err = s.e.execute(t, s.receiver, redeem)
	if err == nil || len(logs) != 0 {
		t.Fatal("replayed token redemption accepted", err)
	}
	s.balance(t, s.receiver, big.NewInt(200))
}

// This combines sampled Decimal Checks code with an official upstream STANDARD
// EIP2612 token. It does not certify any Decimal token deployment's permit.
func TestLocalEVMCurrentChecksTokenByStandardPermit(t *testing.T) {
	for _, scenario := range []string{"valid-and-replay", "expired", "wrong-value"} {
		t.Run(scenario, func(t *testing.T) {
			s := newCurrentRouteChecks(t)
			owner := testSigner(t, 1)
			defer owner.Close()
			s.owner = owner.Address()
			s.supply = big.NewInt(1000)
			artifact := loadLocalArtifact(t, "permit-compiled.json")
			address := s.e.deploy(t, s.owner, artifact.ABI, common.FromHex(artifact.Bytecode), "StandardPermit", "SPT", s.owner, s.supply)
			var err error
			s.token, err = s.e.client(t).Token(address)
			if err != nil {
				t.Fatal(err)
			}
			deadline := big.NewInt(100)
			value := big.NewInt(200)
			if scenario == "expired" {
				deadline = big.NewInt(0)
			}
			if scenario == "wrong-value" {
				value = big.NewInt(199)
			}
			permit := Permit{Owner: s.owner, Spender: s.checks.Address, Value: value, Nonce: big.NewInt(0), Deadline: deadline}
			signature, err := s.e.client(t).CheckedPermit(context.Background(), owner, address, "StandardPermit", "1", permit)
			if err != nil {
				t.Fatal(err)
			}
			before, err := s.checks.ReadFromAt(context.Background(), s.owner, nil, "nonces()")
			if err != nil {
				t.Fatal(err)
			}
			create, err := s.checks.BuildCreateChecksTokenByPermit([]common.Address{s.secrets[0].Address(), s.secrets[1].Address()}, big.NewInt(100), big.NewInt(1000), before[0].(*big.Int), address, deadline, signature.V, signature.R, signature.S)
			if err != nil {
				t.Fatal(err)
			}
			output, logs, err := s.e.execute(t, s.owner, create)
			if scenario != "valid-and-replay" {
				if err == nil || len(logs) != 0 {
					t.Fatal("invalid permit issuance accepted", err)
				}
				s.balance(t, s.owner, big.NewInt(1000))
				s.balance(t, s.checks.Address, big.NewInt(0))
				s.allowance(t, big.NewInt(0))
				tokenNonce, readErr := s.token.ReadNonces(context.Background(), s.owner)
				if readErr != nil || tokenNonce.Value0.Sign() != 0 {
					t.Fatal("failed permit changed nonce", tokenNonce, readErr)
				}
				after, readErr := s.checks.ReadFromAt(context.Background(), s.owner, nil, "nonces()")
				if readErr != nil || after[0].(*big.Int).Cmp(before[0].(*big.Int)) != 0 {
					t.Fatal("failed permit changed Checks nonce", after, readErr)
				}
				return
			}
			if err != nil {
				t.Fatal("valid permit issuance failed", err)
			}
			values, err := s.parsed.Unpack("createChecksTokenByPermit", output)
			if err != nil {
				t.Fatal(err)
			}
			hashes := values[0].([][32]byte)
			if len(hashes) != 2 || hashes[0] == hashes[1] {
				t.Fatal("permit issuance hashes", hashes)
			}
			created := map[[32]byte]bool{}
			for _, log := range logs {
				if log.Address == s.checks.Address && len(log.Topics) > 0 && log.Topics[0] == s.parsed.Events["CreatedCheck"].ID {
					fields, err := s.parsed.Events["CreatedCheck"].Inputs.Unpack(log.Data)
					if err != nil {
						t.Fatal(err)
					}
					created[fields[0].([32]byte)] = true
				}
			}
			if len(created) != 2 || !created[hashes[0]] || !created[hashes[1]] {
				t.Fatal("permit creation events differ from output")
			}
			s.balance(t, s.owner, big.NewInt(800))
			s.balance(t, s.checks.Address, big.NewInt(200))
			s.allowance(t, big.NewInt(0))
			tokenNonce, err := s.token.ReadNonces(context.Background(), s.owner)
			if err != nil || tokenNonce.Value0.Cmp(big.NewInt(1)) != 0 {
				t.Fatal("permit nonce not consumed", tokenNonce, err)
			}
			if _, logs, err = s.e.execute(t, s.owner, create); err == nil || len(logs) != 0 {
				t.Fatal("permit issuance replay accepted", err)
			}
			s.balance(t, s.owner, big.NewInt(800))
			s.balance(t, s.checks.Address, big.NewInt(200))
			redeem, err := s.checks.BuildRedeemChecks(s.signatures(t, hashes), hashes)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = s.e.execute(t, s.receiver, redeem); err != nil {
				t.Fatal("current helper could not redeem permit-funded Checks", err)
			}
			s.balance(t, s.receiver, big.NewInt(200))
			s.balance(t, s.checks.Address, big.NewInt(0))
		})
	}
}

func TestLocalEVMCurrentChecksExpiryRejectsRecipientProof(t *testing.T) {
	s := newCurrentRouteChecks(t)
	s.e.fund(s.owner, big.NewInt(1000))
	nonce, err := s.checks.ReadFromAt(context.Background(), s.owner, nil, "nonces()")
	if err != nil {
		t.Fatal(err)
	}
	create, err := s.checks.BuildCreateChecksDEL(big.NewInt(100), []common.Address{s.secrets[0].Address()}, big.NewInt(100), big.NewInt(1000), nonce[0].(*big.Int))
	if err != nil {
		t.Fatal(err)
	}
	output, _, err := s.e.execute(t, s.owner, create)
	if err != nil {
		t.Fatal(err)
	}
	values, err := s.parsed.Unpack("createChecksDEL", output)
	if err != nil {
		t.Fatal(err)
	}
	hashes := values[0].([][32]byte)
	if len(hashes) != 1 {
		t.Fatal("creation hashes", hashes)
	}
	proof, err := SignEVMCheckRedemption(context.Background(), s.secrets[0], common.Hash(hashes[0]), s.receiver)
	if err != nil {
		t.Fatal(err)
	}
	redeem, err := s.checks.BuildRedeemChecks([][]byte{proof}, hashes)
	if err != nil {
		t.Fatal(err)
	}
	s.e.block(1001)
	result, logs, err := s.e.execute(t, s.receiver, redeem)
	expected := s.parsed.Errors["ExpiredDueBlock"].ID
	if err == nil || len(logs) != 0 || len(result) != 4 || string(result) != string(expected[:4]) {
		t.Fatalf("expiry rejected incorrectly: %v %x", err, result)
	}
	if s.e.balance(s.checks.Address).Cmp(big.NewInt(100)) != 0 || s.e.balance(s.receiver).Sign() != 0 {
		t.Fatal("expiry changed custody")
	}
}
