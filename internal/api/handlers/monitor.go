package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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
		"username":     c.GetString("username"),
		"userInitial":  userInitial(c),
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

	c.HTML(http.StatusOK, "monitor_detail.html", gin.H{
		"title":      mon.Name + " — GoWatch",
		"monitor":    mon,
		"heartbeats": heartbeats,
		"username":   c.GetString("username"),
		"userInitial": userInitial(c),
	})
}

// --- API Endpoints ---

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

	var req struct {
		Name             string `json:"name" binding:"required"`
		Type             string `json:"type" binding:"required"`
		URL              string `json:"url"`
		Port             int    `json:"port"`
		Interval         int    `json:"interval"`
		Timeout          int    `json:"timeout"`
		Retries          int    `json:"retries"`
		NotificationIDs  string `json:"notification_ids"`
		MaxRedirects     int    `json:"max_redirects"`
		AcceptedCodes    string `json:"accepted_codes"`
		Description      string `json:"description"`
		DNSResolveType   string `json:"dns_resolve_type"`
		DNSResolveServer string `json:"dns_resolve_server"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Interval == 0 {
		req.Interval = 60
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}
	if req.AcceptedCodes == "" {
		req.AcceptedCodes = "200-299"
	}

	mon := models.Monitor{
		UserID:           uid,
		Name:             req.Name,
		Type:             models.MonitorType(req.Type),
		URL:              req.URL,
		Port:             req.Port,
		Interval:         req.Interval,
		Timeout:          req.Timeout,
		Retries:          req.Retries,
		Active:           true,
		Status:           models.StatusPending,
		NotificationIDs:  req.NotificationIDs,
		MaxRedirects:     req.MaxRedirects,
		AcceptedCodes:    req.AcceptedCodes,
		Description:      req.Description,
		DNSResolveType:   req.DNSResolveType,
		DNSResolveServer: req.DNSResolveServer,
	}

	if err := database.DB.Create(&mon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// user_id değiştirilemesin
	delete(req, "user_id")

	if err := database.DB.Model(&mon).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.First(&mon, id)

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
	database.DB.Save(&mon)

	if monitor.GlobalManager != nil {
		if mon.Active {
			monitor.GlobalManager.AddMonitor(&mon)
		} else {
			monitor.GlobalManager.RemoveMonitor(mon.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": mon, "message": "Monitor toggled"})
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

	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var heartbeats []models.Heartbeat
	database.DB.Where("monitor_id = ? AND time > ?", id, since).
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
