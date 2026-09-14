package service

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
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

func isCommunityOwner(requester *domain.UserDomain, communityId string, communityRepository repository.CommunityRepository) bool {
	if communityId == "" {
		return false
	}
	community, err := communityRepository.FindById(communityId)
	if err != nil {
		return false
	}
	return community.Owner.Id == requester.Id
}

func canApproveEvent(requester *domain.UserDomain, event *domain.EventDomain) bool {
	if isStaff(requester) {
		return true
	}
	return event.Community != nil && event.Community.Owner.Id == requester.Id
}

func forbiddenApproveEvent() *rest_err.RestErr {
	return rest_err.NewForbiddenError("only the community owner can approve this event")
}

func isEventApproved(event *domain.EventDomain) bool {
	return event.Status == "" || event.Status == domain.EventStatusApproved
}

func isEventVisible(requester *domain.UserDomain, event *domain.EventDomain) bool {
	if isEventApproved(event) {
		return true
	}
	return canManageEvent(requester, event)
}

func eventNotApprovedError() *rest_err.RestErr {
	return rest_err.NewBadRequestValidationError(
		"Invalid participation data",
		[]rest_err.Causes{{
			Field:   "event_id",
			Message: "event is not approved yet",
		}})
}
