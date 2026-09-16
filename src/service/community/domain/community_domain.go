package domain

import (
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type CommunityDomain struct {
	Id          string
	Name        string
	Description string
	Owner       userdomain.UserDomain
	Address     addressdomain.AddressDomain
}
