package entity

import (
	"time"
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
}
