package entity

import (
	"strings"

	"github.com/domino14/macondo/game"
)

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

func ToVariantKey(lexiconName string, variantName game.Variant, timeControl TimeControl) VariantKey {
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
		// This is user-visible as CEL, but we still call it ECWL internally.
		// This is the Common English List.
		newlex = "ECWL"
	default:
		newlex = lexiconName
	}
	return VariantKey(newlex + "." + string(variantName) + "." + string(timeControl))
}
