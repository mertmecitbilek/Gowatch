package monitor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gowatch/internal/database"
	"gowatch/internal/database/models"
	"gowatch/internal/notification"
	ws "gowatch/internal/websocket"

	"github.com/robfig/cron/v3"
)

// Manager tüm monitörlerin zamanlayıcısını yönetir
type Manager struct {
	cron *cron.Cron
	jobs map[uint]cron.EntryID
	mu   sync.Mutex
}

var GlobalManager *Manager

func NewManager() *Manager {
	return &Manager{
		cron: cron.New(cron.WithSeconds()),
		jobs: make(map[uint]cron.EntryID),
	}
}

// Start cron scheduler'ı başlat ve mevcut aktif monitörleri yükle
func (m *Manager) Start() {
	m.cron.Start()
	log.Println("Monitor scheduler started")

	// Eski heartbeat kayıtlarını her gün temizle (veritabanı sınırsız büyümesin)
	go cleanupHeartbeats()
	m.cron.AddFunc("@every 24h", cleanupHeartbeats)

	// Veritabanındaki aktif monitörleri yükle
	var monitors []models.Monitor
	database.DB.Where("active = ?", true).Find(&monitors)
	for _, mon := range monitors {
		if err := m.AddMonitor(&mon); err != nil {
			log.Printf("Failed to schedule monitor %s: %v", mon.Name, err)
		}
	}
	log.Printf("Loaded %d monitors", len(monitors))
}

// Stop scheduler'ı durdur
func (m *Manager) Stop() {
	m.cron.Stop()
}

// AddMonitor yeni monitör ekle
func (m *Manager) AddMonitor(mon *models.Monitor) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Eğer zaten varsa kaldır
	if entryID, exists := m.jobs[mon.ID]; exists {
		m.cron.Remove(entryID)
	}

	// İlk kontrolü hemen yap
	go m.runCheck(mon.ID)

	// Cron ifadesi: her N saniyede bir
	cronExpr := fmt.Sprintf("@every %ds", mon.Interval)
	entryID, err := m.cron.AddFunc(cronExpr, func() {
		m.runCheck(mon.ID)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	m.jobs[mon.ID] = entryID
	log.Printf("Monitor scheduled: %s (every %ds)", mon.Name, mon.Interval)
	return nil
}

// RemoveMonitor monitörü scheduler'dan kaldır
func (m *Manager) RemoveMonitor(monitorID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entryID, exists := m.jobs[monitorID]; exists {
		m.cron.Remove(entryID)
		delete(m.jobs, monitorID)
		log.Printf("Monitor removed from scheduler: %d", monitorID)
	}
}

// runCheck tek bir kontrolü gerçekleştirir
func (m *Manager) runCheck(monitorID uint) {
	var mon models.Monitor
	if err := database.DB.First(&mon, monitorID).Error; err != nil {
		log.Printf("Monitor %d not found: %v", monitorID, err)
		return
	}

	if !mon.Active {
		return
	}

	var result CheckResult

	// Retry mekanizması
	for i := 0; i <= mon.Retries; i++ {
		switch mon.Type {
		case models.TypeHTTP:
			result = CheckHTTP(&mon)
		case models.TypeTCP:
			result = CheckTCP(&mon)
		case models.TypePing:
			result = CheckPing(&mon)
		case models.TypeDNS:
			result = CheckDNS(&mon)
		default:
			result = CheckHTTP(&mon)
		}

		if result.Status == models.StatusUp {
			break
		}
		if i < mon.Retries {
			time.Sleep(3 * time.Second)
		}
	}

	now := time.Now()

	// Heartbeat kaydet
	heartbeat := models.Heartbeat{
		MonitorID: mon.ID,
		Status:    result.Status,
		Latency:   result.Latency,
		Msg:       result.Message,
		Time:      now,
	}
	database.DB.Create(&heartbeat)

	// Durum değişikliği kontrolü
	previousStatus := mon.Status
	statusChanged := previousStatus != result.Status

	// Monitor güncelle (istatistikler bir kez hesaplanır)
	mon.Status = result.Status
	mon.LastCheckedAt = &now
	mon.AvgLatency = calculateAvgLatency(mon.ID)
	mon.UptimePercent = calculateUptimePercent(mon.ID)
	database.DB.Model(&mon).Updates(map[string]interface{}{
		"status":          mon.Status,
		"avg_latency":     mon.AvgLatency,
		"uptime_percent":  mon.UptimePercent,
		"last_checked_at": now,
	})

	// Durum değişikliğinde bildirim gönder
	if statusChanged && previousStatus != models.StatusPending {
		go notification.SendStatusChange(&mon, heartbeat, previousStatus)
	}

	// WebSocket ile yalnızca monitörün sahibine bildir
	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(mon.UserID, "heartbeat", map[string]interface{}{
			"monitor_id":     mon.ID,
			"monitor_name":   mon.Name,
			"status":         result.Status,
			"latency":        result.Latency,
			"msg":            result.Message,
			"time":           now,
			"uptime_percent": mon.UptimePercent,
			"avg_latency":    mon.AvgLatency,
			"status_changed": statusChanged,
		})
	}
}

// calculateAvgLatency son 24 saatin ortalama gecikmesi
func calculateAvgLatency(monitorID uint) int64 {
	var avg float64
	since := time.Now().Add(-24 * time.Hour)
	database.DB.Model(&models.Heartbeat{}).
		Where("monitor_id = ? AND time > ? AND status = ?", monitorID, since, models.StatusUp).
		Select("AVG(latency)").
		Scan(&avg)
	return int64(avg)
}

// calculateUptimePercent son 30 günün uptime yüzdesi
func calculateUptimePercent(monitorID uint) float64 {
	var total, up int64
	since := time.Now().Add(-30 * 24 * time.Hour)

	database.DB.Model(&models.Heartbeat{}).
		Where("monitor_id = ? AND time > ?", monitorID, since).
		Count(&total)

	if total == 0 {
		return 100.0
	}

	database.DB.Model(&models.Heartbeat{}).
		Where("monitor_id = ? AND time > ? AND status = ?", monitorID, since, models.StatusUp).
		Count(&up)

	return float64(up) / float64(total) * 100
}

// HeartbeatRetention bu süreden eski kontrol kayıtları silinir (uptime en fazla 30 gün üzerinden hesaplanır)
const HeartbeatRetention = 31 * 24 * time.Hour

func cleanupHeartbeats() {
	res := database.DB.Where("time < ?", time.Now().Add(-HeartbeatRetention)).Delete(&models.Heartbeat{})
	if res.Error != nil {
		log.Printf("Heartbeat cleanup failed: %v", res.Error)
	} else if res.RowsAffected > 0 {
		log.Printf("Heartbeat cleanup: %d old records deleted", res.RowsAffected)
	}
}
