package glicko

import (
	"math"
)

const (
	InitialRating               int     = 1500
	InitialVolatility           float64 = 0.06
	MaximumVolatility           float64 = 0.1
	InitialRatingDeviation      int     = 350
	MinimumRatingDeviation      int     = 60
	MaximumRatingDeviation      int     = 350
	VolatilityDeltaConstraint   float64 = 0.5
	GlickoToGlicko225Conversion float64 = 173.7178
	ConvergenceTolerance        float64 = 0.000001
	SpreadScaling               int     = 125
	MinimumWinBoost             float64 = float64(1.0 / 3.0)
	Steepness                   float64 = 0.005
	RatingPeriodinSeconds       int     = 60 * 60 * 24 * 4
	iterationMaximum            int     = 1000
)

// Params are the constants of one rating system. V1 is the system used for
// games today, and for puzzles; V2 is the system described in
// docs/ratings_plan.md.
type Params struct {
	InitialRating          float64 // a new player's rating, and the scale's zero point
	InitialRatingDeviation float64
	MinimumRatingDeviation float64
	MaximumRatingDeviation float64
	InitialVolatility      float64
	MaximumVolatility      float64
	VolatilityConstraint   float64 // tau
	Conversion             float64 // rating points per Glicko-2 unit; sets the curve's width
	SpreadScaling          int
	MinimumWinBoost        float64
	Steepness              float64 // how fast the win boost rises with rating
	WinBoostCenter         float64 // the rating where the win boost is halfway
	RatingPeriodSeconds    int
	// EstablishedRatingDeviation is the highest RD at which a rating counts
	// as established. Above it the rating shows with a "?", and seeks that
	// require an established rating are closed to the player.
	EstablishedRatingDeviation float64
}

var V1 = Params{
	InitialRating:              float64(InitialRating),
	InitialRatingDeviation:     float64(InitialRatingDeviation),
	MinimumRatingDeviation:     float64(MinimumRatingDeviation),
	MaximumRatingDeviation:     float64(MaximumRatingDeviation),
	InitialVolatility:          InitialVolatility,
	MaximumVolatility:          MaximumVolatility,
	VolatilityConstraint:       VolatilityDeltaConstraint,
	Conversion:                 GlickoToGlicko225Conversion,
	SpreadScaling:              SpreadScaling,
	MinimumWinBoost:            MinimumWinBoost,
	Steepness:                  Steepness,
	WinBoostCenter:             float64(InitialRating),
	RatingPeriodSeconds:        RatingPeriodinSeconds,
	EstablishedRatingDeviation: float64(MinimumRatingDeviation + 30),
}

// V2 flattens the curve to about NASPA's (ten-to-one odds at about 725 points
// apart instead of 400), starts new players lower and less certain, and
// rescales the win boost's steepness to the new curve. Its center stays at
// 1500. The established cutoff is V1's 90 scaled to the wider curve (about
// 90 × 315 / 174); left at 90, most players would need over 100 games to lose
// the "?".
var V2 = Params{
	InitialRating:              1200,
	InitialRatingDeviation:     600,
	MinimumRatingDeviation:     60,
	MaximumRatingDeviation:     600,
	InitialVolatility:          InitialVolatility,
	MaximumVolatility:          MaximumVolatility,
	VolatilityConstraint:       VolatilityDeltaConstraint,
	Conversion:                 315,
	SpreadScaling:              SpreadScaling,
	MinimumWinBoost:            MinimumWinBoost,
	Steepness:                  0.00276,
	WinBoostCenter:             1500,
	RatingPeriodSeconds:        RatingPeriodinSeconds,
	EstablishedRatingDeviation: 160,
}

// Rate rates a player under V1.
func Rate(
	playerUnscaledRating float64,
	playerUnscaledRatingDeviation float64,
	playerVolatility float64,
	opponentUnscaledRating float64,
	opponentUnscaledRatingDeviation float64,
	spread int,
	secondsSinceLastGame int) (float64, float64, float64) {
	return V1.Rate(playerUnscaledRating, playerUnscaledRatingDeviation, playerVolatility,
		opponentUnscaledRating, opponentUnscaledRatingDeviation, spread, secondsSinceLastGame)
}

