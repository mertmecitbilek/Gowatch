package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gowatch/internal/api"
	"gowatch/internal/api/middleware"
	"gowatch/internal/config"
	"gowatch/internal/database"
	"gowatch/internal/monitor"
	ws "gowatch/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// Config yükle
	config.Load()

	// Gin modu
	gin.SetMode(config.App.GinMode)

	// Veritabanı bağlan
	database.Connect()

	// Session store başlat
	middleware.InitSession(config.App.SessionSecret, config.App.CookieSecure)

	// WebSocket hub başlat
	ws.GlobalHub = ws.NewHub()
	go ws.GlobalHub.Run()

	// Monitor manager başlat
	monitor.GlobalManager = monitor.NewManager()
	monitor.GlobalManager.Start()

	// Router kur
	r := api.SetupRouter()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := ":" + config.App.Port
		log.Printf("🚀 GoWatch started on http://localhost%s", addr)
		log.Printf("📊 Dashboard: http://localhost%s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down GoWatch...")
	monitor.GlobalManager.Stop()
	log.Println("GoWatch stopped gracefully")
}
