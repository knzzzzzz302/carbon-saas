package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	TenantPlanStarter    = "starter"
	TenantPlanGrowth     = "growth"
	TenantPlanEnterprise = "enterprise"

	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleMember  = "member"
	RoleViewer  = "viewer"
)

type Tenant struct {
	gorm.Model
	Name         string            `json:"name"`
	Slug         string            `gorm:"uniqueIndex" json:"slug"`
	Plan         string            `json:"plan"`
	Status       string            `json:"status"`
	FeatureFlags datatypes.JSONMap `json:"feature_flags"`
	Metadata     datatypes.JSONMap `json:"metadata"`
}

type Membership struct {
	gorm.Model
	TenantID uint   `json:"tenant_id"`
	UserID   uint   `json:"user_id"`
	Role     string `json:"role"`
	Status   string `json:"status"`

	Tenant Tenant `gorm:"constraint:OnDelete:CASCADE"`
	User   User   `gorm:"constraint:OnDelete:CASCADE"`
}

type ServiceAccount struct {
	gorm.Model
	TenantID    uint              `json:"tenant_id"`
	Name        string            `json:"name"`
	ClientID    string            `gorm:"uniqueIndex" json:"client_id"`
	SecretHash  string            `json:"-"`
	Permissions datatypes.JSONMap `json:"permissions"`
	LastUsedAt  *time.Time        `json:"last_used_at"`

	Tenant Tenant `gorm:"constraint:OnDelete:CASCADE"`
}
