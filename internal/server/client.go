package server

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 16384 // Увеличил с 512 до 16KB
)

type Client struct {
	id       string
	conn     *websocket.Conn
	server   *Server
	send     chan []byte
	nickname string
	isAlive  bool
	lastPing time.Time
}

func NewClient(conn *websocket.Conn, server *Server) *Client {
	return &Client{
		id:       uuid.New().String(),
		conn:     conn,
		server:   server,
		send:     make(chan []byte, 256),
		isAlive:  true,
		lastPing: time.Now(),
	}
}

func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.isAlive = true
		c.lastPing = time.Now()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Обрабатываем сообщение от клиента
		c.server.handleClientMessage(c, message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Сервер закрыл канал
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Добавляем другие сообщения в буфер
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			c.lastPing = time.Now()

			// Проверяем живой ли клиент (если не получали pong слишком долго)
			if time.Since(c.lastPing) > pongWait*2 {
				log.Printf("Client %s seems dead, disconnecting", c.id)
				c.isAlive = false
				return
			}
		}
	}
}

// SendMessage отправляет сообщение клиенту
func (c *Client) SendMessage(messageType int, data []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, data)
}

// Close закрывает соединение
func (c *Client) Close() {
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	time.Sleep(100 * time.Millisecond)
	c.conn.Close()
}
