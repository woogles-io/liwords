package entity

import (
	"time"
)

type PoolMember struct {
	Id          int
	Rating      int
	RatingRange [2]int
	Blocking    []int
	Since       time.Time
	SitCounter  int
	Misses      int
}

func NewPoolMember(id int, rating int, minimumRating int, maximumRating int, blocking []int) *PoolMember {
	return &PoolMember{
		Id:          id,
		Rating:      rating,
		RatingRange: [2]int{minimumRating, maximumRating},
		Blocking:    blocking,
		Since:       time.Now(),
		SitCounter:  0,
		Misses:      0}
}
