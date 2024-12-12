package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/domino14/macondo/game"
	"github.com/woogles-io/liwords/pkg/glicko"
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

func NewDefaultRating(lastGameIsNow bool) *SingleRating {
	lastGameTimeStamp := int64(0)
	if lastGameIsNow {
		lastGameTimeStamp = time.Now().Unix()
	}
	return &SingleRating{
		Rating:            float64(glicko.InitialRating),
		RatingDeviation:   float64(glicko.InitialRatingDeviation),
		Volatility:        glicko.InitialVolatility,
		LastGameTimestamp: int64(lastGameTimeStamp),
	}
}

const PuzzleVariant = "puzzle"

func ToVariantKey(lexiconName string, variantName game.Variant, timeControl TimeControl) VariantKey {
	return VariantKey(transformLexiconName(lexiconName) + "." + string(variantName) + "." + string(timeControl))
}

func LexiconToPuzzleVariantKey(lexicon string) VariantKey {
	return ToVariantKey(lexicon, PuzzleVariant, TCCorres)
}

func (r *Ratings) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *Ratings) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unexpected type %T for ratings", value)
	}

	return json.Unmarshal(b, &r)
}

// XXX: Get rid of these when we're no longer using Gorm anywhere.
// PGXV5 should be able to directly scan into these types.
func (r *SingleRating) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *SingleRating) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unexpected type %T for SingleRating", value)
	}

	return json.Unmarshal(b, &r)

}

func transformLexiconName(lexiconName string) string {
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
	case strings.HasPrefix(lexiconName, "NSF"):
		newlex = "NSF21"
	case strings.HasPrefix(lexiconName, "RD"):
		newlex = "RD28"
	case strings.HasPrefix(lexiconName, "FRA"):
		newlex = "FRA20"
	case strings.HasPrefix(lexiconName, "DISC"):
		newlex = "DISC"
	case strings.HasPrefix(lexiconName, "OSPS"):
		newlex = "OSPS"
	case strings.HasPrefix(lexiconName, "FILE"):
		newlex = "FILE"
	default:
		newlex = lexiconName
	}
	return newlex
}
