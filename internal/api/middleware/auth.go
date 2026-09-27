package middleware

import (
	"crypto/sha256"
	"net/http"

	"gowatch/internal/database"
	"gowatch/internal/database/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const SessionName = "gowatch-session"

var Store *sessions.CookieStore

func InitSession(secret string, secure bool) {
	// Tek bir secret'tan imzalama (HMAC) ve şifreleme (AES-256) anahtarları türetilir;
	// böylece çerez içeriği hem değiştirilemez hem de okunamaz.
	hashKey := sha256.Sum256([]byte("gowatch-cookie-hash:" + secret))
	blockKey := sha256.Sum256([]byte("gowatch-cookie-block:" + secret))

	Store = sessions.NewCookieStore(hashKey[:], blockKey[:])
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// SaveUserSession kullanıcı için oturum çerezini yazar
func SaveUserSession(c *gin.Context, user *models.User) error {
	session, _ := Store.Get(c.Request, SessionName)
	session.Values["user_id"] = user.ID
	session.Values["session_version"] = user.SessionVersion
	return session.Save(c.Request, c.Writer)
}

// CurrentUser oturumdaki kullanıcıyı veritabanından doğrular.
// Kullanıcı silinmişse veya şifre değiştiği için oturum sürümü eskimişse nil döner.
func CurrentUser(r *http.Request) *models.User {
	session, err := Store.Get(r, SessionName)
	if err != nil {
		return nil
	}
	uid, ok := session.Values["user_id"].(uint)
	if !ok {
		return nil
	}
	version, _ := session.Values["session_version"].(int)

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		return nil
	}
	if user.SessionVersion != version {
		return nil
	}
	return &user
}

// AuthRequired oturum kontrolü middleware
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c.Request)
		if user == nil {
			if c.Request.Header.Get("Accept") == "application/json" ||
				c.Request.Header.Get("Content-Type") == "application/json" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			} else {
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Next()
	}
}
