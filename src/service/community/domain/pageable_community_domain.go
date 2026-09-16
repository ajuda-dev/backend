package domain

type PageableCommunity struct {
	HasNext bool
	Data    []*CommunityDomain
}
