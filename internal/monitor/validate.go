package monitor

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"gowatch/internal/database/models"
)

// Kaynak tüketimini sınırlamak için değer aralıkları
const (
	MinInterval     = 30
	MaxInterval     = 86400
	MinTimeout      = 1
	MaxTimeout      = 60
	MaxRetries      = 10
	MaxRedirectsCap = 20
)

var (
	acceptedCodesRe = regexp.MustCompile(`^\d{3}(-\d{3})?(\s*,\s*\d{3}(-\d{3})?)*$`)
	hostRe          = regexp.MustCompile(`^[A-Za-z0-9.\-:\[\]]+$`)
	dnsTypes        = map[string]bool{"": true, "A": true, "AAAA": true, "CNAME": true, "MX": true, "NS": true, "TXT": true}
)

// Validate monitör alanlarını kontrol eder, hatalı alan varsa açıklayıcı bir hata döndürür
func Validate(m *models.Monitor) error {
	name := strings.TrimSpace(m.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.New("name must be 1-100 characters")
	}
	if utf8.RuneCountInString(m.Description) > 255 {
		return errors.New("description must be at most 255 characters")
	}

	switch m.Type {
	case models.TypeHTTP:
		u, err := url.Parse(m.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("url must be a valid http:// or https:// address")
		}
		if m.AcceptedCodes != "" && !acceptedCodesRe.MatchString(m.AcceptedCodes) {
			return errors.New(`accepted_codes must look like "200-299" or "200,201,204"`)
		}
		if m.MaxRedirects < 0 || m.MaxRedirects > MaxRedirectsCap {
			return fmt.Errorf("max_redirects must be 0-%d", MaxRedirectsCap)
		}
	case models.TypeTCP:
		if !validHost(m.URL) {
			return errors.New("host must be a hostname or IP address (without http://)")
		}
		if m.Port < 1 || m.Port > 65535 {
			return errors.New("port must be 1-65535")
		}
	case models.TypePing:
		if !validHost(m.URL) {
			return errors.New("host must be a hostname or IP address (without http://)")
		}
	case models.TypeDNS:
		if !validHost(m.URL) {
			return errors.New("domain must be a valid hostname")
		}
		if !dnsTypes[strings.ToUpper(m.DNSResolveType)] {
			return errors.New("dns_resolve_type must be one of A, AAAA, CNAME, MX, NS, TXT")
		}
	default:
		return errors.New("type must be one of http, tcp, ping, dns")
	}

	if m.Interval < MinInterval || m.Interval > MaxInterval {
		return fmt.Errorf("interval must be %d-%d seconds", MinInterval, MaxInterval)
	}
	if m.Timeout < MinTimeout || m.Timeout > MaxTimeout {
		return fmt.Errorf("timeout must be %d-%d seconds", MinTimeout, MaxTimeout)
	}
	if m.Retries < 0 || m.Retries > MaxRetries {
		return fmt.Errorf("retries must be 0-%d", MaxRetries)
	}
	return nil
}

func validHost(h string) bool {
	return h != "" && len(h) <= 253 && hostRe.MatchString(h)
}
