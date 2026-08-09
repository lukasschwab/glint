//nolint:nilinterface,inspectprobe
package testdata

var topLevelFileScope = main(nil)

func alsoFileScoped() {
	var _ = main(nil)
}
