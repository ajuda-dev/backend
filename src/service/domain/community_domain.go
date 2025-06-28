package domain


type CommunityDomain struct {
	Id          string 
	Name        string
	Description string
	Owner       UserDomain
}	