package models

import (
	"time"

	"gorm.io/gorm"
)

// MonitorType monitör türü
type MonitorType string

const (
	TypeHTTP MonitorType = "http"
	TypeTCP  MonitorType = "tcp"
	TypePing MonitorType = "ping"
	TypeDNS  MonitorType = "dns"
)

// MonitorStatus monitör durumu
type MonitorStatus int

const (
	StatusDown    MonitorStatus = 0
	StatusUp      MonitorStatus = 1
	StatusPending MonitorStatus = 2
)

// User sistem kullanıcısı
type User struct {
	gorm.Model
	Username string `json:"username" gorm:"uniqueIndex;not null"`
	Password string `json:"-" gorm:"not null"` // bcrypt hash
	// SessionVersion şifre değiştiğinde artırılır; eski oturum çerezleri geçersiz olur
	SessionVersion int `json:"-" gorm:"not null;default:0"`
}

// Monitor izlenen servis
type Monitor struct {
	gorm.Model
	UserID           uint          `json:"user_id" gorm:"index;not null;default:1"`
	Name             string        `json:"name" gorm:"not null"`
	Type             MonitorType   `json:"type" gorm:"not null"`
	URL              string        `json:"url"`
	Port             int           `json:"port"`
	Interval         int           `json:"interval" gorm:"default:60"`
	Timeout          int           `json:"timeout" gorm:"default:30"`
	Retries          int           `json:"retries" gorm:"default:1"`
	Active           bool          `json:"active" gorm:"default:true"`
	Status           MonitorStatus `json:"status" gorm:"default:2"`
	UptimePercent    float64       `json:"uptime_percent"`
	AvgLatency       int64         `json:"avg_latency"`
	LastCheckedAt    *time.Time    `json:"last_checked_at"`
	NotificationIDs  string        `json:"notification_ids"`
	MaxRedirects     int           `json:"max_redirects" gorm:"default:10"`
	AcceptedCodes    string        `json:"accepted_codes" gorm:"default:\"200-299\""`
	Description      string        `json:"description"`
	DNSResolveType   string        `json:"dns_resolve_type"`
	DNSResolveServer string        `json:"dns_resolve_server"`
	CronJobID        int           `json:"-" gorm:"-"`
}

// Heartbeat tek bir kontrol sonucu
type Heartbeat struct {
	ID        uint          `json:"id" gorm:"primarykey"`
	MonitorID uint          `json:"monitor_id" gorm:"index;not null"`
	Status    MonitorStatus `json:"status"`
	Latency   int64         `json:"latency"`
	Msg       string        `json:"msg"`
	Time      time.Time     `json:"time" gorm:"index"`
}

// Notification bildirim kanalı
type Notification struct {
	gorm.Model
	UserID uint   `json:"user_id" gorm:"index;not null;default:1"`
	Name   string `json:"name" gorm:"not null"`
	Type   string `json:"type" gorm:"not null"`
	Config string `json:"config" gorm:"type:text"`
	Active bool   `json:"active" gorm:"default:true"`
}

// MaintenanceWindow bakım penceresi
type MaintenanceWindow struct {
	gorm.Model
	UserID    uint      `json:"user_id" gorm:"index;not null;default:1"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Active    bool      `json:"active" gorm:"default:true"`
}
