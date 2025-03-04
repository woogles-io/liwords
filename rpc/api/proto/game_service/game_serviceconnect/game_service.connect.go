// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: proto/game_service/game_service.proto

package game_serviceconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	game_service "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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
	// GameMetadataServiceName is the fully-qualified name of the GameMetadataService service.
	GameMetadataServiceName = "game_service.GameMetadataService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// GameMetadataServiceGetMetadataProcedure is the fully-qualified name of the GameMetadataService's
	// GetMetadata RPC.
	GameMetadataServiceGetMetadataProcedure = "/game_service.GameMetadataService/GetMetadata"
	// GameMetadataServiceGetGCGProcedure is the fully-qualified name of the GameMetadataService's
	// GetGCG RPC.
	GameMetadataServiceGetGCGProcedure = "/game_service.GameMetadataService/GetGCG"
	// GameMetadataServiceGetGameHistoryProcedure is the fully-qualified name of the
	// GameMetadataService's GetGameHistory RPC.
	GameMetadataServiceGetGameHistoryProcedure = "/game_service.GameMetadataService/GetGameHistory"
	// GameMetadataServiceGetRecentGamesProcedure is the fully-qualified name of the
	// GameMetadataService's GetRecentGames RPC.
	GameMetadataServiceGetRecentGamesProcedure = "/game_service.GameMetadataService/GetRecentGames"
	// GameMetadataServiceGetRematchStreakProcedure is the fully-qualified name of the
	// GameMetadataService's GetRematchStreak RPC.
	GameMetadataServiceGetRematchStreakProcedure = "/game_service.GameMetadataService/GetRematchStreak"
	// GameMetadataServiceGetGameDocumentProcedure is the fully-qualified name of the
	// GameMetadataService's GetGameDocument RPC.
	GameMetadataServiceGetGameDocumentProcedure = "/game_service.GameMetadataService/GetGameDocument"
)

// GameMetadataServiceClient is a client for the game_service.GameMetadataService service.
type GameMetadataServiceClient interface {
	GetMetadata(context.Context, *connect.Request[game_service.GameInfoRequest]) (*connect.Response[ipc.GameInfoResponse], error)
	// GetGCG gets a GCG string for the given game ID.
	GetGCG(context.Context, *connect.Request[game_service.GCGRequest]) (*connect.Response[game_service.GCGResponse], error)
	// GetGameHistory gets a GameHistory for the given game ID. GameHistory
	// is our internal representation of a game's state.
	GetGameHistory(context.Context, *connect.Request[game_service.GameHistoryRequest]) (*connect.Response[game_service.GameHistoryResponse], error)
	// GetRecentGames gets recent games for a user.
	GetRecentGames(context.Context, *connect.Request[game_service.RecentGamesRequest]) (*connect.Response[ipc.GameInfoResponses], error)
	GetRematchStreak(context.Context, *connect.Request[game_service.RematchStreakRequest]) (*connect.Response[game_service.StreakInfoResponse], error)
	// GetGameDocument gets a Game Document. This will eventually obsolete
	// GetGameHistory. Does not work with annotated games for now.
	GetGameDocument(context.Context, *connect.Request[game_service.GameDocumentRequest]) (*connect.Response[game_service.GameDocumentResponse], error)
}

// NewGameMetadataServiceClient constructs a client for the game_service.GameMetadataService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewGameMetadataServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) GameMetadataServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	gameMetadataServiceMethods := game_service.File_proto_game_service_game_service_proto.Services().ByName("GameMetadataService").Methods()
	return &gameMetadataServiceClient{
		getMetadata: connect.NewClient[game_service.GameInfoRequest, ipc.GameInfoResponse](
			httpClient,
			baseURL+GameMetadataServiceGetMetadataProcedure,
			connect.WithSchema(gameMetadataServiceMethods.ByName("GetMetadata")),
			connect.WithClientOptions(opts...),
		),
		getGCG: connect.NewClient[game_service.GCGRequest, game_service.GCGResponse](
			httpClient,
			baseURL+GameMetadataServiceGetGCGProcedure,
			connect.WithSchema(gameMetadataServiceMethods.ByName("GetGCG")),
			connect.WithClientOptions(opts...),
		),
		getGameHistory: connect.NewClient[game_service.GameHistoryRequest, game_service.GameHistoryResponse](
			httpClient,
			baseURL+GameMetadataServiceGetGameHistoryProcedure,
			connect.WithSchema(gameMetadataServiceMethods.ByName("GetGameHistory")),
			connect.WithClientOptions(opts...),
		),
		getRecentGames: connect.NewClient[game_service.RecentGamesRequest, ipc.GameInfoResponses](
			httpClient,
			baseURL+GameMetadataServiceGetRecentGamesProcedure,
			connect.WithSchema(gameMetadataServiceMethods.ByName("GetRecentGames")),
			connect.WithClientOptions(opts...),
		),
		getRematchStreak: connect.NewClient[game_service.RematchStreakRequest, game_service.StreakInfoResponse](
			httpClient,
			baseURL+GameMetadataServiceGetRematchStreakProcedure,
			connect.WithSchema(gameMetadataServiceMethods.ByName("GetRematchStreak")),
			connect.WithClientOptions(opts...),
		),
		getGameDocument: connect.NewClient[game_service.GameDocumentRequest, game_service.GameDocumentResponse](
			httpClient,
			baseURL+GameMetadataServiceGetGameDocumentProcedure,
			connect.WithSchema(gameMetadataServiceMethods.ByName("GetGameDocument")),
			connect.WithClientOptions(opts...),
		),
	}
}

