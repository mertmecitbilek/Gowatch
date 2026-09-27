package middleware

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SameOrigin durum değiştiren isteklerde (POST/PUT/PATCH/DELETE) Origin başlığını kontrol eder.
// Başka bir siteden gönderilen istekler (CSRF) reddedilir.
func SameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			// Tarayıcı dışı istemciler (curl vb.) Origin göndermez
			c.Next()
			return
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host != c.Request.Host {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross-origin request rejected"})
			return
		}
		c.Next()
	}
}

// RateLimiter anahtar (ör. IP) başına sabit pencere içinde en fazla N denemeye izin verir
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]*rateEntry
	now     func() time.Time
}

type rateEntry struct {
	count int
	start time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]*rateEntry),
		now:     time.Now,
	}
}

// Blocked anahtarın limiti aşıp aşmadığını döndürür (sayaç artırılmaz)
func (rl *RateLimiter) Blocked(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.entries[key]
	if !ok || rl.now().Sub(e.start) > rl.window {
		return false
	}
	return e.count >= rl.limit
}

// Hit anahtar için bir deneme kaydeder
func (rl *RateLimiter) Hit(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()

	// Bellek şişmesin diye süresi dolmuş kayıtları temizle
	if len(rl.entries) > 10000 {
		for k, e := range rl.entries {
			if now.Sub(e.start) > rl.window {
				delete(rl.entries, k)
			}
		}
	}

	e, ok := rl.entries[key]
	if !ok || now.Sub(e.start) > rl.window {
		rl.entries[key] = &rateEntry{count: 1, start: now}
		return
	}
	e.count++
}

// Reset anahtarın sayacını sıfırlar (ör. başarılı girişten sonra)
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.entries, key)
}

// SecurityHeaders tarayıcı tarafı güvenlik başlıklarını ekler
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY") // clickjacking koruması
		h.Set("Referrer-Policy", "same-origin")
		c.Next()
	}
}
