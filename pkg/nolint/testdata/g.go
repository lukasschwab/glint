package testdata

var _ = main(nil) //nolint:nilinterface,inspectprobe
var _ = main(nil) // want "nil passed to interface parameter"
var _ = main(nil) //nolint:other // want "nil passed to interface parameter"

//nolint:nilinterface,inspectprobe
func declarationScope() {
	var _ = main(nil)
}

func unsuppressedDeclaration() {
	var _ = main(nil) // want "nil passed to interface parameter"
}

//nolint:decl:nilinterface,inspectprobe
var declarationValue = main(nil)

var explicitLine = main(nil) //nolint:line:nilinterface,inspectprobe

var (
	//nolint:decl:nilinterface,inspectprobe
	valueSpec  = main(nil)
	otherValue = main(nil) // want "nil passed to interface parameter"
)
