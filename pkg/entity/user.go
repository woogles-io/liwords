package entity

import "github.com/jinzhu/gorm"

// A User should be a minimal object. All information such as user profile,
// awards, ratings, records, etc should be in a profile object that
// joins 1-1 with this User object.
type User struct {
	gorm.Model

	UUID     string `gorm:"type:varchar(24);index"`
	Username string `gorm:"type:varchar(32);unique_index"`
	// Password will be hashed.
	Password string `gorm:"type:varchar(128)"`
}
