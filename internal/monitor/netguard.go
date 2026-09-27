package monitor

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
	"time"

	"gowatch/internal/config"
)

// ErrBlockedTarget iç ağ adreslerine yapılan kontrol reddedildiğinde döner
var ErrBlockedTarget = errors.New("target resolves to a private/internal address (set ALLOW_PRIVATE_TARGETS=true to allow)")

// Loopback, private, link-local vb. dışında kalan ama yine de dahili sayılan aralıklar
var extraBlockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),     // "bu ağ"
	netip.MustParsePrefix("100.64.0.0/10"), // CGNAT / paylaşımlı adres alanı
	netip.MustParsePrefix("192.0.0.0/24"),  // IETF protokol atamaları
	netip.MustParsePrefix("198.18.0.0/15"), // benchmark ağları
	netip.MustParsePrefix("64:ff9b:1::/48"),
}

// isBlockedIP adresin sunucunun iç ağına veya kendisine ait olup olmadığını döndürür
func isBlockedIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	for _, p := range extraBlockedPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func allowPrivateTargets() bool {
	return config.App != nil && config.App.AllowPrivateTargets
}

// newDialer kontrol bağlantıları için bir net.Dialer döndürür.
// Adres DNS çözümlemesinden SONRA, bağlantı kurulmadan hemen önce kontrol edildiği için
// DNS rebinding ve yönlendirme (redirect) ile iç ağa erişim de engellenir.
func newDialer(timeout time.Duration) *net.Dialer {
	d := &net.Dialer{Timeout: timeout}
	if allowPrivateTargets() {
		return d
	}
	d.Control = func(network, address string, _ syscall.RawConn) error {
		ap, err := netip.ParseAddrPort(address)
		if err != nil {
			return fmt.Errorf("invalid address %q: %w", address, err)
		}
		if isBlockedIP(ap.Addr()) {
			return ErrBlockedTarget
		}
		return nil
	}
	return d
}
