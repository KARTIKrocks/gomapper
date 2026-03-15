// Basic demonstrates simple struct-to-DTO mapping with gomapper.
//
// Run: go generate ./...
package basic

//go:generate gomapper -src User -dst UserDTO

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
