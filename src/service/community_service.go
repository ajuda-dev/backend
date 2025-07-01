package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
)

type CommunityService interface {
	CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr)
}

type communityService struct {
	userService         UserService
	addressService      AddressService
	communityRepository repository.CommunityRepository
}



func NewCommunityService(userService UserService,
	addressService AddressService,
	communityRepository repository.CommunityRepository) CommunityService {
	return &communityService{
		userService:         userService,
		addressService:      addressService,
		communityRepository: communityRepository,
	}
}


func (c *communityService) CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr) {
	user, err_u := c.userService.FindById(community.Owner.Id)
	if err_u != nil {
		return &domain.CommunityDomain{}, err_u
	}
	community.Owner = *user
	address, err_a := c.addressService.GetAddressById(community.Address.Id)
	if err_a != nil {
		return &domain.CommunityDomain{}, err_a
	}
	community.Address = *address
	return c.communityRepository.CreateCommunity(community)
}
