package service

import "github.com/ajuda-dev/backend/src/service/domain"

func applyVisibilityFilter(target *domain.UserDomain, requester *domain.UserDomain) {
	if target == nil || requester == nil {
		return
	}
	target.Password = ""
	if target.Id == requester.Id || requester.Role == domain.UserRoleAdmin {
		return
	}
	visible := target.ConfigVisibility.VisibleToOthers()
	if _, shared := visible[domain.VisibilityKeyEmail]; !shared {
		target.Email = ""
	}
	target.ConfigVisibility = visible
}
