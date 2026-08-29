package notification

import (
	"encoding/json"
	"fmt"
	"log"

	"gowatch/internal/database"
	"gowatch/internal/database/models"
)

// SendStatusChange durum değişikliğinde bildirimleri gönderir
func SendStatusChange(mon *models.Monitor, hb models.Heartbeat, previousStatus models.MonitorStatus) {
	if mon.NotificationIDs == "" {
		return
	}

	var notifIDs []uint
	if err := json.Unmarshal([]byte(mon.NotificationIDs), &notifIDs); err != nil {
		return
	}

	statusText := "🔴 DOWN"
	if hb.Status == models.StatusUp {
		statusText = "✅ UP"
	}

	title := fmt.Sprintf("[GoWatch] %s is %s", mon.Name, statusText)
	body := fmt.Sprintf("Monitor: %s\nStatus: %s\nMessage: %s\nLatency: %dms",
		mon.Name, statusText, hb.Msg, hb.Latency)

	for _, id := range notifIDs {
		var notif models.Notification
		if err := database.DB.First(&notif, id).Error; err != nil {
			continue
		}
		if !notif.Active {
			continue
		}

		var err error
		switch notif.Type {
		case "telegram":
			err = SendTelegram(&notif, title, body)
		default:
			log.Printf("Unknown notification type: %s", notif.Type)
			continue
		}

		if err != nil {
			log.Printf("Failed to send notification %s: %v", notif.Name, err)
		} else {
			log.Printf("Notification sent via %s: %s", notif.Type, notif.Name)
		}
	}
}
