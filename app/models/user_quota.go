package models

import (
	"encoding/json"
	"time"
)

type UserQuota struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"unique;not null" json:"user_id"`
	TotalLimit    int        `gorm:"default:0;not null" json:"total_limit"`
	UsedTotal     int        `gorm:"default:0;not null" json:"used_total"`
	DailyLimit    int        `gorm:"default:0;not null" json:"daily_limit"`
	UsedToday     int        `gorm:"default:0;not null" json:"used_today"`
	LastResetDate time.Time  `json:"last_reset_date"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (UserQuota) TableName() string {
	return "user_quotas"
}

func (q *UserQuota) RemainingTotal() int {
	if q == nil {
		return 0
	}
	rem := q.TotalLimit - q.UsedTotal
	if rem < 0 {
		return 0
	}
	return rem
}

func (q *UserQuota) RemainingDaily() int {
	if q == nil {
		return 0
	}
	rem := q.DailyLimit - q.UsedToday
	if rem < 0 {
		return 0
	}
	return rem
}

func (q UserQuota) MarshalJSON() ([]byte, error) {
	type Alias UserQuota
	return json.Marshal(&struct {
		Alias
		RemainingTotal int `json:"remaining_total"`
		RemainingDaily int `json:"remaining_daily"`
	}{
		Alias:          Alias(q),
		RemainingTotal: q.RemainingTotal(),
		RemainingDaily: q.RemainingDaily(),
	})
}
