package community

import (
	"time"

	"github.com/ajuda-dev/backend/src/config/quota"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	"github.com/ajuda-dev/backend/src/service/identity"
)

type CommunityUserService interface {
	JoinCommunity(communityId string, userId string) (*communitydomain.CommunityUserDomain, *rest_err.RestErr)
	LeaveCommunity(communityId string, userId string) *rest_err.RestErr
	GetCommunityMembers(communityId string, requesterId string, page int, limit int) (*communitydomain.PageableCommunityMember, *rest_err.RestErr)
	GetUserCommunities(userId string, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr)
}

type communityUserService struct {
	userService             identity.UserService
	communityService        CommunityService
	communityUserRepository communityrepo.CommunityUserRepository
	quotaCfg                quota.Config
	rateLimiter             *quota.HourlyLimiter
}

func NewCommunityUserService(
	userService identity.UserService,
	communityService CommunityService,
	communityUserRepository communityrepo.CommunityUserRepository,
	quotaCfg quota.Config,
	rateLimiter *quota.HourlyLimiter) CommunityUserService {
	return &communityUserService{
		userService:             userService,
		communityService:        communityService,
		communityUserRepository: communityUserRepository,
		quotaCfg:                quotaCfg,
		rateLimiter:             rateLimiter,
	}
}

func (c *communityUserService) JoinCommunity(communityId string, userId string) (*communitydomain.CommunityUserDomain, *rest_err.RestErr) {
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return nil, err
	}
	user, err := identity.AuthenticatedUser(c.userService, userId)
	if err != nil {
		return nil, err
	}
	if err := identity.RequireVerifiedEmail(user); err != nil {
		return nil, err
	}

	rateLimit, rateBypass := quota.LimitForRole(user.Role, c.quotaCfg.RateCommunityJoinPerHour, c.quotaCfg.RateCommunityJoinPerHourModerator)
	if !rateBypass {
		if err := c.rateLimiter.Check(quota.BucketCommunityJoin, user.Id, rateLimit, time.Now(), "too many community joins"); err != nil {
			return nil, err
		}
	}

	memberLimit, memberBypass := quota.LimitForRole(user.Role, c.quotaCfg.MaxCommunityMemberships, c.quotaCfg.MaxCommunityMembershipsModerator)
	if !memberBypass {
		count, countErr := c.communityUserRepository.CountByUserId(user.Id)
		if countErr != nil {
			return nil, countErr
		}
		if count >= int64(memberLimit) {
			return nil, rest_err.NewTooManyRequestsError("community memberships limit reached")
		}
	}

	created, createErr := c.communityUserRepository.Create(&communitydomain.CommunityUserDomain{
		CommunityId: communityId,
		UserId:      user.Id,
	})
	if createErr != nil {
		return nil, createErr
	}
	if !rateBypass {
		c.rateLimiter.Record(quota.BucketCommunityJoin, user.Id, time.Now())
	}
	return created, nil
}

func (c *communityUserService) LeaveCommunity(communityId string, userId string) *rest_err.RestErr {
	if _, err := c.communityService.GetCommunityById(communityId); err != nil {
		return err
	}
	return c.communityUserRepository.DeleteByCommunityAndUser(communityId, userId)
}

func (c *communityUserService) GetCommunityMembers(communityId string, requesterId string, page int, limit int) (*communitydomain.PageableCommunityMember, *rest_err.RestErr) {
	requester, err := identity.AuthenticatedUser(c.userService, requesterId)
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
		identity.ApplyVisibilityFilter(member.User, requester)
	}
	return result, nil
}

func (c *communityUserService) GetUserCommunities(userId string, page int, limit int) (*communitydomain.PageableCommunity, *rest_err.RestErr) {
	return c.communityUserRepository.FindCommunitiesByUser(userId, page, limit)
}
