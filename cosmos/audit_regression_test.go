package cosmos

import (
	"context"
	"encoding/hex"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"net/http"
	"net/http/httptest"
	"testing"
)

type auditConn struct {
	calls        int
	hash, height string
}

func (c *auditConn) Invoke(_ context.Context, _ string, _, out any, _ ...grpc.CallOption) error {
	c.calls++
	response := out.(*dynamicpb.Message)
	tx := response.Mutable(response.Descriptor().Fields().ByName("tx_response")).Message()
	tx.Set(tx.Descriptor().Fields().ByName("txhash"), protoreflect.ValueOfString(c.hash))
	var height int64
	fmt.Sscan(c.height, &height)
	tx.Set(tx.Descriptor().Fields().ByName("height"), protoreflect.ValueOfInt64(height))
	return nil
}
func (*auditConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	panic("unexpected stream")
}
func TestGRPCRejectsForeignDescriptorBeforeTransport(t *testing.T) {
	// Same full name with an incompatible field wire type must not pass the guard.
	f, e := protodesc.NewFile(&descriptorpb.FileDescriptorProto{Name: proto.String("foreign.proto"), Package: proto.String("cosmos.auth.v1beta1"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("QueryAccountRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("address"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum()}}}}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	conn := &auditConn{}
	g, _ := NewGRPC(conn)
	_, e = g.Query(context.Background(), "cosmos.auth.v1beta1.Query", "Account", dynamicpb.NewMessage(f.Messages().Get(0)))
	if e == nil || conn.calls != 0 {
		t.Fatal("foreign schema reached transport", e)
	}
}
func TestConfirmationRequiresCommittedHeight(t *testing.T) {
	hash := TransactionHash([]byte("fixture"))
	encoded := hex.EncodeToString(hash[:])
	for _, height := range []string{"", "0", "-1", "9223372036854775808", "abc", "7"} {
		t.Run(height, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"tx_response":{"txhash":%q,"height":%q,"code":0}}`, encoded, height)
			}))
			defer server.Close()
			rest, _ := NewREST(server.URL, nil)
			_, e := rest.Transaction(context.Background(), hash)
			if (e == nil) != (height == "7") {
				t.Fatal("REST height acceptance", e)
			}
			if height == "" || height == "0" || height == "-1" || height == "7" {
				g, _ := NewGRPC(&auditConn{hash: encoded, height: height})
				_, e = g.Transaction(context.Background(), hash)
				if (e == nil) != (height == "7") {
					t.Fatal("gRPC height acceptance", e)
				}
			}
		})
	}
}
