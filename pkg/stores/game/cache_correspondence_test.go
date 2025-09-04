package game

import (
	"context"
	"testing"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/stretchr/testify/assert"
	"github.com/woogles-io/liwords/pkg/entity"
	gs "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// MockBackingStore for testing
type MockBackingStore struct {
	getCallCount     int
	setCallCount     int
	createCallCount  int
	metadataResponse *pb.GameInfoResponse
	gameResponse     *entity.Game
}

func (m *MockBackingStore) Get(ctx context.Context, id string) (*entity.Game, error) {
	m.getCallCount++
	return m.gameResponse, nil
}

func (m *MockBackingStore) GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error) {
	return m.metadataResponse, nil
}

func (m *MockBackingStore) Set(ctx context.Context, game *entity.Game) error {
	m.setCallCount++
	return nil
}

func (m *MockBackingStore) Create(ctx context.Context, game *entity.Game) error {
	m.createCallCount++
	return nil
}

// Implement other required methods with stubs
func (m *MockBackingStore) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	return nil, nil
}
func (m *MockBackingStore) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	return nil, nil
}
func (m *MockBackingStore) GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	return nil, nil
}
func (m *MockBackingStore) CreateRaw(ctx context.Context, game *entity.Game, gt pb.GameType) error {
	return nil
}
func (m *MockBackingStore) Exists(ctx context.Context, id string) (bool, error) {
	return false, nil
}
func (m *MockBackingStore) ListActive(ctx context.Context, tourneyID string) (*pb.GameInfoResponses, error) {
	return nil, nil
}
func (m *MockBackingStore) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *MockBackingStore) GameEventChan() chan<- *entity.EventWrapper {
	return nil
}
func (m *MockBackingStore) SetGameEventChan(ch chan<- *entity.EventWrapper) {}
func (m *MockBackingStore) Disconnect() {}
func (m *MockBackingStore) SetReady(ctx context.Context, gid string, pidx int) (int, error) {
	return 0, nil
}
func (m *MockBackingStore) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
	return nil, nil
}

func TestCacheBypassForCorrespondenceGames(t *testing.T) {
	ctx := context.Background()
	
	// Create a mock correspondence game
	corresGame := &entity.Game{
		Game: macondogame.Game{},
		GameReq: &entity.GameRequest{
			GameRequest: &pb.GameRequest{
				GameMode: pb.GameMode_CORRESPONDENCE,
			},
		},
	}
	// Initialize the internal Game with a history
	hist := &macondopb.GameHistory{
		Uid: "corres-game-1",
	}
	corresGame.Game.SetHistory(hist)
	
	// Create a mock backing store
	mockStore := &MockBackingStore{
		metadataResponse: &pb.GameInfoResponse{
			GameRequest: &pb.GameRequest{
				GameMode: pb.GameMode_CORRESPONDENCE,
			},
		},
		gameResponse: corresGame,
	}
	
	// Create cache with mock backing store
	cache := NewCache(mockStore)
	
	// Test Get - should bypass cache for correspondence game
	game, err := cache.Get(ctx, "corres-game-1")
	assert.NoError(t, err)
	assert.NotNil(t, game)
	assert.Equal(t, 1, mockStore.getCallCount, "Get should be called once on backing store")
	
	// Call Get again - should still go to backing store (not cached)
	game2, err := cache.Get(ctx, "corres-game-1")
	assert.NoError(t, err)
	assert.NotNil(t, game2)
	assert.Equal(t, 2, mockStore.getCallCount, "Get should be called twice on backing store for correspondence game")
	
	// Test Set - should bypass cache for correspondence game
	err = cache.Set(ctx, mockStore.gameResponse)
	assert.NoError(t, err)
	assert.Equal(t, 1, mockStore.setCallCount, "Set should be called once on backing store")
	
	// Test Create - should bypass cache for correspondence game
	err = cache.Create(ctx, mockStore.gameResponse)
	assert.NoError(t, err)
	assert.Equal(t, 1, mockStore.createCallCount, "Create should be called once on backing store")
}

func TestCacheUsedForRegularGames(t *testing.T) {
	ctx := context.Background()
	
	// Create a mock regular game
	regularGame := &entity.Game{
		Game: macondogame.Game{},
		GameReq: &entity.GameRequest{
			GameRequest: &pb.GameRequest{
				GameMode: pb.GameMode_REAL_TIME,
			},
		},
	}
	// Initialize the internal Game with a history
	hist := &macondopb.GameHistory{
		Uid: "regular-game-1",
	}
	regularGame.Game.SetHistory(hist)
	
	// Create a mock backing store for regular game
	mockStore := &MockBackingStore{
		metadataResponse: &pb.GameInfoResponse{
			GameRequest: &pb.GameRequest{
				GameMode: pb.GameMode_REAL_TIME,
			},
		},
		gameResponse: regularGame,
	}
	
	// Create cache with mock backing store
	cache := NewCache(mockStore)
	
	// Test Get - should use cache for regular game
	game, err := cache.Get(ctx, "regular-game-1")
	assert.NoError(t, err)
	assert.NotNil(t, game)
	assert.Equal(t, 1, mockStore.getCallCount, "Get should be called once on backing store")
	
	// Call Get again - should be cached (no additional backing store call)
	game2, err := cache.Get(ctx, "regular-game-1")
	assert.NoError(t, err)
	assert.NotNil(t, game2)
	assert.Equal(t, 1, mockStore.getCallCount, "Get should still be called only once (cached)")
}