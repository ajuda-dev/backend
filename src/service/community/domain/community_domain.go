package domain

import (
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

const (
	LinkKeyGithub    = "github"
	LinkKeyLinkedin  = "linkedin"
	LinkKeyOtherlink = "otherlink"
	LinkKeyPhoto     = "photo"
)

type CommunityLink struct {
	Value string `json:"value"`
}

type CommunityLinks map[string]CommunityLink

type CommunityDomain struct {
	Id               string
	Name             string
	Description      string
	Owner            userdomain.UserDomain
	Address          addressdomain.AddressDomain
	ConfigVisibility CommunityLinks
}
