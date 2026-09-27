package monitor

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gowatch/internal/database/models"
)

// CheckResult tek kontrol sonucu
type CheckResult struct {
	Status  models.MonitorStatus
	Latency int64
	Message string
}

// CheckHTTP HTTP/HTTPS kontrolü
func CheckHTTP(m *models.Monitor) CheckResult {
	timeout := time.Duration(m.Timeout) * time.Second
	client := &http.Client{
		Timeout: timeout,
		// Her bağlantı (yönlendirmeler dahil) iç ağ korumalı dialer'dan geçer
		Transport: &http.Transport{
			DialContext:         newDialer(timeout).DialContext,
			TLSHandshakeTimeout: timeout,
			DisableKeepAlives:   true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if m.MaxRedirects > 0 && len(via) >= m.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Get(m.URL)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{
			Status:  models.StatusDown,
			Latency: latency,
			Message: fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	// Kabul edilen status kodlarını kontrol et (varsayılan: 200-299)
	if isAcceptedCode(resp.StatusCode, m.AcceptedCodes) {
		return CheckResult{
			Status:  models.StatusUp,
			Latency: latency,
			Message: resp.Status,
		}
	}

	return CheckResult{
		Status:  models.StatusDown,
		Latency: latency,
		Message: fmt.Sprintf("Unexpected status: %s", resp.Status),
	}
}

// CheckTCP TCP port kontrolü
func CheckTCP(m *models.Monitor) CheckResult {
	host := m.URL
	if m.Port > 0 {
		host = net.JoinHostPort(m.URL, strconv.Itoa(m.Port))
	}

	start := time.Now()
	conn, err := newDialer(time.Duration(m.Timeout)*time.Second).Dial("tcp", host)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{
			Status:  models.StatusDown,
			Latency: latency,
			Message: fmt.Sprintf("TCP connection failed: %v", err),
		}
	}
	defer conn.Close()

	return CheckResult{
		Status:  models.StatusUp,
		Latency: latency,
		Message: fmt.Sprintf("TCP connection established to %s", host),
	}
}

// CheckPing host erişilebilirlik kontrolü.
// ICMP root yetkisi gerektirdiği için yaygın portlara (443, 80) TCP bağlantısı denenir.
// Bağlantının kurulması veya host'un "connection refused" ile cevap vermesi host'un
// ayakta olduğunu gösterir; zaman aşımı veya ağ hatası ise DOWN kabul edilir.
func CheckPing(m *models.Monitor) CheckResult {
	ports := []string{"443", "80"}
	perPort := time.Duration(m.Timeout) * time.Second / time.Duration(len(ports))
	dialer := newDialer(perPort)

	var lastErr error
	for _, port := range ports {
		start := time.Now()
		conn, err := dialer.Dial("tcp", net.JoinHostPort(m.URL, port))
		latency := time.Since(start).Milliseconds()

		if err == nil {
			conn.Close()
			return CheckResult{
				Status:  models.StatusUp,
				Latency: latency,
				Message: fmt.Sprintf("Host reachable: %s (tcp/%s open)", m.URL, port),
			}
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			return CheckResult{
				Status:  models.StatusUp,
				Latency: latency,
				Message: fmt.Sprintf("Host reachable: %s (tcp/%s refused)", m.URL, port),
			}
		}
		if errors.Is(err, ErrBlockedTarget) {
			return CheckResult{Status: models.StatusDown, Latency: latency, Message: err.Error()}
		}
		lastErr = err
	}

	return CheckResult{
		Status:  models.StatusDown,
		Latency: int64(m.Timeout) * 1000,
		Message: fmt.Sprintf("Host unreachable: %v", lastErr),
	}
}

// CheckDNS DNS çözümleme kontrolü
func CheckDNS(m *models.Monitor) CheckResult {
	start := time.Now()

	resolveType := m.DNSResolveType
	if resolveType == "" {
		resolveType = "A"
	}

	var err error
	var result string

	switch strings.ToUpper(resolveType) {
	case "A", "AAAA":
		addrs, e := net.LookupHost(m.URL)
		err = e
		if e == nil {
			result = strings.Join(addrs, ", ")
		}
	case "CNAME":
		cname, e := net.LookupCNAME(m.URL)
		err = e
		result = cname
	case "MX":
		mxs, e := net.LookupMX(m.URL)
		err = e
		if e == nil {
			var parts []string
			for _, mx := range mxs {
				parts = append(parts, fmt.Sprintf("%s (pref %d)", mx.Host, mx.Pref))
			}
			result = strings.Join(parts, ", ")
		}
	case "NS":
		nss, e := net.LookupNS(m.URL)
		err = e
		if e == nil {
			var parts []string
			for _, ns := range nss {
				parts = append(parts, ns.Host)
			}
			result = strings.Join(parts, ", ")
		}
	case "TXT":
		txts, e := net.LookupTXT(m.URL)
		err = e
		result = strings.Join(txts, "; ")
	default:
		addrs, e := net.LookupHost(m.URL)
		err = e
		if e == nil {
			result = strings.Join(addrs, ", ")
		}
	}

	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{
			Status:  models.StatusDown,
			Latency: latency,
			Message: fmt.Sprintf("DNS lookup failed: %v", err),
		}
	}

	return CheckResult{
		Status:  models.StatusUp,
		Latency: latency,
		Message: fmt.Sprintf("DNS resolved (%s): %s", resolveType, result),
	}
}

// isAcceptedCode verilen status kodunun kabul edilen aralıkta olup olmadığını kontrol eder
// Format: "200-299" veya "200,201,204" veya "200-299,404"
func isAcceptedCode(code int, acceptedCodes string) bool {
	if acceptedCodes == "" {
		acceptedCodes = "200-299"
	}

	parts := strings.Split(acceptedCodes, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			var min, max int
			fmt.Sscanf(part, "%d-%d", &min, &max)
			if code >= min && code <= max {
				return true
			}
		} else {
			var expected int
			fmt.Sscanf(part, "%d", &expected)
			if code == expected {
				return true
			}
		}
	}
	return false
}
