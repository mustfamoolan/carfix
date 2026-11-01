package models

import (
	"time"
)

type SearchLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	User         *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	VIN          string    `gorm:"size:17;index;not null" json:"vin"`
	VehicleTitle string    `gorm:"size:255" json:"vehicle_title"`
	ResultStatus string    `gorm:"size:50" json:"result_status"`
	FetchSource  string    `gorm:"size:50" json:"fetch_source"`
	DurationMs   int64     `json:"duration_ms"`
	IPAddress    string    `gorm:"size:50" json:"ip_address"`
	UserAgent    string    `gorm:"type:text" json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}

func (SearchLog) TableName() string {
	return "search_logs"
}

