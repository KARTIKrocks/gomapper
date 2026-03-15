package advanced

// --- map:"-" skip ---

type SkipSrc struct {
	Name   string
	Secret string `map:"-"`
}

type SkipDst struct {
	Name     string
	Internal string `map:"-"`
	Secret   string
}

// --- Pointer fields ---

type PtrSrc struct {
	Name *string
	Age  *int
}

type PtrDst struct {
	Name string
	Age  int64
}

type AddrSrc struct {
	Name string
}

type AddrDst struct {
	Name *string
}

// --- Slice fields ---

type Item struct {
	ID   int
	Name string
}

type ItemDTO struct {
	ID   int
	Name string
}

type SliceSrc struct {
	Title string
	Items []Item
}

type SliceDst struct {
	Title string
	Items []ItemDTO
}

// --- Nested struct mapping ---

type Address struct {
	Street string
	City   string
}

type AddressDTO struct {
	Street string
	City   string
}

type Order struct {
	ID   int
	Addr Address
}

type OrderDTO struct {
	ID   int
	Addr AddressDTO
}

// --- Case-insensitive matching ---

type CISrc struct {
	UserName string
	EMail    string
}

type CIDst struct {
	Username string
	Email    string
}
