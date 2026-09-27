package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	rl := NewRateLimiter(3, time.Minute)
	rl.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if rl.Blocked("1.2.3.4") {
			t.Fatalf("blocked too early at attempt %d", i)
		}
		rl.Hit("1.2.3.4")
	}
	if !rl.Blocked("1.2.3.4") {
		t.Fatal("expected to be blocked after 3 attempts")
	}
	if rl.Blocked("5.6.7.8") {
		t.Fatal("other keys must not be affected")
	}

	now = now.Add(61 * time.Second)
	if rl.Blocked("1.2.3.4") {
		t.Fatal("expected block to expire after window")
	}

	rl.Hit("9.9.9.9")
	rl.Reset("9.9.9.9")
	if _, ok := rl.entries["9.9.9.9"]; ok {
		t.Fatal("Reset should remove the entry")
	}
}

func TestSameOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SameOrigin())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		method, origin string
		want           int
	}{
		{"POST", "", http.StatusOK},                            // curl vb.
		{"POST", "http://gowatch.local", http.StatusOK},        // aynı site
		{"POST", "https://evil.example", http.StatusForbidden}, // CSRF
		{"POST", "null", http.StatusForbidden},
		{"GET", "https://evil.example", http.StatusOK}, // okuma istekleri etkilenmez
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, "http://gowatch.local/x", nil)
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("%s with Origin %q: got %d, want %d", c.method, c.origin, w.Code, c.want)
		}
	}
}
