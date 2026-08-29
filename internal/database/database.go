package database

import (
	"log"
	"os"

	"gowatch/internal/config"
	"gowatch/internal/database/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	var err error

	// DB yolunu environment'tan oku (Docker volume desteği için)
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "gowatch.db"
	}

	gormConfig := &gorm.Config{}
	if config.App.GinMode == "release" {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	DB, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrations
	err = DB.AutoMigrate(
		&models.User{},
		&models.Monitor{},
		&models.Heartbeat{},
		&models.Notification{},
		&models.MaintenanceWindow{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database connected and migrated successfully")

	// Admin kullanıcısını oluştur (yoksa)
	adminID := seedAdmin()

	// Mevcut sahipsiz kayıtları (user_id=0) admin'e ata
	DB.Model(&models.Monitor{}).Where("user_id = ?", 0).Update("user_id", adminID)
	DB.Model(&models.Notification{}).Where("user_id = ?", 0).Update("user_id", adminID)
}

// seedAdmin admin kullanıcısını oluşturur, user ID'sini döndürür
func seedAdmin() uint {
	var admin models.User
	result := DB.Where("username = ?", config.App.AdminUsername).First(&admin)

	if result.Error == nil {
		// Admin zaten var
		return admin.ID
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(config.App.AdminPassword), bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}

	admin = models.User{
		Username: config.App.AdminUsername,
		Password: string(hashedPassword),
	}

	if err := DB.Create(&admin).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("Admin user created: %s", config.App.AdminUsername)
	return admin.ID
}
