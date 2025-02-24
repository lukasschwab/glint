package a

type MyInterface interface{}

func main(mi MyInterface) {}

func test() {
	main(nil) // want "nil passed to interface parameter"
}
