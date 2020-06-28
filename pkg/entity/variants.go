package entity

// Variants, time controls, etc.

type Variant string
type TimeControl string

const (
	VarClassic   Variant = "classic"
	VarAWorth100         = "a-is-worth-100"
	VarDogworms          = "dogworms" // OMGWords scrambled = dogworms?
	VarSuper             = "superomg"
)

const (
	TCRegular    TimeControl = "regular"    // > 14/0
	TCRapid                  = "rapid"      // 6/0 to <= 14/0
	TCBlitz                  = "blitz"      // > 2/0 to < 6/0
	TCUltraBlitz             = "ultrablitz" // 2/0 and under
	TCCorres                 = "corres"
)

const (
	// Cutoffs in seconds for different time controls.
	CutoffUltraBlitz = 2 * 60
	CutoffBlitz      = 6 * 60
	CutoffRapid      = 14 * 60
)
