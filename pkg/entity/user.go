package entity

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/glicko"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

const (
	// SessionExpiration - Expire a session after this much time.
	SessionExpiration = time.Hour * 24 * 30
)

type AuthMethod string

const (
	AuthMethodCookie = "cookie"
	AuthMethodAPIKey = "apikey"
)

// DEPRECATED: use db actions
type Actions struct {
	Current map[string]*ms.ModAction
	History []*ms.ModAction
}

// User - the db-specific details are in the store package.
type User struct {
	sync.RWMutex

	Anonymous bool
	// ID is the database ID. Since this increases monotonically, we should
	// not expose it to the user
	ID uint
	// UUID is the "user-exposed" ID, in any APIs.
	UUID     string
	Username string
	Password string
	Email    string
	Profile  *Profile
	// CurrentChannel tracks presence; where is the user currently?
	CurrentChannel string
	IsBot          bool

	// Email verification fields
	Verified              bool
	VerificationToken     string
	VerificationExpiresAt time.Time

	// DEPRECATED: use db actions
	Actions      *Actions
	Notoriety    int
	AuthedMethod AuthMethod
}

type UserPermission int

const (
	PermDirector UserPermission = iota
	PermMod
	PermAdmin
	PermBot
)

// Session - The db specific-details are in the store package.
type Session struct {
	ID        string
	Username  string
	UserUUID  string
	Expiry    time.Time
	CSRFToken string
}

// Profile is a user profile. It might not be defined for anonymous users.
type Profile struct {
	FirstName string
	LastName  string
	// BirthDate uses ISO format YYYY-MM-DD
	BirthDate   string
	CountryCode string
	Title       string
	About       string
	Ratings     Ratings
	Stats       ProfileStats
	AvatarUrl   string
}

// If the RD is <= this number, the rating is "known"
const RatingDeviationConfidence = float64(glicko.MinimumRatingDeviation + 30)

// RelevantRating returns the rating from a Ratings object given a rating key.
func RelevantRating(ratings Ratings, ratingKey VariantKey) string {
	return RelevantRatingValue(ratings, ratingKey).String()
}

// RelevantRatingValue is the rating RelevantRating displays, as a number.
func RelevantRatingValue(ratings Ratings, ratingKey VariantKey) StartingRating {
	ratdict, ok := ratings.Data[ratingKey]
	if !ok {
		// An unrated user, or one with no rating in this variant, has the
		// default rating.
		return StartingRating{Rating: float64(glicko.InitialRating), Provisional: true}
	}
	return StartingRating{
		Rating:      ratdict.Rating,
		Provisional: ratdict.RatingDeviation > RatingDeviationConfidence,
	}
}

// StartingRating is a player's rating in a game's variant as of the start of
// the game.
type StartingRating struct {
	Rating float64
	// Provisional is true when the rating is not yet confident, which it is
	// displayed with a "?" for.
	Provisional bool
}

// String is the rating as players see it: rounded, with "?" if provisional.
func (r StartingRating) String() string {
	s := strconv.Itoa(int(math.Round(r.Rating)))
	if r.Provisional {
		s += "?"
	}
	return s
}

// ParseDisplayRating reads back a rating that String wrote, such as the ones
// in a game's quickdata. It returns nil for anything else, like "UnratedAnon"
// or an empty string. The rating comes back rounded, as it was displayed.
func ParseDisplayRating(s string) *StartingRating {
	provisional := strings.HasSuffix(s, "?")
	n, err := strconv.Atoi(strings.TrimSuffix(s, "?"))
	if err != nil {
		return nil
	}
	return &StartingRating{Rating: float64(n), Provisional: provisional}
}

// GetRelevantRating gets a displayable rating for this user, based on the passed-in
// rating key (encoding variant, time control, etc)
func (u *User) GetRelevantRating(ratingKey VariantKey) string {
	if u.Profile == nil {
		return "UnratedAnon"
	}
	return RelevantRating(u.Profile.Ratings, ratingKey)
}

