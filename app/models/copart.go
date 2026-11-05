package models

import (
	"time"
)

// CopartLotCache stores vehicle auction details and high-res images fetched from Copart
type CopartLotCache struct {
	LotNumber        string    `gorm:"primaryKey;size:32" json:"lot_number"`
	VIN              string    `gorm:"size:17;index" json:"vin"`
	VehicleTitle     string    `gorm:"size:255" json:"vehicle_title"`
	Year             int       `json:"year"`
	Make             string    `gorm:"size:100" json:"make"`
	Model            string    `gorm:"size:100" json:"model"`
	Odometer         string    `gorm:"size:50" json:"odometer"`
	DamagePrimary    string    `gorm:"size:100" json:"damage_primary"`
	DamageSecondary  string    `gorm:"size:100" json:"damage_secondary"`
	SaleLocation     string    `gorm:"size:150" json:"sale_location"`
	AuctionDate      *time.Time `json:"auction_date"`
	CurrentBid       float64   `json:"current_bid"`
	BuyNowPrice      float64   `json:"buy_now_price"`
	Currency         string    `gorm:"size:10;default:'USD'" json:"currency"`
	ItemStatus       string    `gorm:"size:50" json:"item_status"` // e.g. Upcoming, Live, Ended
	ImagesJson       string    `gorm:"type:jsonb" json:"images_json"` // High-res image gallery URLs
	RawDetailsJson   string    `gorm:"type:jsonb" json:"raw_details_json"`
	FetchedAt        time.Time `json:"fetched_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (CopartLotCache) TableName() string {
	return "copart_lot_caches"
}

// CopartBid tracks every single bid executed by our system users through the master account
type CopartBid struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	User           *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	LotNumber      string    `gorm:"size:32;index;not null" json:"lot_number"`
	VIN            string    `gorm:"size:17" json:"vin"`
	VehicleTitle   string    `gorm:"size:255" json:"vehicle_title"`
	BidType        string    `gorm:"size:20;default:'pre_bid'" json:"bid_type"` // pre_bid, live_bid, buy_it_now
	MaxBidAmount   float64   `json:"max_bid_amount"`
	PlacedAmount   float64   `json:"placed_amount"`
	FinalSalePrice float64   `json:"final_sale_price"`
	Status         string    `gorm:"size:50;default:'pending'" json:"status"` // pending, accepted, outbid, winning, won, lost, rejected
	CopartRef      string    `gorm:"size:100" json:"copart_ref"`              // Internal Copart bid reference code
	IPAddress      string    `gorm:"size:50" json:"ip_address"`
	UserNotes      string    `gorm:"type:text" json:"user_notes"`
	AdminNotes     string    `gorm:"type:text" json:"admin_notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (CopartBid) TableName() string {
	return "copart_bids"
}

