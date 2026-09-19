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

func canRescheduleEvent(requester *userdomain.UserDomain, event *eventdomain.EventDomain, isInviteParticipant bool) bool {
	if canManageEvent(requester, event) {
		return true
	}
	return isInviteParticipant
}

func isInviteRole(role string) bool {
	return role == eventdomain.RoleSpeaker ||
		role == eventdomain.RoleHost ||
		role == eventdomain.RoleMentor ||
		role == eventdomain.RoleMentee
}

func isActiveInviteParticipant(participants []*eventdomain.EventUserDomain, userId string, category string) bool {
	for _, participant := range participants {
		if participant == nil || participant.UserId != userId || participant.Status == eventdomain.StatusCancelled {
			continue
		}
		if category == eventdomain.CategoryMentoring {
			return true
		}
		if participant.Role == eventdomain.RoleSpeaker {
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

func isEventPublic(event *eventdomain.EventDomain) bool {
	return event.Visibility == "" || event.Visibility == eventdomain.EventVisibilityPublic
}

func redactEventMeetingLink(event *eventdomain.EventDomain, requester *userdomain.UserDomain, isParticipant bool) {
	if event == nil || requester == nil {
		return
	}
	if canManageEvent(requester, event) || isParticipant {
		return
	}
	event.MeetingLink = ""
}

func isActiveEventParticipant(participants []*eventdomain.EventUserDomain, userId string) bool {
	for _, participant := range participants {
		if participant == nil || participant.UserId != userId {
			continue
		}
		if participant.Status != eventdomain.StatusCancelled {
			return true
		}
	}
	return false
}

func isEventVisible(requester *userdomain.UserDomain, event *eventdomain.EventDomain, isParticipant bool) bool {
	if canManageEvent(requester, event) {
		return true
	}
	if event.Status == eventdomain.EventStatusRejected {
		return false
	}
	if isEventPublic(event) && isEventApproved(event) {
		return true
	}
	return isParticipant
}

func eventNotApprovedError() *rest_err.RestErr {
	return rest_err.NewBadRequestValidationError(
		"Invalid participation data",
		[]rest_err.Causes{{
			Field:   "event_id",
			Message: "event is not approved yet",
		}})
}

func eventNotPublicError() *rest_err.RestErr {
	return rest_err.NewBadRequestValidationError(
		"Invalid participation data",
		[]rest_err.Causes{{
			Field:   "event_id",
			Message: "event is not public yet",
		}})
}
