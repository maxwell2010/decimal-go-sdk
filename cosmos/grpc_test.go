package cosmos

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGRPCWireSimulationAndExplicitSubmission(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	var submissions atomic.Int32
	var reject atomic.Bool
	server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		path, _ := grpc.MethodFromServerStream(stream)
		parts := strings.Split(path, "/")
		method, e := grpcMethod(parts[1], parts[2])
		if e != nil {
			return e
		}
		request := dynamicpb.NewMessage(method.Input())
		if e = stream.RecvMsg(request); e != nil {
			return e
		}
		response := dynamicpb.NewMessage(method.Output())
		switch parts[2] {
		case "Simulate":
			gas := response.Mutable(response.Descriptor().Fields().ByName("gas_info")).Message()
			gas.Set(gas.Descriptor().Fields().ByName("gas_wanted"), protoreflect.ValueOfUint64(42))
			gas.Set(gas.Descriptor().Fields().ByName("gas_used"), protoreflect.ValueOfUint64(21))
		case "BroadcastTx":
			submissions.Add(1)
			if reject.Load() {
				return status.Error(codes.Unavailable, "private credential error")
			}
			if request.Get(request.Descriptor().Fields().ByName("mode")).Enum() != 2 {
				t.Error("not SYNC mode")
			}
			raw := request.Get(request.Descriptor().Fields().ByName("tx_bytes")).Bytes()
			hash := TransactionHash(raw)
			tx := response.Mutable(response.Descriptor().Fields().ByName("tx_response")).Message()
			tx.Set(tx.Descriptor().Fields().ByName("txhash"), protoreflect.ValueOfString(hex.EncodeToString(hash[:])))
		default:
			return status.Error(codes.Unimplemented, "fixture method")
		}
		return stream.SendMsg(response)
	}))
	go server.Serve(listener)
	defer server.Stop()
	conn, e := grpc.NewClient("passthrough:///fixture", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDisableRetry())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	client, _ := NewGRPC(conn)
	var vector map[string]string
	b, e := os.ReadFile("schema/reference-tx.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &vector); e != nil {
		t.Fatal(e)
	}
	raw, e := hex.DecodeString(vector["tx_raw"])
	if e != nil {
		t.Fatal(e)
	}
	gas, e := client.Simulate(context.Background(), raw)
	if e != nil || gas.Used != "21" || gas.Wanted != "42" {
		t.Fatal("wire simulation", e)
	}
	request := message("cosmos.tx.v1beta1.BroadcastTxRequest")
	if _, e = client.Query(context.Background(), "cosmos.tx.v1beta1.Service", "BroadcastTx", request); e == nil || submissions.Load() != 0 {
		t.Fatal("query invoked write")
	}
	result, e := client.Broadcast(context.Background(), raw)
	if e != nil || result.Code != 0 || submissions.Load() != 1 {
		t.Fatal("explicit submission fixture", e)
	}
	reject.Store(true)
	_, e = client.Broadcast(context.Background(), raw)
	var unknown *SubmissionUnknownError
	if !errors.As(e, &unknown) || submissions.Load() != 2 || strings.Contains(e.Error(), "credential") {
		t.Fatal("ambiguous submission retried/leaked", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = client.Simulate(ctx, raw); !errors.Is(e, context.Canceled) {
		t.Fatal("context lost", e)
	}
}

func TestGRPCSchemaGuards(t *testing.T) {
	if _, e := NewQueryRequest("decimal.coin.v1.Msg", "SendCoin"); e == nil {
		t.Fatal("Msg service allowed as query")
	}
	if _, e := NewQueryRequest("cosmos.auth.v1beta1.Query", "Account"); e != nil {
		t.Fatal(e)
	}
	if _, e := NewQueryRequest("decimal.fee.v1.Query", "CalculateCommission"); e != nil {
		t.Fatal(e)
	}
	if _, e := NewQueryRequest("unknown.Query", "Unknown"); e == nil {
		t.Fatal("unknown service accepted")
	}
	if _, e := DialGRPC("host@credential", nil); e == nil {
		t.Fatal("credential target")
	}
}
