package testdata

var explicitFileScope = main(nil)

//nolint:file:nilinterface,inspectprobe
func explicitFileScopeAlsoSuppressesEarlierDiagnostics() {
	var _ = main(nil)
}
