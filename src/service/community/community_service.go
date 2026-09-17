package community

import (
	"time"

	"github.com/ajuda-dev/backend/src/config/quota"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	"github.com/ajuda-dev/backend/src/service/address"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	communityvalidator "github.com/ajuda-dev/backend/src/service/community/validator"
	"github.com/ajuda-dev/backend/src/service/identity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type CommunityService interface {
	CreateCommunity(community *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr)
	GetCommunityById(id string) (*communitydomain.CommunityDomain, *rest_err.RestErr)
	GetAll(filter communityrepo.CommunityFilter, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr)
	UpdateCommunity(id string, requesterId string, changes *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr)
	DeleteCommunity(id string, requesterId string) *rest_err.RestErr
}

type communityService struct {
	userService             identity.UserService
	addressService          address.AddressService
	communityRepository     communityrepo.CommunityRepository
	communityUserRepository communityrepo.CommunityUserRepository
	communityValidator      communityvalidator.CommunityValidator
	quotaCfg                quota.Config
	rateLimiter             *quota.HourlyLimiter
}

func NewCommunityService(userService identity.UserService,
	addressService address.AddressService,
	communityRepository communityrepo.CommunityRepository,
	communityUserRepository communityrepo.CommunityUserRepository,
	communityValidator communityvalidator.CommunityValidator,
	quotaCfg quota.Config,
	rateLimiter *quota.HourlyLimiter) CommunityService {
	return &communityService{
		userService:             userService,
		addressService:          addressService,
		communityRepository:     communityRepository,
		communityUserRepository: communityUserRepository,
		communityValidator:      communityValidator,
		quotaCfg:                quotaCfg,
		rateLimiter:             rateLimiter,
	}
}

func (c *communityService) CreateCommunity(community *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	err := c.communityValidator.ValidatorRegisterCommunity(*community)
	if err != nil {
		return &communitydomain.CommunityDomain{}, err
	}
	rc, err_rc := c.communityRepository.FindByName(community.Name)

	if err_rc != nil && err_rc.Code != 404 {
		return rc, err_rc
	}

	if rc != nil {
		return &communitydomain.CommunityDomain{},
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
		return &communitydomain.CommunityDomain{}, err_u
	}
	if err := identity.RequireVerifiedEmail(user); err != nil {
		return &communitydomain.CommunityDomain{}, err
	}
	community.Owner = *user

	rateLimit, rateBypass := quota.LimitForRole(user.Role, c.quotaCfg.RateCommunityCreatePerHour, c.quotaCfg.RateCommunityCreatePerHourModerator)
	if !rateBypass {
		if err := c.rateLimiter.Check(quota.BucketCommunityCreate, user.Id, rateLimit, time.Now(), "too many community creations"); err != nil {
			return &communitydomain.CommunityDomain{}, err
		}
	}

	ownedLimit, ownedBypass := quota.LimitForRole(user.Role, c.quotaCfg.MaxOwnedCommunities, c.quotaCfg.MaxOwnedCommunitiesModerator)
	if !ownedBypass {
		count, countErr := c.communityRepository.CountByOwnerId(user.Id)
		if countErr != nil {
			return &communitydomain.CommunityDomain{}, countErr
		}
		if count >= int64(ownedLimit) {
			return &communitydomain.CommunityDomain{}, rest_err.NewTooManyRequestsError("owned communities limit reached")
		}
	}

	address, err_a := c.addressService.GetAddressById(community.Address.Id)
	if err_a != nil {
		if err_a.Code == 404 {
			return &communitydomain.CommunityDomain{},
				rest_err.NewBadRequestValidationError("Invalid community data", []rest_err.Causes{
					{
						Field:   "address_id",
						Message: "address_id is not valid, not found this address",
					},
				})
		}
		return &communitydomain.CommunityDomain{}, err_a
	}
	community.Address = *address
	created, createErr := c.communityRepository.CreateCommunity(community)
	if createErr != nil {
		return &communitydomain.CommunityDomain{}, createErr
	}
	if !rateBypass {
		c.rateLimiter.Record(quota.BucketCommunityCreate, user.Id, time.Now())
	}
	return created, nil
}

func (c *communityService) GetCommunityById(id string) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	return c.communityRepository.FindById(id)
}

func (c *communityService) GetAll(filter communityrepo.CommunityFilter, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr) {
	return c.communityRepository.FindAll(filter, page, limit)
}

func (c *communityService) UpdateCommunity(id string, requesterId string, changes *communitydomain.CommunityDomain) (*communitydomain.CommunityDomain, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(c.userService, requesterId)
	if err != nil {
		return nil, err
	}
	community, err := c.GetCommunityById(id)
	if err != nil {
		return nil, err
	}
	if !identity.RoleAtLeast(requester.Role, userdomain.UserRoleModerator) && community.Owner.Id != requester.Id {
		return nil, rest_err.NewForbiddenError("only the community owner can update this community")
	}
	if err := c.communityValidator.ValidateUpdateCommunity(*changes); err != nil {
		return nil, err
	}
	if changes.Name != "" && changes.Name != community.Name {
		existing, findErr := c.communityRepository.FindByName(changes.Name)
		if findErr != nil && findErr.Code != rest_err.NOT_FOUND {
			return nil, findErr
		}
		if existing != nil && existing.Id != id {
			return nil, rest_err.NewBadRequestValidationError(
				"Invalid community data",
				[]rest_err.Causes{{
					Field:   "name",
					Message: "already has a community with this name",
				}})
		}
	}
	if changes.Address.Id != "" && changes.Address.Id != community.Address.Id {
		address, addrErr := c.addressService.GetAddressById(changes.Address.Id)
		if addrErr != nil {
			if addrErr.Code == rest_err.NOT_FOUND {
				return nil, rest_err.NewBadRequestValidationError(
					"Invalid community data",
					[]rest_err.Causes{{
						Field:   "address_id",
						Message: "address_id is not valid, not found this address",
					}})
			}
			return nil, addrErr
		}
		changes.Address = *address
	}
	return c.communityRepository.Update(id, changes)
}

func (c *communityService) DeleteCommunity(id string, requesterId string) *rest_err.RestErr {
	requester, err := identity.AuthenticatedUser(c.userService, requesterId)
	if err != nil {
		return err
	}
	community, err := c.GetCommunityById(id)
	if err != nil {
		return err
	}
	if !identity.RoleAtLeast(requester.Role, userdomain.UserRoleModerator) {
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
