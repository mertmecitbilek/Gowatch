package monitor

import (
	"errors"
	"net"
	"net/netip"
	"testing"

	"gowatch/internal/config"
	"gowatch/internal/database/models"
)

func validHTTP() models.Monitor {
	return models.Monitor{
		Name: "Site", Type: models.TypeHTTP, URL: "https://example.com",
		Interval: 60, Timeout: 30, Retries: 1, MaxRedirects: 10, AcceptedCodes: "200-299",
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		modify func(m *models.Monitor)
		ok     bool
	}{
		{"valid http", func(m *models.Monitor) {}, true},
		{"empty name", func(m *models.Monitor) { m.Name = "  " }, false},
		{"unknown type", func(m *models.Monitor) { m.Type = "smtp" }, false},
		{"ftp scheme", func(m *models.Monitor) { m.URL = "ftp://example.com" }, false},
		{"file scheme", func(m *models.Monitor) { m.URL = "file:///etc/passwd" }, false},
		{"interval too small", func(m *models.Monitor) { m.Interval = 1 }, false},
		{"negative interval", func(m *models.Monitor) { m.Interval = -5 }, false},
		{"timeout too big", func(m *models.Monitor) { m.Timeout = 600 }, false},
		{"retries 0 allowed", func(m *models.Monitor) { m.Retries = 0 }, true},
		{"too many retries", func(m *models.Monitor) { m.Retries = 99 }, false},
		{"codes list", func(m *models.Monitor) { m.AcceptedCodes = "200-299, 404" }, true},
		{"bad codes", func(m *models.Monitor) { m.AcceptedCodes = "abc" }, false},
		{"tcp ok", func(m *models.Monitor) { m.Type = models.TypeTCP; m.URL = "db.example.com"; m.Port = 5432 }, true},
		{"tcp no port", func(m *models.Monitor) { m.Type = models.TypeTCP; m.URL = "db.example.com"; m.Port = 0 }, false},
		{"tcp url with scheme", func(m *models.Monitor) { m.Type = models.TypeTCP; m.URL = "http://x"; m.Port = 80 }, false},
		{"dns bad type", func(m *models.Monitor) { m.Type = models.TypeDNS; m.URL = "example.com"; m.DNSResolveType = "ANY" }, false},
		{"dns mx", func(m *models.Monitor) { m.Type = models.TypeDNS; m.URL = "example.com"; m.DNSResolveType = "MX" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validHTTP()
			tt.modify(&m)
			err := Validate(&m)
			if (err == nil) != tt.ok {
				t.Fatalf("Validate() error = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestIsAcceptedCode(t *testing.T) {
	cases := []struct {
		code  int
		codes string
		want  bool
	}{
		{200, "", true},
		{204, "200-299", true},
		{404, "200-299", false},
		{404, "200-299,404", true},
		{301, "200,301", true},
		{500, "200,301", false},
	}
	for _, c := range cases {
		if got := isAcceptedCode(c.code, c.codes); got != c.want {
			t.Errorf("isAcceptedCode(%d, %q) = %v, want %v", c.code, c.codes, got, c.want)
		}
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.10", "169.254.169.254",
		"0.0.0.0", "100.64.0.1", "::1", "fd00::1", "fe80::1", "::ffff:127.0.0.1"}
	allowed := []string{"1.1.1.1", "8.8.8.8", "140.82.121.4", "2606:4700:4700::1111"}

	for _, s := range blocked {
		if !isBlockedIP(netip.MustParseAddr(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
	for _, s := range allowed {
		if isBlockedIP(netip.MustParseAddr(s)) {
			t.Errorf("%s should be allowed", s)
		}
	}
}

func TestDialerBlocksPrivateTargets(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	config.App = &config.Config{AllowPrivateTargets: false}
	defer func() { config.App = nil }()

	res := CheckTCP(&models.Monitor{URL: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Timeout: 2})
	if res.Status != models.StatusDown {
		t.Fatalf("expected DOWN for loopback target, got %v (%s)", res.Status, res.Message)
	}

	_, err = newDialer(0).Dial("tcp", ln.Addr().String())
	if !errors.Is(err, ErrBlockedTarget) {
		t.Fatalf("expected ErrBlockedTarget, got %v", err)
	}

	// İzin verildiğinde aynı hedef UP olmalı
	config.App.AllowPrivateTargets = true
	res = CheckTCP(&models.Monitor{URL: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Timeout: 2})
	if res.Status != models.StatusUp {
		t.Fatalf("expected UP when private targets allowed, got %v (%s)", res.Status, res.Message)
	}
}

func TestPingRefusedMeansReachable(t *testing.T) {
	config.App = &config.Config{AllowPrivateTargets: true}
	defer func() { config.App = nil }()

	// 127.0.0.1'de 443/80 genelde kapalıdır: "connection refused" host'un ayakta olduğunu gösterir
	res := CheckPing(&models.Monitor{URL: "127.0.0.1", Timeout: 2})
	if res.Status != models.StatusUp {
		t.Fatalf("expected UP for reachable host, got %v (%s)", res.Status, res.Message)
	}
}
