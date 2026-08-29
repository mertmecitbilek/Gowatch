package handlers

import (
	"net/http"

	"gowatch/internal/database"
	"gowatch/internal/database/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	ws "gowatch/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Aynı host'tan gelen bağlantılara izin ver
		return r.Header.Get("Origin") == "" ||
			r.Header.Get("Origin") == "http://"+r.Host ||
			r.Header.Get("Origin") == "https://"+r.Host
	},
}

// ServeWs WebSocket bağlantısını yönetir
func ServeWs(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	ws.GlobalHub.ServeWs(conn)
}

// GetStatusPage public status page
func GetStatusPage(c *gin.Context) {
	var monitors []models.Monitor
	database.DB.Where("active = ?", true).Find(&monitors)

	// Son heartbeat'leri al (her monitör için)
	type MonitorWithHeartbeats struct {
		models.Monitor
		RecentHeartbeats []models.Heartbeat
	}

	var monitorsWithHB []MonitorWithHeartbeats
	for _, m := range monitors {
		var hbs []models.Heartbeat
		database.DB.Where("monitor_id = ?", m.ID).
			Order("time DESC").
			Limit(90).
			Find(&hbs)
		monitorsWithHB = append(monitorsWithHB, MonitorWithHeartbeats{
			Monitor:          m,
			RecentHeartbeats: hbs,
		})
	}

	c.HTML(http.StatusOK, "status_page.html", gin.H{
		"title":    "Status Page — GoWatch",
		"monitors": monitorsWithHB,
	})
}
