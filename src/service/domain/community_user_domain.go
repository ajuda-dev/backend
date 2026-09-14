package domain

type CommunityUserDomain struct {
	Id          string
	CommunityId string
	UserId      string
	User        *UserDomain
}

type PageableCommunityMember struct {
	HasNext bool
	Data    []*CommunityUserDomain
}
