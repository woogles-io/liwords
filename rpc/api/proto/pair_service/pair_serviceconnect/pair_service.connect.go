// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: proto/pair_service/pair_service.proto

package pair_serviceconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pair_service "github.com/woogles-io/liwords/rpc/api/proto/pair_service"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// PairServiceName is the fully-qualified name of the PairService service.
	PairServiceName = "pair_service.PairService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// PairServiceHandlePairRequestProcedure is the fully-qualified name of the PairService's
	// HandlePairRequest RPC.
	PairServiceHandlePairRequestProcedure = "/pair_service.PairService/HandlePairRequest"
)

// PairServiceClient is a client for the pair_service.PairService service.
type PairServiceClient interface {
	HandlePairRequest(context.Context, *connect.Request[ipc.PairRequest]) (*connect.Response[ipc.PairResponse], error)
}

// NewPairServiceClient constructs a client for the pair_service.PairService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewPairServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) PairServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	pairServiceMethods := pair_service.File_proto_pair_service_pair_service_proto.Services().ByName("PairService").Methods()
	return &pairServiceClient{
		handlePairRequest: connect.NewClient[ipc.PairRequest, ipc.PairResponse](
			httpClient,
			baseURL+PairServiceHandlePairRequestProcedure,
			connect.WithSchema(pairServiceMethods.ByName("HandlePairRequest")),
			connect.WithClientOptions(opts...),
		),
	}
}

// pairServiceClient implements PairServiceClient.
type pairServiceClient struct {
	handlePairRequest *connect.Client[ipc.PairRequest, ipc.PairResponse]
}

// HandlePairRequest calls pair_service.PairService.HandlePairRequest.
func (c *pairServiceClient) HandlePairRequest(ctx context.Context, req *connect.Request[ipc.PairRequest]) (*connect.Response[ipc.PairResponse], error) {
	return c.handlePairRequest.CallUnary(ctx, req)
}

// PairServiceHandler is an implementation of the pair_service.PairService service.
type PairServiceHandler interface {
	HandlePairRequest(context.Context, *connect.Request[ipc.PairRequest]) (*connect.Response[ipc.PairResponse], error)
}

// NewPairServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewPairServiceHandler(svc PairServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	pairServiceMethods := pair_service.File_proto_pair_service_pair_service_proto.Services().ByName("PairService").Methods()
	pairServiceHandlePairRequestHandler := connect.NewUnaryHandler(
		PairServiceHandlePairRequestProcedure,
		svc.HandlePairRequest,
		connect.WithSchema(pairServiceMethods.ByName("HandlePairRequest")),
		connect.WithHandlerOptions(opts...),
	)
	return "/pair_service.PairService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case PairServiceHandlePairRequestProcedure:
			pairServiceHandlePairRequestHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedPairServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedPairServiceHandler struct{}

func (UnimplementedPairServiceHandler) HandlePairRequest(context.Context, *connect.Request[ipc.PairRequest]) (*connect.Response[ipc.PairResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("pair_service.PairService.HandlePairRequest is not implemented"))
}
