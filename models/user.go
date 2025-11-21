package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name            string `json:"name"`
	Email           string `gorm:"uniqueIndex" json:"email"`
	Password        string `json:"-"`
	DefaultTenantID uint   `json:"default_tenant_id"`
	Timezone        string `json:"timezone"`
	Locale          string `json:"locale"`

	Memberships []Membership `json:"memberships"`
}
