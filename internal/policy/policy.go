package policy

import "github.com/UnendingLoop/WarehouseControl/internal/model"

type PolicyChecker struct{}

func (pc PolicyChecker) AccessToDelete(role string) bool {
	return role == model.RoleAdmin
}

func (pc PolicyChecker) AccessToCreate(role string) bool {
	if role == model.RoleManager || role == model.RoleAdmin {
		return true
	}
	return false
}

func (pc PolicyChecker) AccessToUpdate(role string) bool {
	if role == model.RoleManager || role == model.RoleAdmin {
		return true
	}
	return false
}

func (pc PolicyChecker) AccessToGetHistory(role string) bool {
	if role == model.RoleAuditor || role == model.RoleAdmin {
		return true
	}
	return false
}

func (pc PolicyChecker) AccessToGetItems(role string) bool {
	_, ok := model.RolesMap[role]
	return ok
}

func (pc PolicyChecker) AccessToSeeDeleted(role string) bool {
	if role == model.RoleAuditor || role == model.RoleAdmin {
		return true
	}
	return false
}
