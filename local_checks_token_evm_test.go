package decimal

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// These are interoperability tests for two exact historical official artifacts,
// not current deployment acceptance. Check-only signatures stay inside tests.
type localTokenChecks struct {
	e               *localEVM
	token           *Token
	checks          *Checks
	parsed          abi.ABI
	owner, receiver common.Address
	secrets         []*KeySigner
	supply          *big.Int
}

func newLocalTokenChecks(t *testing.T) *localTokenChecks {
	t.Helper()
	e := newLocalEVM(t)
	owner := common.HexToAddress("0x1101")
	receiver := common.HexToAddress("0x1102")
	tokenArtifact := loadLocalArtifact(t, "NewToken.compiled.json")
	checksArtifact := loadLocalArtifact(t, "Checks.official.json")
	tokenAddress := e.deploy(t, owner, tokenArtifact.ABI, common.FromHex(tokenArtifact.Bytecode), "Historical token", "HIS", big.NewInt(1))
	checksAddress := e.deploy(t, owner, checksArtifact.ABI, common.FromHex(checksArtifact.Bytecode))
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
	s := &localTokenChecks{e: e, token: token, checks: checks, parsed: parsed, owner: owner, receiver: receiver, supply: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)}
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
func (s *localTokenChecks) balance(t *testing.T, address common.Address, want *big.Int) {
	t.Helper()
	r, err := s.token.ReadBalanceOf(context.Background(), address)
	if err != nil || r.Value0.Cmp(want) != 0 {
		t.Fatalf("token balance %s got %v error %v want %v", address, r, err, want)
	}
}
func (s *localTokenChecks) allowance(t *testing.T, want *big.Int) {
	t.Helper()
	r, err := s.token.ReadAllowance(context.Background(), s.owner, s.checks.Address)
	if err != nil || r.Value0.Cmp(want) != 0 {
		t.Fatal("allowance", r, err, want)
	}
}
func (s *localTokenChecks) approve(t *testing.T, amount int64) {
	t.Helper()
	call, err := s.token.BuildApprove(s.checks.Address, big.NewInt(amount))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.e.execute(t, s.owner, call); err != nil {
		t.Fatal(err)
	}
}
func (s *localTokenChecks) create(t *testing.T) Call {
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
func (s *localTokenChecks) signatures(t *testing.T, hashes [][32]byte) [][]byte {
	t.Helper()
	var out [][]byte
	for i, h := range hashes {
		sig, err := s.secrets[i].SignDigest(context.Background(), common.BytesToHash(accounts.TextHash(h[:])))
		if err != nil {
			t.Fatal(err)
		}
		sig[64] += 27
		out = append(out, sig)
	}
	return out
}

func TestLocalEVMHistoricalChecksTokenAllowanceAndFundingRollback(t *testing.T) {
	for _, scenario := range []string{"no-allowance", "partial-allowance", "insufficient-balance"} {
		t.Run(scenario, func(t *testing.T) {
			s := newLocalTokenChecks(t)
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
func TestLocalEVMHistoricalChecksTokenBatchRedemptionAtomicity(t *testing.T) {
	s := newLocalTokenChecks(t)
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
		t.Fatal("historical token redemption", err)
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
