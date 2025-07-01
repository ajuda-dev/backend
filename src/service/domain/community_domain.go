package domain


type CommunityDomain struct {
	Id          uint 
	Name        string
	Description string
	Owner       UserDomain
	Address     AddressDomain
}	