package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"gowatch/internal/database"
	"gowatch/internal/database/models"
	"gowatch/internal/notification"

	"github.com/gin-gonic/gin"
)

// notificationRequest oluşturma/güncelleme isteğinde kabul edilen alanlar
type notificationRequest struct {
	Name   *string `json:"name"`
	Type   *string `json:"type"`
	Config *string `json:"config"`
	Active *bool   `json:"active"`
}

func validateNotification(n *models.Notification) error {
	if n.Name == "" || utf8.RuneCountInString(n.Name) > 100 {
		return errors.New("name must be 1-100 characters")
	}
	return notification.ValidateConfig(n.Type, n.Config)
}

// APIGetNotifications kullanıcının bildirim kanallarını listele
func APIGetNotifications(c *gin.Context) {
	uid := currentUserID(c)
	var notifs []models.Notification
	database.DB.Where("user_id = ?", uid).Find(&notifs)
	c.JSON(http.StatusOK, gin.H{"data": notifs})
}

// APICreateNotification yeni bildirim kanalı oluştur
func APICreateNotification(c *gin.Context) {
	uid := currentUserID(c)

	var req notificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == nil || req.Type == nil || req.Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, type and config are required"})
		return
	}

	notif := models.Notification{
		UserID: uid,
		Name:   strings.TrimSpace(*req.Name),
		Type:   strings.TrimSpace(*req.Type),
		Config: *req.Config,
		Active: true,
	}
	if err := validateNotification(&notif); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification channel"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": notif})
}

// APIUpdateNotification bildirim kanalını güncelle (sahiplik kontrolü ile)
func APIUpdateNotification(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var notif models.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&notif).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	var req notificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != nil {
		notif.Name = strings.TrimSpace(*req.Name)
	}
	if req.Type != nil {
		notif.Type = strings.TrimSpace(*req.Type)
	}
	if req.Config != nil {
		notif.Config = *req.Config
	}
	if req.Active != nil {
		notif.Active = *req.Active
	}
	if err := validateNotification(&notif); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&notif).Select("name", "type", "config", "active").Updates(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": notif})
}

// APIDeleteNotification bildirim kanalını sil (sahiplik kontrolü ile)
func APIDeleteNotification(c *gin.Context) {
	uid := currentUserID(c)
	id := c.Param("id")

	var notif models.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&notif).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	database.DB.Delete(&notif)
	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// GetSettingsPage ayarlar sayfası
func GetSettingsPage(c *gin.Context) {
	uid := currentUserID(c)
	var notifs []models.Notification
	database.DB.Where("user_id = ?", uid).Find(&notifs)

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"title":         "Settings — GoWatch",
		"notifications": notifs,
		"username":      c.GetString("username"),
		"userInitial":   userInitial(c),
	})
}
