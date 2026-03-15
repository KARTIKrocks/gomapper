package embedded

type Base struct {
	ID        int
	CreatedAt string
}

type User struct {
	Base
	Name  string
	Email string
}

type UserDTO struct {
	ID   int
	Name string
}
