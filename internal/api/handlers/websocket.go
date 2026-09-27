package handlers

import (
	"net/http"

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

// ServeWs WebSocket bağlantısını oturumdaki kullanıcıya bağlar
func ServeWs(c *gin.Context) {
	uid := currentUserID(c)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	ws.GlobalHub.ServeWs(conn, uid)
}
