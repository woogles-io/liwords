package entity

// A Player has a username and a rating. They may also have other
// fields but these are the most important at the moment.
type Player struct {
	username string
	rating   float64

	// Maybe we want these later?
	bulletRating  float64
	blitzRating   float64
	rapidRating   float64
	classicRating float64
}
