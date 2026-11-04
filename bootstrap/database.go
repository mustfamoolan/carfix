package bootstrap

import (
	"fmt"
	"log"
	"time"

	"carfix/app/models"
	"carfix/config"
	"carfix/database/seeders"
	"github.com/fatih/color"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitializeDatabase() *gorm.DB {
	var err error
	var dialector gorm.Dialector

	dbConfig := config.Global.Database

	switch dbConfig.Driver {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Database)
		dialector = mysql.Open(dsn)
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
			dbConfig.Host, dbConfig.Username, dbConfig.Password, dbConfig.Database, dbConfig.Port)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(dbConfig.Database + ".db")
	default:
		log.Fatalf("%s Unsupported database driver: %s", color.RedString("FATAL:"), dbConfig.Driver)
	}

	for i := 0; i < 15; i++ {
		DB, err = gorm.Open(dialector, &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("%s Failed to connect to database: %v", color.RedString("FATAL:"), err)
	}

	// Ensure core tables and relations are migrated
	if err := DB.AutoMigrate(
		&models.User{},
		&models.UserQuota{},
		&models.SearchLog{},
		&models.ReportCache{},
		&models.CopartLotCache{},
		&models.CopartBid{},
	); err != nil {
		log.Printf("%s AutoMigrate warning: %v\n", color.YellowString("⚠️"), err)
	}

	// Ensure seeders run (safe idempotent admin seeder)
	if err := seeders.Run(DB); err != nil {
		log.Printf("%s Seeder warning: %v\n", color.YellowString("⚠️"), err)
	}

	fmt.Printf("%s Database connected, migrated and seeded successfully (%s)\n", color.CyanString("⚙️"), dbConfig.Driver)
	return DB
}
