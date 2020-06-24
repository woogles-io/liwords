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
	UUID      string
	Username  string
	Password  string
	Email     string
}

// Session - The db specific-details are in the store package.
type Session struct {
	ID       string
	Username string
	UserUUID string
}
