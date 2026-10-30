package cosmos

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GRPC uses pinned descriptors. Query never invokes Msg or BroadcastTx services.
// An injected connection is caller-owned; configure it without write retries.
type GRPC struct {
	conn  grpc.ClientConnInterface
	close func() error
}

func NewGRPC(conn grpc.ClientConnInterface) (*GRPC, error) {
	if conn == nil {
		return nil, errors.New("gRPC connection required")
	}
	return &GRPC{conn: conn}, nil
}

// DialGRPC creates a lazy TLS client with retries disabled. Supplying credentials
// is explicit, e.g. insecure credentials only for a controlled local test node.
func DialGRPC(target string, transport credentials.TransportCredentials) (*GRPC, error) {
	if target == "" || strings.ContainsAny(target, "@?#\\") {
		return nil, errors.New("invalid gRPC target")
	}
	if transport == nil {
		transport = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	}
	conn, e := grpc.NewClient(target, grpc.WithTransportCredentials(transport), grpc.WithDisableRetry(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4<<20), grpc.MaxCallSendMsgSize(4<<20)))
	if e != nil {
		return nil, errors.New("gRPC client initialization failed")
	}
	return &GRPC{conn: conn, close: conn.Close}, nil
}
func (g *GRPC) Close() error {
	if g.close != nil {
		return g.close()
	}
	return nil
}

