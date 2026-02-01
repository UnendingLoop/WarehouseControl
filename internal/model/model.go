// Package model holds shared data-structures of the app
package model

import (
	"context"
	"encoding/json"
	"time"
)

//================ Пользователь и роли ========================

type User struct {
	ID        int       `json:"id,omitempty" db:"id"`
	UserName  string    `json:"username" db:"username"`
	Role      string    `json:"role" db:"role"`
	PassHash  string    `json:"password" db:"pass_hash"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleViewer  = "viewer"
	RoleAuditor = "auditor"
)

var RolesMap = map[string]struct{}{RoleAdmin: {}, RoleManager: {}, RoleViewer: {}, RoleAuditor: {}}

// =============== Товар ========================

type Item struct {
	ID              int        `json:"id" db:"id"`
	Title           string     `json:"title" db:"title"`
	Description     string     `json:"description,omitempty" db:"description"`
	Price           int64      `json:"price" db:"price"`
	Visible         bool       `json:"visible" db:"visible"`
	AvailableAmount int        `json:"available_amount" db:"available_amount"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	UpdatedBy       string     `json:"-" db:"updated_by"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
type ItemUpdate struct {
	ID              int     `json:"id" db:"id"`
	Title           *string `json:"title" db:"title"`
	Description     *string `json:"description,omitempty" db:"description"`
	Price           *int64  `json:"price" db:"price"`
	Visible         *bool   `json:"visible" db:"visible"`
	AvailableAmount *int    `json:"available_amount" db:"available_amount"`
	UpdatedBy       string  `json:"-" db:"updated_by"`
}

const (
	ItemsOrderByID           = "id"
	ItemsOrderByTitle        = "title"
	ItemsOrderByPrice        = "price"
	ItemsOrderByAvailability = "availability"
	ItemsOrderByVisibility   = "visibility"
)

var OrderByItemsMap = map[string]struct{}{
	ItemsOrderByID:           {},
	ItemsOrderByTitle:        {},
	ItemsOrderByPrice:        {},
	ItemsOrderByAvailability: {},
	ItemsOrderByVisibility:   {},
}

// ========== История изменений ================

type ItemHistory struct {
	ID        int             `json:"id" db:"id"`
	ItemID    int             `json:"item_id" db:"item_id"`
	Version   int             `json:"version" db:"version"`
	Action    string          `json:"action" db:"action"`
	ChangedAt time.Time       `json:"changed_at" db:"changed_at"`
	ChangedBy string          `json:"changed_by" db:"changed_by"`
	OldData   json.RawMessage `json:"old" db:"old_data"`
	NewData   json.RawMessage `json:"new" db:"new_data"`
}

type RequestParam struct {
	OrderBy   *string    `form:"order_by"`
	ASC       bool       `form:"asc"`
	DESC      bool       `form:"desc"`
	StartTime *time.Time `form:"from"`
	EndTime   *time.Time `form:"to"`
	Page      *int       `form:"page"`
	Limit     *int       `form:"limit"`
}

const (
	HistoryOrderByID      = "id"
	HistoryOrderByAction  = "action"
	HistoryOrderByVersion = "version"
	HistoryOrderByActor   = "actor"
)

var OrderByHistoryMap = map[string]struct{}{
	HistoryOrderByID:      {},
	HistoryOrderByAction:  {},
	HistoryOrderByVersion: {},
	HistoryOrderByActor:   {},
}

//====================================

func RequestIDFromCtx(ctx context.Context) string {
	if v := ctx.Value("request_id"); v != nil {
		return v.(string)
	}
	return ""
}
