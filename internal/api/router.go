package api

import (
	"github.com/gin-gonic/gin"

	"gowatch/internal/api/handlers"
	"gowatch/internal/api/middleware"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")

	// Public Routes
	r.GET("/login", handlers.GetLoginPage)
	r.POST("/login", handlers.PostLogin)
	r.GET("/register", handlers.GetRegisterPage)
	r.POST("/register", handlers.PostRegister)

	// Protected Routes
	auth := r.Group("/")
	auth.Use(middleware.AuthRequired())
	{
		auth.GET("/ws", handlers.ServeWs)
		auth.GET("/", handlers.GetDashboard)
		auth.GET("/monitors/:id", handlers.GetMonitorDetail)
		auth.GET("/settings", handlers.GetSettingsPage)
		auth.POST("/logout", handlers.PostLogout)

		api := auth.Group("/api")
		{
			api.GET("/monitors", handlers.APIGetMonitors)
			api.POST("/monitors", handlers.APICreateMonitor)
			api.GET("/monitors/:id", handlers.APIGetMonitor)
			api.PUT("/monitors/:id", handlers.APIUpdateMonitor)
			api.DELETE("/monitors/:id", handlers.APIDeleteMonitor)
			api.POST("/monitors/:id/toggle", handlers.APIToggleMonitor)
			api.GET("/monitors/:id/heartbeats", handlers.APIGetHeartbeats)
			api.GET("/monitors/:id/stats", handlers.APIGetStats)

			api.GET("/notifications", handlers.APIGetNotifications)
			api.POST("/notifications", handlers.APICreateNotification)
			api.PUT("/notifications/:id", handlers.APIUpdateNotification)
			api.DELETE("/notifications/:id", handlers.APIDeleteNotification)

			// Profil
			api.PUT("/profile/username", handlers.APIUpdateUsername)
			api.PUT("/profile/password", handlers.APIUpdatePassword)
		}
	}

	return r
}
