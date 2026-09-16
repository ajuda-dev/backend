package identity

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

func ApplyVisibilityFilter(target *userdomain.UserDomain, requester *userdomain.UserDomain) {
	if target == nil || requester == nil {
		return
	}
	target.Password = ""
	if target.Id == requester.Id || requester.Role == userdomain.UserRoleAdmin {
		return
	}
	visible := target.ConfigVisibility.VisibleToOthers()
	if _, shared := visible[userdomain.VisibilityKeyEmail]; !shared {
		target.Email = ""
	}
	target.ConfigVisibility = visible
}
