package testdata

var unrelated = 0 //nolint:decl:nilinterface,inspectprobe

func notAttachedToTrailingComment() {
	var _ = main(nil) // want "nil passed to interface parameter"
}

// This directive shares a documentation comment group.
//
//nolint:nilinterface,inspectprobe
func attachedToDocumentationGroup() {
	var _ = main(nil)
}