func grpcMethod(service, method string) (protoreflect.MethodDescriptor, error) {
	descriptor, e := files.FindDescriptorByName(protoreflect.FullName(service))
	if e != nil {
		return nil, errors.New("unknown pinned gRPC service")
	}
	s, ok := descriptor.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, errors.New("not a gRPC service")
	}
	m := s.Methods().ByName(protoreflect.Name(method))
	if m == nil || m.IsStreamingClient() || m.IsStreamingServer() {
		return nil, errors.New("unknown or streaming gRPC method")
	}
	return m, nil
}
func readGRPCMethod(service, method string) bool {
	if strings.HasSuffix(service, ".Query") {
		return true
	}
	return service == "cosmos.tx.v1beta1.Service" && (method == "Simulate" || method == "GetTx" || method == "GetTxsEvent" || method == "GetBlockWithTxs")
}
func NewQueryRequest(service, method string) (*dynamicpb.Message, error) {
	if !readGRPCMethod(service, method) {
		return nil, errors.New("method is not a read-only query")
	}
	m, e := grpcMethod(service, method)
	if e != nil {
		return nil, e
	}
	return dynamicpb.NewMessage(m.Input()), nil
}
func (g *GRPC) Query(ctx context.Context, service, method string, request proto.Message) (*dynamicpb.Message, error) {
	if !readGRPCMethod(service, method) {
		return nil, errors.New("method is not a read-only query")
	}
	return g.invoke(ctx, service, method, request)
}
func (g *GRPC) invoke(ctx context.Context, service, method string, request proto.Message) (*dynamicpb.Message, error) {
	if ctx == nil {
		return nil, errors.New("context required")
	}
	if g == nil || g.conn == nil {
		return nil, errors.New("gRPC connection required")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	m, e := grpcMethod(service, method)
	if e != nil {
		return nil, e
	}
	if request == nil || (reflect.ValueOf(request).Kind() == reflect.Pointer && reflect.ValueOf(request).IsNil()) || !request.ProtoReflect().IsValid() || request.ProtoReflect().Descriptor() != m.Input() {
		return nil, errors.New("gRPC input does not match pinned schema")
	}
	snapshot := proto.Clone(request)
	if proto.Size(snapshot) > 4<<20 {
		return nil, errors.New("gRPC request exceeds limit")
	}
	response := dynamicpb.NewMessage(m.Output())
	if e = g.conn.Invoke(ctx, "/"+service+"/"+method, snapshot, response, grpc.MaxCallRecvMsgSize(4<<20), grpc.MaxCallSendMsgSize(4<<20), grpc.WaitForReady(false)); e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("Cosmos gRPC request failed")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	return response, nil
}
func (g *GRPC) Simulate(ctx context.Context, raw []byte) (GasInfo, error) {
	if _, e := Decode("cosmos.tx.v1beta1.TxRaw", raw); e != nil {
		return GasInfo{}, e
	}
	request, e := NewQueryRequest("cosmos.tx.v1beta1.Service", "Simulate")
	if e != nil {
		return GasInfo{}, e
	}
	set(request, "tx_bytes", protoreflect.ValueOfBytes(append([]byte(nil), raw...)))
	response, e := g.Query(ctx, "cosmos.tx.v1beta1.Service", "Simulate", request)
	if e != nil {
		return GasInfo{}, e
	}
	field := response.Descriptor().Fields().ByName("gas_info")
	if !response.Has(field) {
		return GasInfo{}, errors.New("simulation gas info missing")
	}
	gas := response.Get(field).Message()
	return GasInfo{Wanted: strconv.FormatUint(gas.Get(gas.Descriptor().Fields().ByName("gas_wanted")).Uint(), 10), Used: strconv.FormatUint(gas.Get(gas.Descriptor().Fields().ByName("gas_used")).Uint(), 10)}, nil
}
func grpcTxResponse(response *dynamicpb.Message) (BroadcastResult, error) {
	field := response.Descriptor().Fields().ByName("tx_response")
	if field == nil || !response.Has(field) {
		return BroadcastResult{}, errors.New("transaction response missing")
	}
	tx := response.Get(field).Message()
	fields := tx.Descriptor().Fields()
	return BroadcastResult{Hash: tx.Get(fields.ByName("txhash")).String(), Code: uint32(tx.Get(fields.ByName("code")).Uint()), Height: strconv.FormatInt(tx.Get(fields.ByName("height")).Int(), 10)}, nil
}

// Broadcast is the sole write method. It submits once with SYNC and preserves
// the locally calculated transaction hash on every ambiguous response.
func (g *GRPC) Broadcast(ctx context.Context, raw []byte) (BroadcastResult, error) {
	snapshot := append([]byte(nil), raw...)
	if e := ValidateTxRaw(snapshot); e != nil {
		return BroadcastResult{}, e
	}
	if len(snapshot) > 4<<20 {
		return BroadcastResult{}, errors.New("transaction exceeds gRPC limit")
	}
	if e := ctx.Err(); e != nil {
		return BroadcastResult{}, e
	}
	hash := TransactionHash(snapshot)
	request := message("cosmos.tx.v1beta1.BroadcastTxRequest")
	set(request, "tx_bytes", protoreflect.ValueOfBytes(snapshot))
	set(request, "mode", protoreflect.ValueOfEnum(2))
	response, e := g.invoke(ctx, "cosmos.tx.v1beta1.Service", "BroadcastTx", request)
	if e != nil {
		return BroadcastResult{}, &SubmissionUnknownError{hash}
	}
	result, e := grpcTxResponse(response)
	if e != nil || !strings.EqualFold(result.Hash, hex.EncodeToString(hash[:])) {
		return result, &SubmissionUnknownError{hash}
	}
	if result.Code != 0 {
		return result, errors.New("Cosmos CheckTx rejected transaction")
	}
	return result, nil
}
func (g *GRPC) Transaction(ctx context.Context, hash [32]byte) (BroadcastResult, error) {
	request, e := NewQueryRequest("cosmos.tx.v1beta1.Service", "GetTx")
	if e != nil {
		return BroadcastResult{}, e
	}
	set(request, "hash", protoreflect.ValueOfString(strings.ToUpper(hex.EncodeToString(hash[:]))))
	response, e := g.Query(ctx, "cosmos.tx.v1beta1.Service", "GetTx", request)
	if e != nil {
		return BroadcastResult{}, e
	}
	result, e := grpcTxResponse(response)
	if e != nil {
		return result, e
	}
	if !strings.EqualFold(result.Hash, hex.EncodeToString(hash[:])) {
		return result, errors.New("transaction confirmation hash mismatch")
	}
	if result.Code != 0 {
		return result, errors.New("Cosmos DeliverTx failed")
	}
	if e := validateConfirmationHeight(result.Height); e != nil {
		return result, e
	}
	return result, nil
}
