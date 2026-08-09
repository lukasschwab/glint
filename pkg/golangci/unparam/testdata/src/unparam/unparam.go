package unparam

func unparamUnused(a, b uint) uint { // want "unparamUnused - b is unused"
	a++
	return a
}
