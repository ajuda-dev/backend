package domain

type AddressDomain struct {
	Id         string
	City       string
	State      string
	Street     string
	ZipCode    string
	Number     string
	Complement string
}

type PageableAddress struct {
	HasNext bool
	Data    []*AddressDomain
}
