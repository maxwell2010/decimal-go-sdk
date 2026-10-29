package decimal

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

type contractReadAuditBackend struct {
	*fakeRPC
	read func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

func (b *contractReadAuditBackend) CallContract(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	return b.read(ctx, msg, block)
}
func TestContractReadPreservesCancellationOnBackendResponse(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful-response", true: "error-response"}[failure], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			backend := &contractReadAuditBackend{fakeRPC: newFake(), read: func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
				cancel()
				if failure {
					return nil, errors.New("synthetic credential must not leak")
				}
				return make([]byte, 32), nil
			}}
			client, err := NewClient(Testnet(), backend)
			if err != nil {
				t.Fatal(err)
			}
			token, err := client.Token(testAddress)
			if err != nil {
				t.Fatal(err)
			}
			if result, err := token.Read(ctx, "totalSupply()"); !errors.Is(err, context.Canceled) || result != nil {
				t.Fatal("cancellation lost", result, err)
			}
		})
	}
}
func TestContractReadOwnsRequestIdentityAndDynamicResult(t *testing.T) {
	backend := &contractReadAuditBackend{fakeRPC: newFake()}
	client, err := NewClient(Testnet(), backend)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := client.ContractWithABI(testAddress, []byte(`[{"type":"function","name":"inspect","stateMutability":"view","inputs":[],"outputs":[{"type":"bytes"},{"type":"string"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := contract.contractABI()
	if err != nil {
		t.Fatal(err)
	}
	response, err := parsed.Methods["inspect"].Outputs.Pack([]byte{1, 2, 3}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	block := big.NewInt(42)
	caller := common.HexToAddress("0x1234")
	backend.read = func(_ context.Context, msg ethereum.CallMsg, number *big.Int) ([]byte, error) {
		if msg.From != caller || msg.To == nil || *msg.To != testAddress || number == nil || number.Int64() != 42 {
			t.Fatal("request identity changed", msg, number)
		}
		number.SetInt64(99)
		*msg.To = common.HexToAddress("0x9999")
		return response, nil
	}
	values, err := contract.ReadFromAt(context.Background(), caller, block, "inspect()")
	if err != nil {
		t.Fatal(err)
	}
	if block.Int64() != 42 || contract.Address != testAddress {
		t.Fatal("backend mutated caller identity", block, contract.Address)
	}
	for i := range response {
		response[i] = 0
	}
	if string(values[0].([]byte)) != "\x01\x02\x03" || values[1].(string) != "fixture" {
		t.Fatal("result borrowed backend memory", values)
	}
}
func TestContractReadRejectsNilReceiverAndContext(t *testing.T) {
	for _, contract := range []*Contract{nil, {}} {
		if _, err := contract.ReadFromAt(context.Background(), common.Address{}, nil, "totalSupply()"); err == nil {
			t.Fatal("unconfigured contract accepted")
		}
		if _, err := contract.ReadFromAt(nil, common.Address{}, nil, "totalSupply()"); err == nil {
			t.Fatal("nil context accepted")
		}
	}
}

func TestContractReadDynamicBytesDoNotAliasBackendBuffer(t *testing.T) {
	backend := &contractReadAuditBackend{fakeRPC: newFake()}
	client, err := NewClient(Testnet(), backend)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := client.ContractWithABI(testAddress, []byte(`[{"type":"function","name":"payload","stateMutability":"view","inputs":[],"outputs":[{"type":"bytes"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := contract.contractABI()
	if err != nil {
		t.Fatal(err)
	}
	response, err := parsed.Methods["payload"].Outputs.Pack([]byte{0x5a, 0x5b})
	if err != nil {
		t.Fatal(err)
	}
	backend.read = func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) { return response, nil }
	values, err := contract.Read(context.Background(), "payload()")
	if err != nil {
		t.Fatal(err)
	}
	for i := range response {
		response[i] = 0
	}
	if string(values[0].([]byte)) != "\x5a\x5b" {
		t.Fatal("backend response was borrowed", values)
	}
}
