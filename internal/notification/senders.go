package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gowatch/internal/database/models"
)

// TelegramConfig telegram bildirim konfigürasyonu
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

var (
	// Bot token formatı: <sayısal bot id>:<gizli anahtar>
	telegramTokenRe  = regexp.MustCompile(`^\d{5,}:[A-Za-z0-9_-]{20,}$`)
	telegramChatIDRe = regexp.MustCompile(`^(-?\d{1,20}|@[A-Za-z0-9_]{5,32})$`)
	httpClient       = &http.Client{Timeout: 10 * time.Second}
)

// ValidateConfig bildirim türünü ve konfigürasyonunu doğrular
func ValidateConfig(notifType, config string) error {
	switch notifType {
	case "telegram":
		var cfg TelegramConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid telegram config")
		}
		if !telegramTokenRe.MatchString(strings.TrimSpace(cfg.BotToken)) {
			return errors.New("invalid telegram bot token")
		}
		if !telegramChatIDRe.MatchString(strings.TrimSpace(cfg.ChatID)) {
			return errors.New("invalid telegram chat id")
		}
		return nil
	default:
		return errors.New("unsupported notification type (supported: telegram)")
	}
}

// SendTelegram Telegram Bot API ile mesaj gönder
func SendTelegram(notif *models.Notification, title, body string) error {
	var cfg TelegramConfig
	if err := json.Unmarshal([]byte(notif.Config), &cfg); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}
	if !telegramTokenRe.MatchString(cfg.BotToken) {
		return errors.New("invalid telegram bot token")
	}

	// Kullanıcı girdisi (monitör adı, hata mesajı) HTML olarak kaçışlanır
	text := fmt.Sprintf("<b>%s</b>\n\n%s", html.EscapeString(title), html.EscapeString(body))
	payload := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)

	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		// Hata mesajı URL'yi (dolayısıyla bot token'ı) içerebilir, loglara sızmasın
		return errors.New(strings.ReplaceAll(err.Error(), cfg.BotToken, "***"))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}
	return nil
}
