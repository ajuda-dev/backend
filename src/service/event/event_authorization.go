package event

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	eventdomain "github.com/ajuda-dev/backend/src/service/event/domain"
	"github.com/ajuda-dev/backend/src/service/identity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

func isStaff(requester *userdomain.UserDomain) bool {
	return identity.RoleAtLeast(requester.Role, userdomain.UserRoleModerator)
}

func canManageEvent(requester *userdomain.UserDomain, event *eventdomain.EventDomain) bool {
	if isStaff(requester) || event.Owner.Id == requester.Id {
		return true
	}
	return event.Community != nil && event.Community.Owner.Id == requester.Id
}

func canRescheduleEvent(requester *userdomain.UserDomain, event *eventdomain.EventDomain, isMentoringParticipant bool) bool {
	if canManageEvent(requester, event) {
		return true
	}
	return event.Category == eventdomain.CategoryMentoring && isMentoringParticipant
}

func isActiveMentoringParticipant(participants []*eventdomain.EventUserDomain, userId string) bool {
	for _, participant := range participants {
		if participant != nil && participant.UserId == userId && participant.Status != eventdomain.StatusCancelled {
			return true
		}
	}
	return false
}

func forbiddenManageEvent() *rest_err.RestErr {
	return rest_err.NewForbiddenError("only the event owner, the community owner or moderators can manage this event")
}

func forbiddenRescheduleEvent() *rest_err.RestErr {
	return rest_err.NewForbiddenError("only the event owner, the invited participant, the community owner or moderators can reschedule this event")
}

func isCommunityOwner(requester *userdomain.UserDomain, communityId string, communityRepository communityrepo.CommunityRepository) bool {
	if communityId == "" {
		return false
	}
	community, err := communityRepository.FindById(communityId)
	if err != nil {
		return false
	}
	return community.Owner.Id == requester.Id
}

func canApproveEvent(requester *userdomain.UserDomain, event *eventdomain.EventDomain) bool {
	if isStaff(requester) {
		return true
	}
	return event.Community != nil && event.Community.Owner.Id == requester.Id
}

func forbiddenApproveEvent() *rest_err.RestErr {
	return rest_err.NewForbiddenError("only the community owner can approve this event")
}

func isEventApproved(event *eventdomain.EventDomain) bool {
	return event.Status == "" || event.Status == eventdomain.EventStatusApproved
}

func isEventVisible(requester *userdomain.UserDomain, event *eventdomain.EventDomain) bool {
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
