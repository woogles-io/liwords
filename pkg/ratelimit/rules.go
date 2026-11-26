package ratelimit

import (
	"strings"
	"time"
)

// A bucket's limits.
//
// Logged-in callers (session cookie or API key) are counted per user, at User.
//
// Anonymous callers are counted per browser (IP and client ID cookie) at
// Anon, so a school full of kids on one IP each get their own. Client IDs
// can be made up, so all anonymous traffic from one IP is also capped at
// IPCeiling. Requests without a client ID (scripts, first visits) share one
// bucket per IP at Anon.
//
// AlwaysAnon buckets count everyone as anonymous: they guard endpoints
// people call before logging in.
type bucket struct {
	Name       string
	Anon       Limit
	IPCeiling  Limit
	User       Limit
	AlwaysAnon bool
}

var (
	// Every request. A page load fires a dozen calls at once, and busy pages
	// reach a few hundred a minute, so this only stops floods.
	generalBucket = bucket{
		Name:      "general",
		Anon:      Limit{Rate: 600, Per: time.Minute, Burst: 600},
		IPCeiling: Limit{Rate: 6000, Per: time.Minute, Burst: 3000},
		User:      Limit{Rate: 1200, Per: time.Minute, Burst: 1200},
	}
	// Endpoints that read a lot of data: S3 downloads, whole game documents,
	// rendered images. A person reading games stays far below this.
	heavyBucket = bucket{
		Name:      "heavy",
		Anon:      Limit{Rate: 60, Per: time.Minute, Burst: 60},
		IPCeiling: Limit{Rate: 300, Per: time.Minute, Burst: 150},
		User:      Limit{Rate: 240, Per: time.Minute, Burst: 120},
	}
	// Endpoints that send an email. The ceiling leaves room for a class
	// signing up together.
	emailBucket = bucket{
		Name:       "email",
		Anon:       Limit{Rate: 5, Per: time.Hour, Burst: 5},
		IPCeiling:  Limit{Rate: 60, Per: time.Hour, Burst: 40},
		AlwaysAnon: true,
	}
	// Endpoints that hash a password (argon2, 64 MB each).
	passwordHashBucket = bucket{
		Name:       "password-hash",
		Anon:       Limit{Rate: 10, Per: time.Minute, Burst: 10},
		IPCeiling:  Limit{Rate: 60, Per: time.Minute, Burst: 40},
		AlwaysAnon: true,
	}
)

// Paths are as the middleware sees them: Connect procedures without the /api
// prefix, other routes as they are.
var procedureBuckets = map[string][]bucket{
	"/analysis_service.AnalysisService/GetAnalysisResult":       {heavyBucket},
	"/game_service.GameMetadataService/GetGCG":                  {heavyBucket},
	"/game_service.GameMetadataService/GetGameHistory":          {heavyBucket},
	"/game_service.GameMetadataService/GetGameDocument":         {heavyBucket},
	"/game_service.GameMetadataService/GetRecentGames":          {heavyBucket},
	"/omgwords_service.GameEventService/GetGameDocument":        {heavyBucket},
	"/omgwords_service.GameEventService/GetCGP":                 {heavyBucket},
	"/word_service.WordService/DefineWords":                     {heavyBucket},
	"/tournament_service.TournamentService/GetTournament":       {heavyBucket},
	"/tournament_service.TournamentService/RecentGames":         {heavyBucket},
	"/league_service.LeagueService/GetDivisionStandings":        {heavyBucket},
	"/league_service.LeagueService/GetAllDivisionStandings":     {heavyBucket},
	"/league_service.LeagueService/GetLeagueRoster":             {heavyBucket},
	"/league_service.LeagueService/GetPlayerLeagueHistory":      {heavyBucket},
	"/league_service.LeagueService/GetPlayerLeagueH2H":          {heavyBucket},
	"/league_service.LeagueService/GetPlayerSeasonGames":        {heavyBucket},
	"/user_service.RegistrationService/Register":                {emailBucket, passwordHashBucket},
	"/user_service.RegistrationService/ResendVerificationEmail": {emailBucket},
	"/user_service.RegistrationService/ConfirmEmailChange":      {emailBucket},
	"/user_service.AuthenticationService/ResetPasswordStep1":    {emailBucket},
	"/user_service.AuthenticationService/ResetPasswordStep2":    {passwordHashBucket},
}

var prefixBuckets = []struct {
	prefix  string
	buckets []bucket
}{
	{"/gameimg/", []bucket{heavyBucket}}, // memento.GameimgPrefix
	{"/embed/", []bucket{heavyBucket}},   // embed.EmbedServicePrefix
}

// bucketsFor lists the buckets a request takes a token from: general, plus
// any its endpoint adds.
func bucketsFor(path string) []bucket {
	if bs, ok := procedureBuckets[path]; ok {
		return append([]bucket{generalBucket}, bs...)
	}
	for _, p := range prefixBuckets {
		if strings.HasPrefix(path, p.prefix) {
			return append([]bucket{generalBucket}, p.buckets...)
		}
	}
	return []bucket{generalBucket}
}
