package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
)

type CommunityService interface {
	CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr)
	GetCommunityById(id string) (*domain.CommunityDomain, *rest_err.RestErr)
	GetAll(address_id string, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr)
	DeleteCommunity(id string, requesterId string) *rest_err.RestErr
}

type communityService struct {
	userService             UserService
	addressService          AddressService
	communityRepository     repository.CommunityRepository
	communityUserRepository repository.CommunityUserRepository
	communityValidator      validator.CommunityValidator
}

func NewCommunityService(userService UserService,
	addressService AddressService,
	communityRepository repository.CommunityRepository,
	communityUserRepository repository.CommunityUserRepository,
	communityValidator validator.CommunityValidator) CommunityService {
	return &communityService{
		userService:             userService,
		addressService:          addressService,
		communityRepository:     communityRepository,
		communityUserRepository: communityUserRepository,
		communityValidator:      communityValidator,
	}
}

func (c *communityService) CreateCommunity(community *domain.CommunityDomain) (*domain.CommunityDomain, *rest_err.RestErr) {
	err := c.communityValidator.ValidatorRegisterCommunity(*community)
	if err != nil {
		return &domain.CommunityDomain{}, err
	}
	rc, err_rc := c.communityRepository.FindByName(community.Name)

	if err_rc != nil && err_rc.Code != 404 {
		return rc, err_rc
	}

	if rc != nil {
		return &domain.CommunityDomain{},
			rest_err.NewBadRequestValidationError("Invalid community data", []rest_err.Causes{
				{
					Field:   "name",
					Message: "already has a community with this name",
				},
			})
	}
	user, err_u := c.userService.FindById(community.Owner.Id)
	if err_u != nil {
		if err_u.Code == 404 {

		}
		return &domain.CommunityDomain{}, err_u
	}
	community.Owner = *user
	address, err_a := c.addressService.GetAddressById(community.Address.Id)
	if err_a != nil {
		if err_a.Code == 404 {
			return &domain.CommunityDomain{},
				rest_err.NewBadRequestValidationError("Invalid community data", []rest_err.Causes{
					{
						Field:   "address_id",
						Message: "address_id is not valid, not found this address",
					},
				})
		}
		return &domain.CommunityDomain{}, err_a
	}
	community.Address = *address
	return c.communityRepository.CreateCommunity(community)
}



func (c *communityService) GetCommunityById(id string) (*domain.CommunityDomain, *rest_err.RestErr) {
	return c.communityRepository.FindById(id)
}

func (c *communityService) GetAll(address_id string, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr) {
	return c.communityRepository.FindAll(address_id, page, limit)
}

func (c *communityService) DeleteCommunity(id string, requesterId string) *rest_err.RestErr {
	requester, err := authenticatedUser(c.userService, requesterId)
	if err != nil {
		return err
	}
	community, err := c.GetCommunityById(id)
	if err != nil {
		return err
	}
	if !roleAtLeast(requester.Role, domain.UserRoleModerator) {
		if community.Owner.Id != requester.Id {
			return rest_err.NewForbiddenError("only the community owner can delete this community")
		}
		memberCount, countErr := c.communityUserRepository.CountByCommunity(id)
		if countErr != nil {
			return countErr
		}
		if memberCount > 0 {
			return rest_err.NewBadRequestValidationError(
				"Invalid delete",
				[]rest_err.Causes{{
					Field:   "members",
					Message: "community has associated members; remove them before deleting",
				}})
		}
	}
	return c.communityRepository.SoftDeleteById(id)
}
