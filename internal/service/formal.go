package service

import (
	"github.com/UnendingLoop/WarehouseControl/internal/mwauthlog"
	"github.com/UnendingLoop/WarehouseControl/internal/policy"
	"github.com/UnendingLoop/WarehouseControl/internal/repository"
)

type WHCService struct {
	repo       repository.WHCRepo
	policy     PolicyChecker
	jwtManager JWTManager
}

func NewWHBService(ebrepo repository.WHCRepo, jwt JWTManager) *WHCService {
	return &WHCService{repo: ebrepo, policy: policy.PolicyChecker{}, jwtManager: jwt}
}

type PolicyChecker interface {
	AccessToDelete(role string) bool
	AccessToCreate(role string) bool
	AccessToUpdate(role string) bool
	AccessToGetHistory(role string) bool
	AccessToGetItems(role string) bool
	AccessToSeeDeleted(role string) bool
	IsCorrectRole(role string) bool
}

type JWTManager interface {
	Generate(uid int, userName string, role string) (string, error)
	Parse(tokenStr string) (*mwauthlog.Claims, error)
}
