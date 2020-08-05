package entity

type PoolMember struct {
	Id          int
	Rating      int
	RatingRange [2]int
	Blocking    []int
	Misses      int
}

func NewPoolMember(id int, rating int, minimumRating int, maximumRating int, blocking []int) *PoolMember {
	return &PoolMember{
		Id:          id,
		Rating:      rating,
		RatingRange: [2]int{minimumRating, maximumRating},
		Blocking:    blocking,
		Misses:      0}
}
