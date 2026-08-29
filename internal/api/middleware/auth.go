package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

var Store *sessions.CookieStore

func InitSession(secret string) {
	Store = sessions.NewCookieStore([]byte(secret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// AuthRequired oturum kontrolü middleware
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := Store.Get(c.Request, "gowatch-session")
		if err != nil || session.Values["user_id"] == nil {
			if c.Request.Header.Get("Accept") == "application/json" ||
				c.Request.Header.Get("Content-Type") == "application/json" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			} else {
				c.Redirect(http.StatusFound, "/login")
			}
			c.Abort()
			return
		}
		c.Set("user_id", session.Values["user_id"])
		c.Set("username", session.Values["username"])
		c.Next()
	}
}
