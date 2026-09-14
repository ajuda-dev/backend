package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
)

type CommunityUserService interface {
	JoinCommunity(communityId string, userId string) (*domain.CommunityUserDomain, *rest_err.RestErr)
	LeaveCommunity(communityId string, userId string) *rest_err.RestErr
	GetCommunityMembers(communityId string, requesterId string, page int, limit int) (*domain.PageableCommunityMember, *rest_err.RestErr)
	GetUserCommunities(userId string, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr)
}

type communityUserService struct {
	userService             UserService
	communityService        CommunityService
	communityUserRepository repository.CommunityUserRepository
}

func NewCommunityUserService(
	userService UserService,
	communityService CommunityService,
	communityUserRepository repository.CommunityUserRepository) CommunityUserService {
	return &communityUserService{
		userService:             userService,
		communityService:        communityService,
		communityUserRepository: communityUserRepository,
	}
}

func (c *communityUserService) JoinCommunity(communityId string, userId string) (*domain.CommunityUserDomain, *rest_err.RestErr) {
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return nil, err
	}
	if _, err := c.userService.FindById(userId); err != nil {
		if err.Code == rest_err.NOT_FOUND {
			return nil, rest_err.NewUnauthorizedError("invalid authenticated user")
		}
		return nil, err
	}
	return c.communityUserRepository.Create(&domain.CommunityUserDomain{
		CommunityId: communityId,
		UserId:      userId,
	})
}

func (c *communityUserService) LeaveCommunity(communityId string, userId string) *rest_err.RestErr {
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return err
	}
	return c.communityUserRepository.DeleteByCommunityAndUser(communityId, userId)
}

func (c *communityUserService) GetCommunityMembers(communityId string, requesterId string, page int, limit int) (*domain.PageableCommunityMember, *rest_err.RestErr) {
	requester, err := authenticatedUser(c.userService, requesterId)
	if err != nil {
		return nil, err
	}
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return nil, err
	}
	result, err := c.communityUserRepository.FindMembersByCommunity(communityId, page, limit)
	if err != nil {
		return nil, err
	}
	for _, member := range result.Data {
		applyVisibilityFilter(member.User, requester)
	}
	return result, nil
}

func (c *communityUserService) GetUserCommunities(userId string, page int, limit int) (*domain.PageableCommunity, *rest_err.RestErr) {
	return c.communityUserRepository.FindCommunitiesByUser(userId, page, limit)
}
