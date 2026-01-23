package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"game/internal/shared"

	"github.com/gorilla/websocket"
)

type Server struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex

	players map[string]*shared.PlayerState

	healthCheckTicker *time.Ticker
	done              chan bool

	gameTickTime time.Time
	tickRate     time.Duration
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16384,
	WriteBufferSize: 16384,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer() *Server {
	s := &Server{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		players:    make(map[string]*shared.PlayerState),
		done:       make(chan bool),
		tickRate:   16 * time.Millisecond,
	}

	s.healthCheckTicker = time.NewTicker(30 * time.Second)
	go s.healthCheck()

	return s
}

func (s *Server) Run() {
	gameTicker := time.NewTicker(s.tickRate)
	defer func() {
		gameTicker.Stop()
		s.healthCheckTicker.Stop()
		s.done <- true
	}()

	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected: %s", client.id)

			s.sendWorldState(client)

		case client := <-s.unregister:
			s.handleDisconnect(client)

		case message := <-s.broadcast:
			s.broadcastToAll(message)

		case <-gameTicker.C:
			s.gameTickTime = time.Now()
			s.updateGameState()
			s.processPlayerInputs()

		case <-s.done:
			return
		}
	}
}

func (s *Server) processPlayerInputs() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, player := range s.players {
		if player.LastInput.IsEmpty() {
			continue
		}

		s.applyPlayerInput(player, player.LastInput)
		player.LastInput = shared.InputKeys{}
	}
}

func (s *Server) applyPlayerInput(player *shared.PlayerState, input shared.InputKeys) {
	speed := float32(5.0)
	if input.Sprint {
		speed = 10.0
	}

	deltaTime := float32(s.tickRate.Seconds())

	if input.Forward {
		player.Position.Z -= speed * deltaTime
	}
	if input.Backward {
		player.Position.Z += speed * deltaTime
	}
	if input.Left {
		player.Position.X -= speed * deltaTime
	}
	if input.Right {
		player.Position.X += speed * deltaTime
	}

	if input.Jump && !player.IsJumping && player.IsGrounded {
		player.Velocity.Y = 10.0
		player.IsJumping = true
	}

	gravity := float32(20.0)
	player.Velocity.Y -= gravity * deltaTime
	player.Position.Y += player.Velocity.Y * deltaTime

	if player.Position.Y < 0 {
		player.Position.Y = 0
		player.Velocity.Y = 0
		player.IsJumping = false
		player.IsGrounded = true
	}

	player.LastUpdate = time.Now()

	if input.Forward || input.Backward || input.Left || input.Right {
		player.Animation = 2
		if input.Sprint {
			player.Animation = 3
		}
	} else {
		player.Animation = 1
	}

	if player.IsJumping {
		player.Animation = 4
	}
}

func (s *Server) handleDisconnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[client]; ok {
		delete(s.clients, client)
		close(client.send)

		if playerState, exists := s.players[client.id]; exists {
			// Удаляем игрока сразу
			delete(s.players, client.id)

			// Отправляем сообщение о выходе игрока
			s.broadcastPlayerLeave(client.id, playerState.Nickname)

			log.Printf("Client disconnected: %s (%s)", client.nickname, client.id)
		}
	}
}

func (s *Server) broadcastToAll(message []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		select {
		case client.send <- message:
		default:
			log.Printf("Client %s buffer full, forcing disconnect", client.id)
			go func(c *Client) {
				s.unregister <- c
			}(client)
		}
	}
}

func (s *Server) sendWorldState(client *Client) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worldState := shared.WorldStateMessage{
		Players: make([]shared.PlayerState, 0, len(s.players)),
	}

	for _, player := range s.players {
		worldState.Players = append(worldState.Players, *player)
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

func (s *Server) broadcastPlayerLeave(playerID, nickname string) {
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

	// Неблокирующая отправка
	select {
	case s.broadcast <- msgData:
		// Сообщение отправлено
	default:
		log.Println("Broadcast channel full, dropping player left message")
	}
}

func (s *Server) updateGameState() {
	s.mu.RLock()
	players := make([]shared.PlayerState, 0, len(s.players))
	for _, player := range s.players {
		players = append(players, *player)
	}
	s.mu.RUnlock()

	if len(players) > 0 {
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

		// Неблокирующая отправка
		select {
		case s.broadcast <- msgData:
			// Сообщение отправлено
		default:
			log.Println("Broadcast channel full, dropping world state message")
		}
	}
}

func (s *Server) healthCheck() {
	for {
		select {
		case <-s.healthCheckTicker.C:
			s.mu.RLock()
			clientsToRemove := []*Client{}

			for client := range s.clients {
				if time.Since(client.lastPing) > pongWait*2 {
					log.Printf("Client %s failed health check, marking for removal", client.id)
					clientsToRemove = append(clientsToRemove, client)
				}
			}
			s.mu.RUnlock()

			for _, client := range clientsToRemove {
				s.unregister <- client
			}

			log.Printf("Health check: %d clients connected", len(s.clients))

		case <-s.done:
			return
		}
	}
}

func (s *Server) handleClientMessage(client *Client, message []byte) {
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

		s.mu.Lock()
		client.nickname = joinMsg.Nickname

		s.players[client.id] = &shared.PlayerState{
			PlayerID:   client.id,
			Nickname:   joinMsg.Nickname,
			Position:   shared.Vector3{X: 0, Y: 0, Z: 0},
			Rotation:   0,
			Velocity:   shared.Vector3{X: 0, Y: 0, Z: 0},
			Animation:  1,
			IsGrounded: true,
			IsJumping:  false,
			JoinedAt:   time.Now(),
			LastUpdate: time.Now(),
		}
		s.mu.Unlock()

		s.broadcastNewPlayer(client.id, joinMsg.Nickname)

		log.Printf("Player joined: %s (%s)", joinMsg.Nickname, client.id)

	case shared.MessageTypeInput:
		var inputMsg shared.InputMessage
		if err := json.Unmarshal(msg.Data, &inputMsg); err != nil {
			log.Printf("Error unmarshaling input message: %v", err)
			return
		}

		s.mu.Lock()
		if player, ok := s.players[client.id]; ok {
			player.LastInput = inputMsg.Keys
			player.Rotation = inputMsg.Rotation
			player.Animation = inputMsg.Animation
			player.Position = inputMsg.Position
		}
		s.mu.Unlock()

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
		s.unregister <- client
	}
}

func (s *Server) broadcastNewPlayer(playerID, nickname string) {
	newPlayerMsg := shared.NewPlayerMessage{
		PlayerID: playerID,
		Nickname: nickname,
	}

	data, _ := json.Marshal(newPlayerMsg)
	message := shared.Message{
		Type: shared.MessageTypeNewPlayer,
		Data: data,
	}

	msgData, _ := json.Marshal(message)

	// Неблокирующая отправка
	select {
	case s.broadcast <- msgData:
		// Сообщение отправлено
	default:
		log.Println("Broadcast channel full, dropping new player message")
	}
}

func ServeWs(s *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, s)

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

	s.register <- client

	go client.writePump()
	go client.readPump()
}

func (s *Server) Stop() {
	s.done <- true
	s.healthCheckTicker.Stop()

	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		client.Close()
		delete(s.clients, client)
	}

	log.Println("Server stopped")
}
