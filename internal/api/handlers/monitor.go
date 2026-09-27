package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gowatch/internal/config"
	"gowatch/internal/database"
	"gowatch/internal/database/models"
	"gowatch/internal/monitor"

	"github.com/gin-gonic/gin"
)

// currentUserID session'dan user ID'sini alır
func currentUserID(c *gin.Context) uint {
	uid, _ := c.Get("user_id")
	if id, ok := uid.(uint); ok {
		return id
	}
	return 0
}

// userInitial kullanıcı adının büyük baş harfini döndürür
func userInitial(c *gin.Context) string {
	username := c.GetString("username")
	if username == "" {
		return "?"
	}
	return strings.ToUpper(string([]rune(username)[0]))
}

// normalizeNotificationIDs JSON dizisindeki bildirim kanallarının kullanıcıya ait olduğunu doğrular
func normalizeNotificationIDs(uid uint, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return "", nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return "", errors.New("notification_ids must be a JSON array of numbers")
	}
	if len(ids) == 0 {
		return "", nil
	}

	var count int64
	database.DB.Model(&models.Notification{}).Where("id IN ? AND user_id = ?", ids, uid).Count(&count)
	unique := map[uint]bool{}
	for _, id := range ids {
		unique[id] = true
	}
	if int(count) != len(unique) {
		return "", errors.New("one or more notification channels were not found")
	}
	out, _ := json.Marshal(ids)
	return string(out), nil
}

// GetDashboard dashboard sayfasını göster
func GetDashboard(c *gin.Context) {
	uid := currentUserID(c)
	var monitors []models.Monitor
	database.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&monitors)

	var totalUp, totalDown, totalPending int
	for _, m := range monitors {
		switch m.Status {
		case models.StatusUp:
			totalUp++
		case models.StatusDown:
			totalDown++
		case models.StatusPending:
			totalPending++
		}
	}

	var notifs []models.Notification
	database.DB.Where("user_id = ?", uid).Find(&notifs)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":         "Dashboard — GoWatch",
		"monitors":      monitors,
		"total_up":      totalUp,
		"total_down":    totalDown,
		"total_pending": totalPending,
		"total":         len(monitors),
		"username":      c.GetString("username"),
		"userInitial":   userInitial(c),
		"notifications": notifs,
	})
}

// GetMonitorDetail monitör detay sayfası
func GetMonitorDetail(c *gin.Context) {
	uid := currentUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var heartbeats []models.Heartbeat
	since := time.Now().Add(-48 * time.Hour)
	database.DB.Where("monitor_id = ? AND time > ?", id, since).
		Order("time DESC").
		Limit(500).
		Find(&heartbeats)

	// Durum çubuğu için son 90 kontrol, eskiden yeniye sıralı
	n := len(heartbeats)
	if n > 90 {
		n = 90
	}
	recent := make([]models.Heartbeat, n)
	for i := 0; i < n; i++ {
		recent[i] = heartbeats[n-1-i]
	}

	c.HTML(http.StatusOK, "monitor_detail.html", gin.H{
		"title":       mon.Name + " — GoWatch",
		"monitor":     mon,
		"heartbeats":  heartbeats,
		"recent":      recent,
		"username":    c.GetString("username"),
		"userInitial": userInitial(c),
	})
}

// --- API Endpoints ---

// monitorRequest oluşturma/güncelleme isteğinde kabul edilen alanlar.
// Pointer alanlar güncellemede "gönderilmedi" ile "sıfır değer"i ayırt etmek içindir.
type monitorRequest struct {
	Name             *string `json:"name"`
	Type             *string `json:"type"`
	URL              *string `json:"url"`
	Port             *int    `json:"port"`
	Interval         *int    `json:"interval"`
	Timeout          *int    `json:"timeout"`
	Retries          *int    `json:"retries"`
	NotificationIDs  *string `json:"notification_ids"`
	MaxRedirects     *int    `json:"max_redirects"`
	AcceptedCodes    *string `json:"accepted_codes"`
	Description      *string `json:"description"`
	DNSResolveType   *string `json:"dns_resolve_type"`
	DNSResolveServer *string `json:"dns_resolve_server"`
	Active           *bool   `json:"active"`
}

// applyTo istekteki alanları monitöre uygular (yalnızca izin verilen alanlar)
func (r *monitorRequest) applyTo(m *models.Monitor) {
	if r.Name != nil {
		m.Name = strings.TrimSpace(*r.Name)
	}
	if r.Type != nil {
		m.Type = models.MonitorType(strings.ToLower(strings.TrimSpace(*r.Type)))
	}
	if r.URL != nil {
		m.URL = strings.TrimSpace(*r.URL)
	}
	if r.Port != nil {
		m.Port = *r.Port
	}
	if r.Interval != nil {
		m.Interval = *r.Interval
	}
	if r.Timeout != nil {
		m.Timeout = *r.Timeout
	}
	if r.Retries != nil {
		m.Retries = *r.Retries
	}
	if r.MaxRedirects != nil {
		m.MaxRedirects = *r.MaxRedirects
	}
	if r.AcceptedCodes != nil {
		m.AcceptedCodes = strings.TrimSpace(*r.AcceptedCodes)
	}
	if r.Description != nil {
		m.Description = strings.TrimSpace(*r.Description)
	}
	if r.DNSResolveType != nil {
		m.DNSResolveType = strings.ToUpper(strings.TrimSpace(*r.DNSResolveType))
	}
	if r.DNSResolveServer != nil {
		m.DNSResolveServer = strings.TrimSpace(*r.DNSResolveServer)
	}
	if r.Active != nil {
		m.Active = *r.Active
	}
}

