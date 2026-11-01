package models

import (
	"time"
)

type User struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	FullName  string     `gorm:"size:255;not null" json:"full_name"`
	Phone     string     `gorm:"size:50" json:"phone"`
	Username  string     `gorm:"size:100;unique;not null" json:"username"`
	Password  string     `gorm:"size:255;not null" json:"-"`
	Role      string     `gorm:"size:50;default:'user';not null" json:"role"`
	IsActive  bool       `gorm:"default:true;not null" json:"is_active"`
	Quota     *UserQuota `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"quota,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
