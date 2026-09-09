package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
)

var roleRank = map[string]int{
	domain.UserRoleUser:      1,
	domain.UserRoleModerator: 2,
	domain.UserRoleAdmin:     3,
}

func roleAtLeast(role string, minimum string) bool {
	return roleRank[role] >= roleRank[minimum]
}

func authenticatedUser(userService UserService, requesterId string) (*domain.UserDomain, *rest_err.RestErr) {
	requester, err := userService.FindById(requesterId)
	if err != nil {
		if err.Code == rest_err.NOT_FOUND {
			return nil, rest_err.NewUnauthorizedError("invalid authenticated user")
		}
		return nil, err
	}
	return requester, nil
}