// gameMetadataServiceClient implements GameMetadataServiceClient.
type gameMetadataServiceClient struct {
	getMetadata      *connect.Client[game_service.GameInfoRequest, ipc.GameInfoResponse]
	getGCG           *connect.Client[game_service.GCGRequest, game_service.GCGResponse]
	getGameHistory   *connect.Client[game_service.GameHistoryRequest, game_service.GameHistoryResponse]
	getRecentGames   *connect.Client[game_service.RecentGamesRequest, ipc.GameInfoResponses]
	getRematchStreak *connect.Client[game_service.RematchStreakRequest, game_service.StreakInfoResponse]
	getGameDocument  *connect.Client[game_service.GameDocumentRequest, game_service.GameDocumentResponse]
}

// GetMetadata calls game_service.GameMetadataService.GetMetadata.
func (c *gameMetadataServiceClient) GetMetadata(ctx context.Context, req *connect.Request[game_service.GameInfoRequest]) (*connect.Response[ipc.GameInfoResponse], error) {
	return c.getMetadata.CallUnary(ctx, req)
}

// GetGCG calls game_service.GameMetadataService.GetGCG.
func (c *gameMetadataServiceClient) GetGCG(ctx context.Context, req *connect.Request[game_service.GCGRequest]) (*connect.Response[game_service.GCGResponse], error) {
	return c.getGCG.CallUnary(ctx, req)
}

// GetGameHistory calls game_service.GameMetadataService.GetGameHistory.
func (c *gameMetadataServiceClient) GetGameHistory(ctx context.Context, req *connect.Request[game_service.GameHistoryRequest]) (*connect.Response[game_service.GameHistoryResponse], error) {
	return c.getGameHistory.CallUnary(ctx, req)
}

// GetRecentGames calls game_service.GameMetadataService.GetRecentGames.
func (c *gameMetadataServiceClient) GetRecentGames(ctx context.Context, req *connect.Request[game_service.RecentGamesRequest]) (*connect.Response[ipc.GameInfoResponses], error) {
	return c.getRecentGames.CallUnary(ctx, req)
}

// GetRematchStreak calls game_service.GameMetadataService.GetRematchStreak.
func (c *gameMetadataServiceClient) GetRematchStreak(ctx context.Context, req *connect.Request[game_service.RematchStreakRequest]) (*connect.Response[game_service.StreakInfoResponse], error) {
	return c.getRematchStreak.CallUnary(ctx, req)
}

// GetGameDocument calls game_service.GameMetadataService.GetGameDocument.
func (c *gameMetadataServiceClient) GetGameDocument(ctx context.Context, req *connect.Request[game_service.GameDocumentRequest]) (*connect.Response[game_service.GameDocumentResponse], error) {
	return c.getGameDocument.CallUnary(ctx, req)
}

// GameMetadataServiceHandler is an implementation of the game_service.GameMetadataService service.
type GameMetadataServiceHandler interface {
	GetMetadata(context.Context, *connect.Request[game_service.GameInfoRequest]) (*connect.Response[ipc.GameInfoResponse], error)
	// GetGCG gets a GCG string for the given game ID.
	GetGCG(context.Context, *connect.Request[game_service.GCGRequest]) (*connect.Response[game_service.GCGResponse], error)
	// GetGameHistory gets a GameHistory for the given game ID. GameHistory
	// is our internal representation of a game's state.
	GetGameHistory(context.Context, *connect.Request[game_service.GameHistoryRequest]) (*connect.Response[game_service.GameHistoryResponse], error)
	// GetRecentGames gets recent games for a user.
	GetRecentGames(context.Context, *connect.Request[game_service.RecentGamesRequest]) (*connect.Response[ipc.GameInfoResponses], error)
	GetRematchStreak(context.Context, *connect.Request[game_service.RematchStreakRequest]) (*connect.Response[game_service.StreakInfoResponse], error)
	// GetGameDocument gets a Game Document. This will eventually obsolete
	// GetGameHistory. Does not work with annotated games for now.
	GetGameDocument(context.Context, *connect.Request[game_service.GameDocumentRequest]) (*connect.Response[game_service.GameDocumentResponse], error)
}

