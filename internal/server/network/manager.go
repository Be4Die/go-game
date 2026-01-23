package network

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"game/internal/server/game"
	"game/internal/shared"

	"github.com/gorilla/websocket"
)

type Manager struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex

	world *game.World

	healthCheckTicker *time.Ticker
	done              chan bool

	tickRate time.Duration
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16384,
	WriteBufferSize: 16384,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewManager() *Manager {
	m := &Manager{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		world:      game.NewWorld(),
		done:       make(chan bool),
		tickRate:   16 * time.Millisecond,
	}

	m.healthCheckTicker = time.NewTicker(30 * time.Second)
	go m.healthCheck()

	return m
}

func (m *Manager) Run() {
	gameTicker := time.NewTicker(m.tickRate)
	defer func() {
		gameTicker.Stop()
		m.healthCheckTicker.Stop()
		m.done <- true
	}()

	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client] = true
			m.mu.Unlock()
			log.Printf("Client connected: %s", client.id)

			m.sendWorldState(client)

		case client := <-m.unregister:
			m.handleDisconnect(client)

		case message := <-m.broadcast:
			m.broadcastToAll(message)

		case <-gameTicker.C:
			// Update game world
			m.world.Update(float32(m.tickRate.Seconds()))
			// Broadcast world state
			m.broadcastWorldState()

		case <-m.done:
			return
		}
	}
}

func (m *Manager) Stop() {
	m.done <- true
	m.healthCheckTicker.Stop()

	m.mu.Lock()
	defer m.mu.Unlock()

	for client := range m.clients {
		client.Close()
		delete(m.clients, client)
	}

	log.Println("Server stopped")
}

func (m *Manager) healthCheck() {
	for {
		select {
		case <-m.healthCheckTicker.C:
			m.mu.RLock()
			clientsToRemove := []*Client{}

			for client := range m.clients {
				if time.Since(client.lastPing) > pongWait*2 {
					log.Printf("Client %s failed health check, marking for removal", client.id)
					clientsToRemove = append(clientsToRemove, client)
				}
			}
			m.mu.RUnlock()

			for _, client := range clientsToRemove {
				m.unregister <- client
			}

			log.Printf("Health check: %d clients connected", len(m.clients))

		case <-m.done:
			return
		}
	}
}

func (m *Manager) handleClientMessage(client *Client, message []byte) {
	var msg shared.Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	switch msg.Type {
	case shared.MessageTypeJoin:
		var joinMsg shared.JoinMessage
		if err := json.Unmarshal(msg.Data, &joinMsg); err != nil {
			log.Printf("Error unmarshaling join message: %v", err)
			return
		}

		client.nickname = joinMsg.Nickname
		m.world.AddPlayer(client.id, joinMsg.Nickname, joinMsg.Model)

		m.broadcastNewPlayer(client.id, joinMsg.Nickname, joinMsg.Model)
		log.Printf("Player joined: %s (%s) model=%s", joinMsg.Nickname, client.id, joinMsg.Model)

	case shared.MessageTypeInput:
		var inputMsg shared.InputMessage
		if err := json.Unmarshal(msg.Data, &inputMsg); err != nil {
			log.Printf("Error unmarshaling input message: %v", err)
			return
		}

		// Ensure the input is for this client
		if inputMsg.PlayerID == "" {
			inputMsg.PlayerID = client.id
		}

		m.world.ProcessInput(client.id, inputMsg)

	case shared.MessageTypeHeartbeat:
		client.isAlive = true
		client.lastPing = time.Now()

		hbResponse := shared.HeartbeatMessage{
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		}

		data, _ := json.Marshal(hbResponse)
		response := shared.Message{
			Type: shared.MessageTypeHeartbeat,
			Data: data,
		}

		msgData, _ := json.Marshal(response)
		client.send <- msgData

	case shared.MessageTypeLeave:
		log.Printf("Client requested disconnect: %s", client.id)
		m.unregister <- client
	}
}

func (m *Manager) handleDisconnect(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[client]; ok {
		delete(m.clients, client)
		close(client.send)

		m.world.RemovePlayer(client.id)

		// Broadcast player leave
		m.broadcastPlayerLeave(client.id, client.nickname)

		log.Printf("Client disconnected: %s (%s)", client.nickname, client.id)
	}
}

func (m *Manager) broadcastToAll(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		select {
		case client.send <- message:
		default:
			log.Printf("Client %s buffer full, forcing disconnect", client.id)
			go func(c *Client) {
				m.unregister <- c
			}(client)
		}
	}
}

func (m *Manager) sendWorldState(client *Client) {
	players := m.world.GetAllPlayers()

	worldState := shared.WorldStateMessage{
		Players: players,
	}

	data, err := json.Marshal(worldState)
	if err != nil {
		log.Printf("Error marshaling world state: %v", err)
		return
	}

	message := shared.Message{
		Type:     shared.MessageTypeWorldState,
		Data:     data,
		PlayerID: client.id,
	}

	msgData, _ := json.Marshal(message)
	client.send <- msgData
}

func (m *Manager) broadcastWorldState() {
	players := m.world.GetAllPlayers()
	if len(players) == 0 {
		return
	}

	worldState := shared.WorldStateMessage{
		Players:    players,
		ServerTime: time.Now().UnixNano() / int64(time.Millisecond),
	}

	data, err := json.Marshal(worldState)
	if err != nil {
		log.Printf("Error marshaling world state: %v", err)
		return
	}

	message := shared.Message{
		Type: shared.MessageTypeWorldState,
		Data: data,
	}

	msgData, _ := json.Marshal(message)

	select {
	case m.broadcast <- msgData:
	default:
		log.Println("Broadcast channel full, dropping world state message")
	}
}

func (m *Manager) broadcastNewPlayer(playerID, nickname, model string) {
	newPlayerMsg := shared.NewPlayerMessage{
		PlayerID: playerID,
		Nickname: nickname,
		Model:    model,
	}

	data, _ := json.Marshal(newPlayerMsg)
	message := shared.Message{
		Type: shared.MessageTypeNewPlayer,
		Data: data,
	}

	msgData, _ := json.Marshal(message)

	select {
	case m.broadcast <- msgData:
	default:
		log.Println("Broadcast channel full, dropping new player message")
	}
}

func (m *Manager) broadcastPlayerLeave(playerID, nickname string) {
	leaveMsg := shared.PlayerLeftMessage{
		PlayerID: playerID,
		Nickname: nickname,
		Reason:   "Disconnected",
	}

	data, _ := json.Marshal(leaveMsg)
	message := shared.Message{
		Type: shared.MessageTypePlayerLeft,
		Data: data,
	}

	msgData, _ := json.Marshal(message)

	select {
	case m.broadcast <- msgData:
	default:
		log.Println("Broadcast channel full, dropping player left message")
	}
}

func ServeWs(m *Manager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m)

	welcomeMsg := shared.WelcomeMessage{
		PlayerID: client.id,
		Message:  "Welcome to the game server",
	}

	data, _ := json.Marshal(welcomeMsg)
	msg := shared.Message{
		Type: shared.MessageTypeWelcome,
		Data: data,
	}

	msgData, _ := json.Marshal(msg)
	client.send <- msgData

	m.register <- client

	go client.writePump()
	go client.readPump()
}
