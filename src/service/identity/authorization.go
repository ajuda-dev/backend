package identity

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

var roleRank = map[string]int{
	userdomain.UserRoleUser:      1,
	userdomain.UserRoleModerator: 2,
	userdomain.UserRoleAdmin:     3,
}

func RoleAtLeast(role string, minimum string) bool {
	return roleRank[role] >= roleRank[minimum]
}

func AuthenticatedUser(userService UserService, requesterId string) (*userdomain.UserDomain, *rest_err.RestErr) {
	requester, err := userService.FindById(requesterId)
	if err != nil {
		if err.Code == rest_err.NOT_FOUND {
			return nil, rest_err.NewUnauthorizedError("invalid authenticated user")
		}
		return nil, err
	}
	return requester, nil
}

func RequireVerifiedEmail(user *userdomain.UserDomain) *rest_err.RestErr {
	if user.EmailVerified() {
		return nil
	}
	return rest_err.NewForbiddenError("email is not verified")
}
