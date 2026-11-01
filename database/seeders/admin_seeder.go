package seeders

import (
	"fmt"
	"time"

	"carfix/app/models"
	"github.com/fatih/color"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminSeeder struct{}

func (s *AdminSeeder) Run(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		fmt.Printf("  %s Admin user already exists, skipping.\n", color.YellowString("ℹ️"))
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	admin := models.User{
		FullName: "Mustafa Admin",
		Phone:    "07700000000",
		Username: "admin",
		Password: string(hashedPassword),
		Role:     "admin",
		IsActive: true,
	}

	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	quota := models.UserQuota{
		UserID:        admin.ID,
		TotalLimit:    100000,
		UsedTotal:     0,
		DailyLimit:    1000,
		UsedToday:     0,
		LastResetDate: time.Now(),
	}

	if err := db.Create(&quota).Error; err != nil {
		return fmt.Errorf("failed to create admin quota: %w", err)
	}

	fmt.Printf("  %s Admin user (admin) and quota created successfully.\n", color.GreenString("✓"))
	return nil
}