// GetStartingRating is GetRelevantRating as a number, or nil for a user with
// no profile, whose rating GetRelevantRating gives as "UnratedAnon".
func (u *User) GetStartingRating(ratingKey VariantKey) *StartingRating {
	if u.Profile == nil {
		return nil
	}
	r := RelevantRatingValue(u.Profile.Ratings, ratingKey)
	return &r
}

// GetRating gets a full Glicko-225 rating for this user, based on the
// passed-in rating key.
func (u *User) GetRating(ratingKey VariantKey) (*SingleRating, error) {
	if u.Profile == nil {
		return nil, errors.New("anonymous user has no rating")
	}
	defaultRating := NewDefaultRating(false)
	if u.Profile.Ratings.Data == nil {
		return defaultRating, nil
	}
	ratdict, ok := u.Profile.Ratings.Data[ratingKey]
	if !ok {
		// Ratings dictionary exists, but user has no rating for this variant.
		return defaultRating, nil
	}
	return &ratdict, nil
}

func (u *User) GetProtoRatings() (map[string]*pb.ProfileUpdate_Rating, error) {
	if u.Profile == nil {
		return nil, errors.New("anonymous user has no rating")
	}
	if u.Profile.Ratings.Data == nil {
		return nil, nil
	}
	ratings := make(map[string]*pb.ProfileUpdate_Rating)
	for k, v := range u.Profile.Ratings.Data {
		ratings[string(k)] = &pb.ProfileUpdate_Rating{
			Rating:    float64(v.Rating),
			Deviation: float64(v.RatingDeviation),
		}
	}
	return ratings, nil
}

// RealName returns a user's real name, or an empty string if anonymous.
func (u *User) RealName() string {
	if u.Profile != nil {
		if u.Profile.FirstName != "" {
			if u.Profile.LastName != "" {
				return u.Profile.FirstName + " " + u.Profile.LastName
			} else {
				return u.Profile.FirstName
			}
		} else {
			return u.Profile.LastName
		}
	}
	return ""
}

// RealNameIfNotYouth returns a user's real name, only if they are older than
// 13. If a birth date has not been provided, do not show it.
func (u *User) RealNameIfNotYouth() string {
	if u.Profile == nil {
		return ""
	}
	if u.IsChild() == pb.ChildStatus_NOT_CHILD {
		return u.RealName()
	}
	return ""
}

func (u *User) AvatarUrl() string {
	if u.IsBot && u.Profile.AvatarUrl == "" {
		return "https://woogles-prod-assets.s3.amazonaws.com/macondog.png"
	} else {
		return u.Profile.AvatarUrl
	}
}

// TournamentID returns the "player ID" of a user. UUID:username is probably not
// a good design, but let's at least narrow it down to this function.
func (u *User) TournamentID() string {
	return u.UUID + ":" + u.Username
}

func InferChildStatus(dob string, now time.Time) pb.ChildStatus {
	// The birth date must be in the form YYYY-MM-DD
	birthDateTime, err := time.Parse(time.RFC3339Nano, dob+"T00:00:00.000Z")
	if err != nil {
		// This means the birth date was either not defined or malformed
		// Either way, the child status should be unknown
		log.Debug().Str("dob", dob).Err(err).Msg("InferChildStatus parse error")
		return pb.ChildStatus_UNKNOWN
	} else {
		timeOfNotChild := birthDateTime.AddDate(13, 0, 0)
		if now.After(timeOfNotChild) {
			return pb.ChildStatus_NOT_CHILD
		} else {
			return pb.ChildStatus_CHILD
		}
	}
}

func IsAdult(dob string, now time.Time) bool {
	return InferChildStatus(dob, now) == pb.ChildStatus_NOT_CHILD
}

func (u *User) IsChild() pb.ChildStatus {
	if u.Profile == nil {
		return pb.ChildStatus_UNKNOWN
	}
	return InferChildStatus(u.Profile.BirthDate, time.Now())
}

type ProfileEntitlements struct {
	BestBotGames struct {
		Since time.Time `json:"since"`
		Count int       `json:"count"`
	} `json:"best_bot_games"`
}
