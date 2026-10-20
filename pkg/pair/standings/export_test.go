package standings

// SetSimPairedRoundHook sets simPairedRoundHook for tests.
func SetSimPairedRoundHook(hook func(pairs [][2]int)) {
	simPairedRoundHook = hook
}
