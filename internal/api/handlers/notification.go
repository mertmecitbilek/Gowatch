package handlers

import (
	"net/http"

	"gowatch/internal/database"
	"gowatch/internal/database/models"

	"github.com/gin-gonic/gin"
)

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

	var req struct {
		Name   string `json:"name" binding:"required"`
		Type   string `json:"type" binding:"required"`
		Config string `json:"config" binding:"required"`
		Active bool   `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notif := models.Notification{
		UserID: uid,
		Name:   req.Name,
		Type:   req.Type,
		Config: req.Config,
		Active: true,
	}

	if err := database.DB.Create(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delete(req, "user_id")

	database.DB.Model(&notif).Updates(req)
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
		"username":       c.GetString("username"),
		"userInitial":    userInitial(c),
	})
}
