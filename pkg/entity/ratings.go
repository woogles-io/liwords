package entity

// SingleRating encodes a whole Glicko-225 rating object.
type SingleRating struct {
	Rating          float64 `json:"r"`
	RatingDeviation float64 `json:"rd"`
	Volatility      float64 `json:"v"`
	// This is the last game timestamp for this user for THIS variant:
	LastGameTimestamp int64 `json:"ts"`
}

// Ratings gets stored into a PostgreSQL database.
type Ratings struct {
	Data map[VariantKey]SingleRating
}

type VariantKey string

func ToVariantKey(lexiconName string, variantName Variant, timeControl TimeControl) VariantKey {
	return VariantKey(lexiconName + "." + string(variantName) + "." + string(timeControl))
}
