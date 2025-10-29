package entity

import (
	"time"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type League struct {
	UUID             string
	Name             string
	Description      string
	Slug             string
	Settings         *pb.LeagueSettings
	CurrentSeasonID  string
	IsActive         bool
	CreatedAt        time.Time
	CreatedBy        int64
}

type Season struct {
	UUID          string
	LeagueID      string
	SeasonNumber  int32
	StartDate     time.Time
	EndDate       time.Time
	ActualEndDate *time.Time
	Status        pb.SeasonStatus
	Divisions     []*Division
}

type Division struct {
	UUID           string
	SeasonID       string
	DivisionNumber int32
	DivisionName   string
	Players        []*pb.PlayerRegistration
	GameIDs        []string
	Standings      []*pb.LeaguePlayerStanding
	IsComplete     bool
}

type PlayerRegistration struct {
	UserID           string
	SeasonID         string
	DivisionID       string
	RegistrationDate time.Time
	StartingRating   int32
	FirstsCount      int32
	Status           string
}

type LeagueStanding struct {
	DivisionID     string
	UserID         string
	Rank           int32
	Wins           int32
	Losses         int32
	Draws          int32
	Spread         int32
	GamesPlayed    int32
	GamesRemaining int32
	Result         pb.StandingResult
	UpdatedAt      time.Time
}
