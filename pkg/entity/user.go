package entity

import (
	"time"

	"github.com/jinzhu/gorm"
)

const (
	// SessionExpiration - Expire a session after this much time.
	SessionExpiration = time.Hour * 24 * 30
)

// A User should be a minimal object. All information such as user profile,
// awards, ratings, records, etc should be in a profile object that
// joins 1-1 with this User object.
// XXX: move db details to store package.
type User struct {
	gorm.Model

	UUID     string `gorm:"type:varchar(24);index"`
	Username string `gorm:"type:varchar(32);unique_index"`
	Email    string `gorm:"type:varchar(100);unique_index"`
	// Password will be hashed.
	Password string `gorm:"type:varchar(128)"`
}

// Session - The db specific-details are in the store package.
type Session struct {
	ID       string
	Username string
	UserUUID string
}
