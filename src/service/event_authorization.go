package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
)

func isStaff(requester *domain.UserDomain) bool {
	return roleAtLeast(requester.Role, domain.UserRoleModerator)
}

func canManageEvent(requester *domain.UserDomain, event *domain.EventDomain) bool {
	if isStaff(requester) || event.Owner.Id == requester.Id {
		return true
	}
	return event.Community != nil && event.Community.Owner.Id == requester.Id
}

func forbiddenManageEvent() *rest_err.RestErr {
	return rest_err.NewForbiddenError("only the event owner, the community owner or moderators can manage this event")
}
