package basic

type User struct {
	ID    int
	Name  string
	Email string
	Age   int
}

type UserDTO struct {
	ID   int
	Name string
	Age  int64
}

type Address struct {
	Street string
	City   string
}

type Person struct {
	Name    string
	Address Address
}

type PersonFlat struct {
	Name   string
	Street string `map:"Address.Street"`
	City   string `map:"Address.City"`
}

// MyString is a non-struct named type (used by loader tests).
type MyString string

type TaggedSource struct {
	FullName string `map:"Name"`
	Score    int
}

type TaggedDest struct {
	Name  string
	Score int
}
