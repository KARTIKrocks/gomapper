// Advanced demonstrates nested structs, slices, pointer fields,
// struct tags, and bidirectional mapping with gomapper.
//
// Run: go generate ./...
package advanced

//go:generate gomapper -pairs Address:AddressDTO,Order:OrderDTO -bidirectional -nil-safe

type Address struct {
	Street string
	City   string
	Zip    *string
}

type AddressDTO struct {
	Street string
	City   string
	Zip    string
}

type Order struct {
	ID       int
	Customer string `map:"Name"`
	Addr     Address
	Items    []Item
}

type OrderDTO struct {
	ID    int
	Name  string
	Addr  AddressDTO
	Items []ItemDTO
}

type Item struct {
	SKU   string
	Name  string
	Price int
}

type ItemDTO struct {
	SKU   string
	Name  string
	Price int64
}
