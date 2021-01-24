package entity

import "strings"

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
	// Transform Lexicon name
	var newlex string
	switch {
	case strings.HasPrefix(lexiconName, "NWL"):
		// Internally we're always going to represent the lexicon as this
		// until we do a data migration.
		newlex = "NWL18"
	case strings.HasPrefix(lexiconName, "CSW"):
		newlex = "CSW19"
	case strings.HasPrefix(lexiconName, "ECWL"):
		newlex = "ECWL"
	default:
		newlex = lexiconName
	}
	return VariantKey(newlex + "." + string(variantName) + "." + string(timeControl))
}
