package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	GinMode             string
	SessionSecret       string
	AdminUsername       string
	AdminPassword       string
	DatabasePath        string
	CookieSecure        bool
	TrustedProxies      []string
	AllowPrivateTargets bool
	MaxMonitorsPerUser  int
}

var App *Config

// Repoda/örnek dosyalarda yer alan, herkesin bildiği değerler.
// Bu değerlerle imzalanan oturum çerezleri taklit edilebileceği için kabul edilmez.
var knownDefaultSecrets = map[string]bool{
	"gowatch-secret-key-change-me":          true,
	"change-this-to-a-strong-random-secret": true,
	"your-super-secret-key-change-this":     true,
}

// DefaultAdminPassword ilk sürümlerde kullanılan varsayılan admin şifresi
const DefaultAdminPassword = "admin123"

func Load() {
	// .env dosyasını yükle (yoksa sistem ortam değişkenlerini kullan)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	App = &Config{
		Port:                getEnv("PORT", "8080"),
		GinMode:             getEnv("GIN_MODE", "debug"),
		AdminUsername:       getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:       getEnv("ADMIN_PASSWORD", ""),
		DatabasePath:        getEnv("DATABASE_PATH", "gowatch.db"),
		CookieSecure:        getEnvBool("COOKIE_SECURE", false),
		TrustedProxies:      splitList(getEnv("TRUSTED_PROXIES", "")),
		AllowPrivateTargets: getEnvBool("ALLOW_PRIVATE_TARGETS", false),
		MaxMonitorsPerUser:  getEnvInt("MAX_MONITORS_PER_USER", 100),
	}
	App.SessionSecret = resolveSessionSecret(getEnv("SESSION_SECRET", ""), filepath.Dir(App.DatabasePath))
}

// resolveSessionSecret geçerli bir oturum anahtarı döndürür.
// SESSION_SECRET boşsa veya bilinen varsayılan değerlerden biriyse, veri dizinindeki
// session.key dosyası kullanılır; dosya yoksa rastgele bir anahtar üretilip kaydedilir.
func resolveSessionSecret(envSecret, dataDir string) string {
	if envSecret != "" && !knownDefaultSecrets[envSecret] {
		if len(envSecret) < 32 {
			log.Println("⚠️  SESSION_SECRET is shorter than 32 characters, consider using a longer random value")
		}
		return envSecret
	}
	if knownDefaultSecrets[envSecret] {
		log.Println("⚠️  SESSION_SECRET is set to a publicly known default value and will be ignored")
	}

	keyPath := filepath.Join(dataDir, "session.key")
	if data, err := os.ReadFile(keyPath); err == nil {
		if key := strings.TrimSpace(string(data)); len(key) >= 32 {
			return key
		}
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("Failed to generate session secret: %v", err)
	}
	key := hex.EncodeToString(buf)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte(key+"\n"), 0o600); err != nil {
		log.Fatalf("Failed to save session secret to %s: %v", keyPath, err)
	}
	log.Printf("Generated a new random session secret: %s", keyPath)
	return key
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v, err := strconv.ParseBool(getEnv(key, "")); err == nil {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v, err := strconv.Atoi(getEnv(key, "")); err == nil {
		return v
	}
	return defaultValue
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
