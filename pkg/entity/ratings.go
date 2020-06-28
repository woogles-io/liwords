package entity

type SingleRating struct {
	Rating          float64 `json:"r"`
	RatingDeviation float64 `json:"rd"`
	Volatility      float64 `json:"v"`
}

type Ratings struct {
	Data map[VariantKey]SingleRating
}

type VariantKey string

func ToVariantKey(lexiconName string, variantName Variant, timeControl TimeControl) VariantKey {
	return VariantKey(lexiconName + "." + string(variantName) + "." + string(timeControl))
}
