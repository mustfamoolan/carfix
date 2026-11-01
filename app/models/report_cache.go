package models

import (
	"time"
)

type ReportCache struct {
	VIN              string    `gorm:"primaryKey;size:17" json:"vin"`
	VehicleTitle     string    `gorm:"size:255" json:"vehicle_title"`
	RawHtmlEn        string    `gorm:"type:text" json:"raw_html_en"`
	TranslatedHtmlAr string    `gorm:"type:text" json:"translated_html_ar"`
	SummaryJson      string    `gorm:"type:jsonb" json:"summary_json"`
	FetchedAt        time.Time `json:"fetched_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (ReportCache) TableName() string {
	return "report_caches"
}