// Rate returns the player's new rating, rating deviation and volatility after
// one game. spread is from the player's point of view.
func (p Params) Rate(
	playerUnscaledRating float64,
	playerUnscaledRatingDeviation float64,
	playerVolatility float64,
	opponentUnscaledRating float64,
	opponentUnscaledRatingDeviation float64,
	spread int,
	secondsSinceLastGame int) (float64, float64, float64) {

	// Step 1 of the Glicko-225 algorithm was performed upon account creation
	// Step 2 of the Glicko-225 algorithm
	playerRating := p.toGlicko(playerUnscaledRating)
	playerRatingDeviation := playerUnscaledRatingDeviation / p.Conversion
	opponentRating := p.toGlicko(opponentUnscaledRating)
	opponentRatingDeviation := opponentUnscaledRatingDeviation / p.Conversion

	// Precompute these values for efficiency
	opponentAdjustedRatingDeviation := adjustedRatingDeviation(opponentRatingDeviation)
	expectedValue := expectedValue(playerRating, opponentRating, opponentAdjustedRatingDeviation)

	// Step 3 of the Glicko-225 algorithm
	variance := 1 / variance(opponentAdjustedRatingDeviation, expectedValue)

	// Step 4 of the Glicko-225 algorithm
	awb := p.adjustWinBoost(float64(playerUnscaledRating+opponentUnscaledRating) / 2)
	improvement := p.improvement(opponentAdjustedRatingDeviation, awb, expectedValue, spread)
	improvementDelta := variance * improvement

	// Step 5 of the Glicko-225 algorithm
	tau := p.VolatilityConstraint
	a := math.Log(math.Pow(playerVolatility, 2))
	A := a
	deltaSquared := math.Pow(improvementDelta, 2)
	rdSquared := math.Pow(playerRatingDeviation, 2)
	var B float64
	if deltaSquared > rdSquared+variance {
		B = math.Log(deltaSquared - rdSquared - variance)
	} else {
		k := 1
		B = a - (float64(k) * tau)
		for iterativeHelper(B, deltaSquared, rdSquared, variance, a, tau) < 0 {
			k = k + 1
			B = a - (float64(k) * tau)
		}

	}

	fA := iterativeHelper(A, deltaSquared, rdSquared, variance, a, tau)
	fB := iterativeHelper(B, deltaSquared, rdSquared, variance, a, tau)
	i := 0
	for math.Abs(B-A) > ConvergenceTolerance && i < iterationMaximum {
		C := A + (((A - B) * fA) / (fB - fA))
		fC := iterativeHelper(C, deltaSquared, rdSquared, variance, a, tau)
		if fB*fC < 0 {
			A = B
			fA = fB
		} else {
			fA = fA / 2
		}
		B = C
		fB = fC
		i++
	}

	newPlayerVolatility := math.Min(p.MaximumVolatility, math.Exp(A/2))

	// Step 6 of the Glicko-225 algorithm
	newPlayerRatingDeviation := math.Sqrt(rdSquared + ((float64(secondsSinceLastGame) / float64(p.RatingPeriodSeconds)) * math.Pow(newPlayerVolatility, 2)))

	// Step 7 of the Glicko-225 algorithm
	newPlayerRatingDeviation = 1 / math.Sqrt((1/math.Pow(newPlayerRatingDeviation, 2))+1/variance)
	newPlayerRating := playerRating + (math.Pow(newPlayerRatingDeviation, 2) * improvement)

	// Step 8 of the Glicko-225 algorithm
	newPlayerRating = p.Conversion*newPlayerRating + p.InitialRating
	newPlayerRatingDeviation = newPlayerRatingDeviation * p.Conversion
	newPlayerRatingDeviation = math.Max(math.Min(newPlayerRatingDeviation, p.MaximumRatingDeviation), p.MinimumRatingDeviation)

	return newPlayerRating, newPlayerRatingDeviation, newPlayerVolatility
}

func (p Params) toGlicko(unscaledRating float64) float64 {
	return (unscaledRating - p.InitialRating) / p.Conversion
}

func adjustWinBoost(rating float64) float64 {
	return V1.adjustWinBoost(rating)
}

func (p Params) adjustWinBoost(rating float64) float64 {
	return ((0.5 - p.MinimumWinBoost) / (1 + math.Exp(-p.Steepness*(rating-p.WinBoostCenter)))) + p.MinimumWinBoost
}

func variance(opponentAdjustedRatingDeviation float64, expectedValue float64) float64 {
	return math.Pow(opponentAdjustedRatingDeviation, 2) * expectedValue * (1 - expectedValue)
}

func (p Params) improvement(opponentAdjustedRatingDeviation float64, awb float64, expectedValue float64, spread int) float64 {
	return opponentAdjustedRatingDeviation * ((boundedResult(float64(spread)/((2*float64(p.SpreadScaling))+p.kfunction(awb))+(float64(sign(spread))*awb)) + 0.5) - expectedValue)
}

func boundedResult(result float64) float64 {
	boundedResult := result
	if boundedResult > 0.5 {
		boundedResult = 0.5
	} else if boundedResult < -0.5 {
		boundedResult = -0.5
	}
	return boundedResult
}

func iterativeHelper(x float64, deltaSquared float64, rdSquared float64, variance float64, a float64, tau float64) float64 {
	ex := math.Exp(x)
	return (ex*(deltaSquared-rdSquared-variance-ex))/
		(2*math.Pow(rdSquared+variance+ex, 2)) -
		(x-a)/math.Pow(tau, 2)
}

func adjustedRatingDeviation(ratingDeviation float64) float64 {
	return 1 / math.Sqrt(1+((3*math.Pow(ratingDeviation, 2))/math.Pow(math.Pi, 2)))
}

func expectedValue(playerRating float64, opponentRating float64, opponentAdjustedRatingDeviation float64) float64 {
	return 1 / (1 + math.Exp(-opponentAdjustedRatingDeviation*(playerRating-opponentRating)))
}

func (p Params) kfunction(wb float64) float64 {
	return (float64(4*float64(p.SpreadScaling)) * wb) / (1 - (2 * wb))
}

func sign(spread int) int {
	sign := 1
	if spread < 0 {
		sign = -1
	} else if spread == 0 {
		sign = 0
	}
	return sign
}
