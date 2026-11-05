package controllers

import (
	"strconv"
	"strings"
	"time"

	"carfix/app/models"
	"carfix/app/services"
	"carfix/bootstrap"
	"github.com/gofiber/fiber/v2"
)

type CopartController struct {
	copartService *services.CopartService
}

func NewCopartController() *CopartController {
	return &CopartController{
		copartService: services.NewCopartService(),
	}
}

// GetLot retrieves Copart lot information, high-res photos, and auction state
func (ctrl *CopartController) GetLot(c *fiber.Ctx) error {
	lotOrVin := strings.TrimSpace(c.Params("lot"))
	if lotOrVin == "" {
		return c.Status(400).JSON(fiber.Map{"message": "يرجى تحديد رقم اللوت أو رقم الشاصي"})
	}

	lotData, err := ctrl.copartService.GetLotDetails(lotOrVin)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"lot":    lotData,
	})
}

// PlacePreBid allows administrators or approved users to place a tracked pre-bid
func (ctrl *CopartController) PlacePreBid(c *fiber.Ctx) error {
	var body struct {
		LotNumber    string  `json:"lot_number"`
		VIN          string  `json:"vin"`
		VehicleTitle string  `json:"vehicle_title"`
		MaxBidAmount float64 `json:"max_bid_amount"`
		Notes        string  `json:"notes"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "بيانات غير صالحة"})
	}

	body.LotNumber = strings.TrimSpace(body.LotNumber)
	if body.LotNumber == "" || body.MaxBidAmount <= 0 {
		return c.Status(400).JSON(fiber.Map{"message": "رقم اللوت ومبلغ المزايدة مطلوبان"})
	}

	userID := c.Locals("user_id").(uint)

	bid := models.CopartBid{
		UserID:         userID,
		LotNumber:      body.LotNumber,
		VIN:            strings.ToUpper(strings.TrimSpace(body.VIN)),
		VehicleTitle:   body.VehicleTitle,
		BidType:        "pre_bid",
		MaxBidAmount:   body.MaxBidAmount,
		PlacedAmount:   body.MaxBidAmount,
		FinalSalePrice: 0,
		Status:         "accepted", // Recorded in our internal broker ledger
		CopartRef:      "CPT-" + strconv.FormatInt(time.Now().Unix(), 10),
		IPAddress:      c.IP(),
		UserNotes:      body.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := bootstrap.DB.Create(&bid).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل تسجيل المزايدة في النظام"})
	}

	return c.Status(201).JSON(fiber.Map{
		"status":  "success",
		"message": "تم تسجيل المزايدة بنجاح تحت الحساب الموحد",
		"bid":     bid,
	})
}

// ListBids returns all internal bids for administrative tracking and auditing
func (ctrl *CopartController) ListBids(c *fiber.Ctx) error {
	lotNumber := strings.TrimSpace(c.Query("lot_number"))
	userID := strings.TrimSpace(c.Query("user_id"))

	query := bootstrap.DB.Model(&models.CopartBid{}).Preload("User")
	if lotNumber != "" {
		query = query.Where("lot_number = ?", lotNumber)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	var bids []models.CopartBid
	if err := query.Order("id DESC").Find(&bids).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل جلب سجل المزايدات"})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"bids":   bids,
		"total":  len(bids),
	})
}

