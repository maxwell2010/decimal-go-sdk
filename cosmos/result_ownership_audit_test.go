package cosmos

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestSigningCallbackCancellationDominatesFailure(t *testing.T) {
	pub, err := hex.DecodeString("0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798")
	if err != nil {
		t.Fatal(err)
	}
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		called := false
		result, err := SignDirectTx(ctx, DirectTx{Body: []byte{}, ChainID: "offline", SignerOrder: [][]byte{pub}, Signers: []DirectSigner{{PublicKey: pub, Sign: func(context.Context, [32]byte) ([]byte, error) {
			called = true
			if cancelled {
				cancel()
			}
			return nil, errors.New("private callback diagnostic")
		}}}})
		cancel()
		if !called || err == nil || len(result.Raw) != 0 || len(result.Body) != 0 || len(result.AuthInfo) != 0 {
			t.Fatal("callback failure leaked a partial transaction", err)
		}
		if cancelled != errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "private callback diagnostic") {
			t.Fatal("cancellation priority or redaction lost", err)
		}
	}
}

type retainingReplyAuditConn struct {
	grpc.ClientConnInterface
	reply *dynamicpb.Message
}

func (c *retainingReplyAuditConn) Invoke(_ context.Context, _ string, _ any, reply any, _ ...grpc.CallOption) error {
	c.reply = reply.(*dynamicpb.Message)
	tx := c.reply.Mutable(c.reply.Descriptor().Fields().ByName("tx_response")).Message()
	tx.Set(tx.Descriptor().Fields().ByName("height"), protoreflect.ValueOfInt64(7))
	return nil
}

func TestGRPCQueryOwnsBackendReply(t *testing.T) {
	conn := &retainingReplyAuditConn{}
	g, err := NewGRPC(conn)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewQueryRequest("cosmos.tx.v1beta1.Service", "GetTx")
	if err != nil {
		t.Fatal(err)
	}
	result, err := g.Query(context.Background(), "cosmos.tx.v1beta1.Service", "GetTx", request)
	if err != nil {
		t.Fatal(err)
	}
	backendTx := conn.reply.Mutable(conn.reply.Descriptor().Fields().ByName("tx_response")).Message()
	backendTx.Set(backendTx.Descriptor().Fields().ByName("height"), protoreflect.ValueOfInt64(99))
	resultTx := result.Get(result.Descriptor().Fields().ByName("tx_response")).Message()
	if resultTx.Get(resultTx.Descriptor().Fields().ByName("height")).Int() != 7 {
		t.Fatal("backend mutation changed returned query")
	}
	resultTx.Set(resultTx.Descriptor().Fields().ByName("height"), protoreflect.ValueOfInt64(42))
	if backendTx.Get(backendTx.Descriptor().Fields().ByName("height")).Int() != 99 {
		t.Fatal("caller mutation changed backend reply")
	}
}
