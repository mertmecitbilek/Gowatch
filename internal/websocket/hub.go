package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message WebSocket mesaj yapısı
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client tek bir WebSocket bağlantısı
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID uint
}

// outgoing belirli bir kullanıcıya gidecek mesaj
type outgoing struct {
	userID uint
	data   []byte
}

// Hub tüm WebSocket istemcilerini yönetir
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan outgoing
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

var GlobalHub *Hub

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan outgoing, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			total := len(h.clients)
			h.mu.Unlock()
			log.Printf("WebSocket client connected. Total: %d", total)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			total := len(h.clients)
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total: %d", total)

		case msg := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				// Kullanıcılar yalnızca kendi monitörlerinin mesajlarını alır
				if client.userID != msg.userID {
					continue
				}
				select {
				case client.send <- msg.data:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SendToUser yalnızca verilen kullanıcının bağlı istemcilerine mesaj gönderir
func (h *Hub) SendToUser(userID uint, msgType string, payload interface{}) {
	data, err := json.Marshal(Message{Type: msgType, Payload: payload})
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}
	select {
	case h.broadcast <- outgoing{userID: userID, data: data}:
	default:
		log.Println("WebSocket broadcast channel full, dropping message")
	}
}

// ServeWs WebSocket bağlantısını kullanıcıya bağlayarak başlatır
func (h *Hub) ServeWs(conn *websocket.Conn, userID uint) {
	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
