package database

import (
	"crypto/rand"
	"encoding/base64"
	"log"

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

	// DB yolu config'den gelir (Docker volume desteği için DATABASE_PATH)
	dbPath := config.App.DatabasePath

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
		// Admin zaten var; hâlâ varsayılan şifreyi kullanıyorsa uyar
		if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(config.DefaultAdminPassword)) == nil {
			log.Printf("⚠️  SECURITY WARNING: user %q still uses the default password %q. Change it from the Settings page!",
				admin.Username, config.DefaultAdminPassword)
		}
		return admin.ID
	}

	// Şifre verilmemişse veya bilinen varsayılan şifreyse rastgele bir şifre üret
	password := config.App.AdminPassword
	generated := false
	if password == "" || password == config.DefaultAdminPassword {
		password = randomPassword()
		generated = true
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
	if generated {
		// Şifre yalnızca bu ilk oluşturmada bir kez gösterilir
		log.Printf("🔑 Generated admin password (shown only once, change it after login): %s", password)
	}
	return admin.ID
}

func randomPassword() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("Failed to generate admin password: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
