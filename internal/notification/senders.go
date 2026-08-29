package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"gowatch/internal/database/models"
)

// TelegramConfig telegram bildirim konfigürasyonu
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

// SendTelegram Telegram Bot API ile mesaj gönder
func SendTelegram(notif *models.Notification, title, body string) error {
	var cfg TelegramConfig
	if err := json.Unmarshal([]byte(notif.Config), &cfg); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}

	text := fmt.Sprintf("*%s*\n\n%s", title, body)
	payload := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}
	return nil
}
