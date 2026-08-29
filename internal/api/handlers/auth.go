package handlers

import (
	"net/http"
	"strings"

	"gowatch/internal/api/middleware"
	"gowatch/internal/database"
	"gowatch/internal/database/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetLoginPage login sayfasını göster
func GetLoginPage(c *gin.Context) {
	session, _ := middleware.Store.Get(c.Request, "gowatch-session")
	if session.Values["user_id"] != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login — GoWatch",
	})
}

// PostLogin giriş işlemini gerçekleştir
func PostLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Login — GoWatch",
			"error": "Invalid username or password",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Login — GoWatch",
			"error": "Invalid username or password",
		})
		return
	}

	session, _ := middleware.Store.Get(c.Request, "gowatch-session")
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, "/")
}

// PostLogout çıkış işlemi
func PostLogout(c *gin.Context) {
	session, _ := middleware.Store.Get(c.Request, "gowatch-session")
	session.Values["user_id"] = nil
	session.Options.MaxAge = -1
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/login")
}

// GetRegisterPage kayıt sayfasını göster
func GetRegisterPage(c *gin.Context) {
	session, _ := middleware.Store.Get(c.Request, "gowatch-session")
	if session.Values["user_id"] != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title": "Register — GoWatch",
	})
}

// PostRegister yeni kullanıcı oluştur
func PostRegister(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	renderError := func(msg string) {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Register — GoWatch",
			"error":    msg,
			"username": username,
		})
	}

	// Validasyon
	if len(username) < 3 {
		renderError("Username must be at least 3 characters")
		return
	}
	if len(username) > 32 {
		renderError("Username must be at most 32 characters")
		return
	}
	if len(password) < 6 {
		renderError("Password must be at least 6 characters")
		return
	}
	if password != confirmPassword {
		renderError("Passwords do not match")
		return
	}

	// Kullanıcı adı müsait mi?
	var existing models.User
	if err := database.DB.Where("username = ?", username).First(&existing).Error; err == nil {
		renderError("Username already taken")
		return
	}

	// Şifreyi hashle
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		renderError("Something went wrong, please try again")
		return
	}

	newUser := models.User{
		Username: username,
		Password: string(hashed),
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		renderError("Could not create account, please try again")
		return
	}

	// Otomatik giriş yap
	session, _ := middleware.Store.Get(c.Request, "gowatch-session")
	session.Values["user_id"] = newUser.ID
	session.Values["username"] = newUser.Username
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, "/")
}

// APIUpdateUsername kullanıcı adını güncelle
func APIUpdateUsername(c *gin.Context) {
	uid := currentUserID(c)

	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < 3 || len(username) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be 3-32 characters"})
		return
	}

	// Başka kullanıcıda var mı?
	var existing models.User
	if err := database.DB.Where("username = ? AND id != ?", username, uid).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", uid).Update("username", username).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update username"})
		return
	}

	// Session güncelle
	session, _ := middleware.Store.Get(c.Request, "gowatch-session")
	session.Values["username"] = username
	session.Save(c.Request, c.Writer)

	c.JSON(http.StatusOK, gin.H{"message": "Username updated", "username": username})
}

// APIUpdatePassword şifreyi güncelle
func APIUpdatePassword(c *gin.Context) {
	uid := currentUserID(c)

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 6 characters"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	database.DB.Model(&models.User{}).Where("id = ?", uid).Update("password", string(hashed))
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
