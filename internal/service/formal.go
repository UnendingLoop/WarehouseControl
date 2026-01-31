package service

import (
	"github.com/UnendingLoop/WarehouseControl/internal/mwauthlog"
	"github.com/UnendingLoop/WarehouseControl/internal/policy"
	"github.com/UnendingLoop/WarehouseControl/internal/repository"
)

type WHCService struct {
	repo       repository.WHCRepo
	policy     PolicyChecker
	jwtManager *mwauthlog.JWTManager
}

func NewWHBService(ebrepo repository.WHCRepo, jwt *mwauthlog.JWTManager) *WHCService {
	return &WHCService{repo: ebrepo, policy: policy.PolicyChecker{}, jwtManager: jwt}
}

type PolicyChecker interface {
	AccessToDelete(role string) bool
	AccessToCreate(role string) bool
	AccessToUpdate(role string) bool
	AccessToGetHistory(role string) bool
	AccessToGetItems(role string) bool
	AccessToSeeDeleted(role string) bool
}
