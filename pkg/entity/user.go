package entity

import (
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/domino14/liwords/pkg/glicko"
)

const (
	// SessionExpiration - Expire a session after this much time.
	SessionExpiration = time.Hour * 24 * 30
)

// User - the db-specific details are in the store package.
type User struct {
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
}

// Session - The db specific-details are in the store package.
type Session struct {
	ID       string
	Username string
	UserUUID string
}

// Profile is a user profile. It might not be defined for anonymous users.
type Profile struct {
	FirstName   string
	LastName    string
	CountryCode string
	Title       string
	About       string
	Ratings     Ratings
	Stats       ProfileStats
}

// RelevantRating returns the rating from a Ratings object given a rating key.
func RelevantRating(ratings Ratings, ratingKey VariantKey) string {
	if ratings.Data == nil {
		// This is not an unrated user. Use default rating.
		return strconv.Itoa(glicko.InitialRating) + "?"
	}
	ratdict, ok := ratings.Data[ratingKey]
	if ok {
		return strconv.Itoa(int(math.Round(ratdict.Rating)))
	}
	// User has no rating in this particular variant.
	return strconv.Itoa(glicko.InitialRating) + "?"
}

// GetRelevantRating gets a displayable rating for this user, based on the passed-in
// rating key (encoding variant, time control, etc)
func (u *User) GetRelevantRating(ratingKey VariantKey) string {
	if u.Profile == nil {
		return "UnratedAnon"
	}
	return RelevantRating(u.Profile.Ratings, ratingKey)
}

// GetRating gets a full Glicko-225 rating for this user, based on the
// passed-in rating key.
func (u *User) GetRating(ratingKey VariantKey) (*SingleRating, error) {
	if u.Profile == nil {
		return nil, errors.New("anonymous user has no rating")
	}
	defaultRating := &SingleRating{
		Rating:          float64(glicko.InitialRating),
		RatingDeviation: float64(glicko.InitialRatingDeviation),
		Volatility:      glicko.InitialVolatility,
	}
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

// RealName returns a user's real name, or an empty string if anonymous.
func (u *User) RealName() string {
	if u.Profile != nil {
		if u.Profile.FirstName != "" {
			return u.Profile.FirstName + " " + u.Profile.LastName
		} else {
			return u.Profile.LastName
		}
	}
	return ""
}