// NewGameMetadataServiceHandler builds an HTTP handler from the service implementation. It returns
// the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewGameMetadataServiceHandler(svc GameMetadataServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	gameMetadataServiceMethods := game_service.File_proto_game_service_game_service_proto.Services().ByName("GameMetadataService").Methods()
	gameMetadataServiceGetMetadataHandler := connect.NewUnaryHandler(
		GameMetadataServiceGetMetadataProcedure,
		svc.GetMetadata,
		connect.WithSchema(gameMetadataServiceMethods.ByName("GetMetadata")),
		connect.WithHandlerOptions(opts...),
	)
	gameMetadataServiceGetGCGHandler := connect.NewUnaryHandler(
		GameMetadataServiceGetGCGProcedure,
		svc.GetGCG,
		connect.WithSchema(gameMetadataServiceMethods.ByName("GetGCG")),
		connect.WithHandlerOptions(opts...),
	)
	gameMetadataServiceGetGameHistoryHandler := connect.NewUnaryHandler(
		GameMetadataServiceGetGameHistoryProcedure,
		svc.GetGameHistory,
		connect.WithSchema(gameMetadataServiceMethods.ByName("GetGameHistory")),
		connect.WithHandlerOptions(opts...),
	)
	gameMetadataServiceGetRecentGamesHandler := connect.NewUnaryHandler(
		GameMetadataServiceGetRecentGamesProcedure,
		svc.GetRecentGames,
		connect.WithSchema(gameMetadataServiceMethods.ByName("GetRecentGames")),
		connect.WithHandlerOptions(opts...),
	)
	gameMetadataServiceGetRematchStreakHandler := connect.NewUnaryHandler(
		GameMetadataServiceGetRematchStreakProcedure,
		svc.GetRematchStreak,
		connect.WithSchema(gameMetadataServiceMethods.ByName("GetRematchStreak")),
		connect.WithHandlerOptions(opts...),
	)
	gameMetadataServiceGetGameDocumentHandler := connect.NewUnaryHandler(
		GameMetadataServiceGetGameDocumentProcedure,
		svc.GetGameDocument,
		connect.WithSchema(gameMetadataServiceMethods.ByName("GetGameDocument")),
		connect.WithHandlerOptions(opts...),
	)
	return "/game_service.GameMetadataService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case GameMetadataServiceGetMetadataProcedure:
			gameMetadataServiceGetMetadataHandler.ServeHTTP(w, r)
		case GameMetadataServiceGetGCGProcedure:
			gameMetadataServiceGetGCGHandler.ServeHTTP(w, r)
		case GameMetadataServiceGetGameHistoryProcedure:
			gameMetadataServiceGetGameHistoryHandler.ServeHTTP(w, r)
		case GameMetadataServiceGetRecentGamesProcedure:
			gameMetadataServiceGetRecentGamesHandler.ServeHTTP(w, r)
		case GameMetadataServiceGetRematchStreakProcedure:
			gameMetadataServiceGetRematchStreakHandler.ServeHTTP(w, r)
		case GameMetadataServiceGetGameDocumentProcedure:
			gameMetadataServiceGetGameDocumentHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedGameMetadataServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedGameMetadataServiceHandler struct{}

func (UnimplementedGameMetadataServiceHandler) GetMetadata(context.Context, *connect.Request[game_service.GameInfoRequest]) (*connect.Response[ipc.GameInfoResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("game_service.GameMetadataService.GetMetadata is not implemented"))
}

func (UnimplementedGameMetadataServiceHandler) GetGCG(context.Context, *connect.Request[game_service.GCGRequest]) (*connect.Response[game_service.GCGResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("game_service.GameMetadataService.GetGCG is not implemented"))
}

func (UnimplementedGameMetadataServiceHandler) GetGameHistory(context.Context, *connect.Request[game_service.GameHistoryRequest]) (*connect.Response[game_service.GameHistoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("game_service.GameMetadataService.GetGameHistory is not implemented"))
}

func (UnimplementedGameMetadataServiceHandler) GetRecentGames(context.Context, *connect.Request[game_service.RecentGamesRequest]) (*connect.Response[ipc.GameInfoResponses], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("game_service.GameMetadataService.GetRecentGames is not implemented"))
}

func (UnimplementedGameMetadataServiceHandler) GetRematchStreak(context.Context, *connect.Request[game_service.RematchStreakRequest]) (*connect.Response[game_service.StreakInfoResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("game_service.GameMetadataService.GetRematchStreak is not implemented"))
}

func (UnimplementedGameMetadataServiceHandler) GetGameDocument(context.Context, *connect.Request[game_service.GameDocumentRequest]) (*connect.Response[game_service.GameDocumentResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("game_service.GameMetadataService.GetGameDocument is not implemented"))
}
