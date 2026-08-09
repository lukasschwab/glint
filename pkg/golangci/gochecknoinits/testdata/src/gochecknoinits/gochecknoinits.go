package gochecknoinits

func init() {} // want "don't use `init` function"

type example struct{}

func (example) init() {}