// APIGetMonitors kullanıcının monitörlerini döndür
func APIGetMonitors(c *gin.Context) {
	uid := currentUserID(c)
	var monitors []models.Monitor
	database.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&monitors)
	c.JSON(http.StatusOK, gin.H{"data": monitors})
}

// APIGetMonitor tek monitör (sahiplik kontrolü ile)
func APIGetMonitor(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mon})
}

// APICreateMonitor yeni monitör oluştur
func APICreateMonitor(c *gin.Context) {
	uid := currentUserID(c)

	var req monitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == nil || req.Type == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and type are required"})
		return
	}

	// Kullanıcı başına monitör sınırı
	var count int64
	database.DB.Model(&models.Monitor{}).Where("user_id = ?", uid).Count(&count)
	if limit := config.App.MaxMonitorsPerUser; limit > 0 && count >= int64(limit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "monitor limit reached (" + strconv.Itoa(limit) + ")"})
		return
	}

	mon := models.Monitor{
		Interval:      60,
		Timeout:       30,
		Retries:       1,
		MaxRedirects:  10,
		AcceptedCodes: "200-299",
	}
	req.applyTo(&mon)
	if mon.AcceptedCodes == "" {
		mon.AcceptedCodes = "200-299"
	}
	mon.UserID = uid
	mon.Active = true
	mon.Status = models.StatusPending

	if err := monitor.Validate(&mon); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.NotificationIDs != nil {
		ids, err := normalizeNotificationIDs(uid, *req.NotificationIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		mon.NotificationIDs = ids
	}

	// GORM, "default" etiketi olan alanlarda sıfır değeri (retries=0, max_redirects=0)
	// Create sırasında atlayıp veritabanı varsayılanını yazar; bu değerler ayrıca kaydedilir.
	retries, maxRedirects := mon.Retries, mon.MaxRedirects
	if err := database.DB.Create(&mon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create monitor"})
		return
	}
	if mon.Retries != retries || mon.MaxRedirects != maxRedirects {
		database.DB.Model(&mon).Updates(map[string]interface{}{"retries": retries, "max_redirects": maxRedirects})
		mon.Retries, mon.MaxRedirects = retries, maxRedirects
	}

	if monitor.GlobalManager != nil {
		monitor.GlobalManager.AddMonitor(&mon)
	}

	c.JSON(http.StatusCreated, gin.H{"data": mon, "message": "Monitor created successfully"})
}

// APIUpdateMonitor monitörü güncelle (sahiplik kontrolü ile)
func APIUpdateMonitor(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	var req monitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.applyTo(&mon)
	if err := monitor.Validate(&mon); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.NotificationIDs != nil {
		ids, err := normalizeNotificationIDs(uid, *req.NotificationIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		mon.NotificationIDs = ids
	}

	// Yalnızca kullanıcı tarafından değiştirilebilen sütunlar kaydedilir
	if err := database.DB.Model(&mon).Select(
		"name", "type", "url", "port", "interval", "timeout", "retries", "notification_ids",
		"max_redirects", "accepted_codes", "description", "dns_resolve_type", "dns_resolve_server", "active",
	).Updates(&mon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update monitor"})
		return
	}

	if monitor.GlobalManager != nil {
		if mon.Active {
			monitor.GlobalManager.AddMonitor(&mon)
		} else {
			monitor.GlobalManager.RemoveMonitor(mon.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": mon, "message": "Monitor updated successfully"})
}

// APIDeleteMonitor monitörü sil (sahiplik kontrolü ile)
func APIDeleteMonitor(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	if monitor.GlobalManager != nil {
		monitor.GlobalManager.RemoveMonitor(mon.ID)
	}

	database.DB.Where("monitor_id = ?", mon.ID).Delete(&models.Heartbeat{})
	database.DB.Delete(&mon)

	c.JSON(http.StatusOK, gin.H{"message": "Monitor deleted successfully"})
}

// APIToggleMonitor aktif/pasif değiştir (sahiplik kontrolü ile)
func APIToggleMonitor(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	mon.Active = !mon.Active
	database.DB.Model(&mon).Update("active", mon.Active)

	if monitor.GlobalManager != nil {
		if mon.Active {
			monitor.GlobalManager.AddMonitor(&mon)
		} else {
			monitor.GlobalManager.RemoveMonitor(mon.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": mon, "message": "Monitor toggled"})
}

// clampInt değeri [min, max] aralığına sığdırır; geçersizse varsayılanı döndürür
func clampInt(s string, def, min, max int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// APIGetHeartbeats heartbeat geçmişi (sahiplik kontrolü ile)
func APIGetHeartbeats(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	// Sahiplik kontrolü
	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	hours := clampInt(c.Query("hours"), 24, 1, 24*31)
	limit := clampInt(c.Query("limit"), 200, 1, 1000)
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var heartbeats []models.Heartbeat
	database.DB.Where("monitor_id = ? AND time > ?", mon.ID, since).
		Order("time DESC").
		Limit(limit).
		Find(&heartbeats)

	c.JSON(http.StatusOK, gin.H{"data": heartbeats})
}

// APIGetStats monitör istatistikleri (sahiplik kontrolü ile)
func APIGetStats(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var mon models.Monitor
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&mon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor not found"})
		return
	}

	uptime24h := calculateUptimeForPeriod(mon.ID, 24*time.Hour)
	uptime7d := calculateUptimeForPeriod(mon.ID, 7*24*time.Hour)
	uptime30d := calculateUptimeForPeriod(mon.ID, 30*24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"uptime_24h":  uptime24h,
			"uptime_7d":   uptime7d,
			"uptime_30d":  uptime30d,
			"avg_latency": mon.AvgLatency,
		},
	})
}

func calculateUptimeForPeriod(monitorID uint, period time.Duration) float64 {
	var total, up int64
	since := time.Now().Add(-period)

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
