package testdata

func statementScope() {
	//nolint:nilinterface,inspectprobe
	main(nil)
	main(nil) // want "nil passed to interface parameter"
}

func explicitBlockScope() {
	//nolint:nilinterface,inspectprobe
	{
		main(nil)
		if true {
			main(nil)
		}
	}
	main(nil) // want "nil passed to interface parameter"
}

func unattachedDirective() {
	//nolint:nilinterface,inspectprobe

	main(nil) // want "nil passed to interface parameter"
}
