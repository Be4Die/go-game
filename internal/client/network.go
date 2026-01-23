package client

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"game/internal/shared"

	"github.com/gorilla/websocket"
)

const (
	clientWriteWait  = 10 * time.Second
	clientPongWait   = 60 * time.Second
	clientPingPeriod = (clientPongWait * 9) / 10
	clientMaxMessage = 16384
)

type Network struct {
	conn        *websocket.Conn
	sendChan    chan []byte
	receiveChan chan []byte
	mu          sync.RWMutex
	isConnected bool
	isClosing   bool
	container   *DataContainer

	heartbeatTicker *time.Ticker
	lastPong        time.Time
	latency         time.Duration
	inputSendTicker *time.Ticker
}

func NewNetwork(conn *websocket.Conn, container *DataContainer) *Network {
	n := &Network{
		conn:        conn,
		sendChan:    make(chan []byte, 256),
		receiveChan: make(chan []byte, 256),
		isConnected: true,
		container:   container,
		lastPong:    time.Now(),
		latency:     0,
	}

	conn.SetPongHandler(func(appData string) error {
		n.lastPong = time.Now()
		n.latency = time.Since(n.lastPong)
		return nil
	})

	go n.writePump()
	go n.readPump()
	go n.heartbeatLoop()
	go n.processMessages()

	n.inputSendTicker = time.NewTicker(16 * time.Millisecond)
	go n.inputSendLoop()

	return n
}

func (n *Network) Send(data []byte) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.isConnected || n.isClosing {
		return nil
	}

	select {
	case n.sendChan <- data:
		return nil
	default:
		log.Println("WebSocket send buffer full, dropping message")
		return nil
	}
}

func (n *Network) SendInput(input shared.InputMessage) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	message := shared.Message{
		Type: shared.MessageTypeInput,
		Data: data,
	}

	msgData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return n.Send(msgData)
}

func (n *Network) inputSendLoop() {
	for range n.inputSendTicker.C {
		if !n.IsConnected() {
			return
		}
	}
}

func (n *Network) writePump() {
	ticker := time.NewTicker(clientPingPeriod)
	defer func() {
		ticker.Stop()
		n.closeConnection()
	}()

	for {
		select {
		case message, ok := <-n.sendChan:
			if !ok {
				n.sendCloseMessage()
				return
			}

			n.conn.SetWriteDeadline(time.Now().Add(clientWriteWait))
			err := n.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			n.conn.SetWriteDeadline(time.Now().Add(clientWriteWait))
			if err := n.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping error: %v", err)
				return
			}

			if time.Since(n.lastPong) > clientPongWait*2 {
				log.Println("No pong received for too long, disconnecting")
				n.container.NetworkError = "Connection timeout"
				n.closeConnection()
				return
			}
		}
	}
}

func (n *Network) readPump() {
	defer n.closeConnection()

	n.conn.SetReadLimit(clientMaxMessage)
	n.conn.SetReadDeadline(time.Now().Add(clientPongWait))
	n.conn.SetPongHandler(func(appData string) error {
		n.conn.SetReadDeadline(time.Now().Add(clientPongWait))
		n.lastPong = time.Now()
		return nil
	})

	for {
		_, message, err := n.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}

			if !n.isClosing {
				n.container.NetworkError = "Disconnected from server"
				n.container.GameState = GameStateMenu
			}

			break
		}

		select {
		case n.receiveChan <- message:
		default:
			log.Println("Receive buffer full, dropping message")
		}
	}
}

func (n *Network) heartbeatLoop() {
	n.heartbeatTicker = time.NewTicker(30 * time.Second)
	defer n.heartbeatTicker.Stop()

	for {
		select {
		case <-n.heartbeatTicker.C:
			if !n.isConnected {
				return
			}

			hbMsg := shared.HeartbeatMessage{
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			}

			data, err := json.Marshal(hbMsg)
			if err != nil {
				log.Printf("Failed to marshal heartbeat: %v", err)
				continue
			}

			message := shared.Message{
				Type: shared.MessageTypeHeartbeat,
				Data: data,
			}

			msgData, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			n.Send(msgData)

			if time.Since(n.lastPong) > time.Minute {
				log.Println("Heartbeat failed, server seems dead")
				n.container.NetworkError = "Server connection lost"
				n.container.GameState = GameStateMenu
				n.closeConnection()
				return
			}
		}
	}
}

func (n *Network) processMessages() {
	for message := range n.receiveChan {
		n.handleMessage(message)
	}
}

func (n *Network) handleMessage(data []byte) {
	var msg shared.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	switch msg.Type {
	case shared.MessageTypeWelcome:
		var welcomeMsg shared.WelcomeMessage
		if err := json.Unmarshal(msg.Data, &welcomeMsg); err != nil {
			log.Printf("Error unmarshaling welcome message: %v", err)
			return
		}

		log.Printf("Connected to server: %s", welcomeMsg.Message)
		n.container.PlayerID = welcomeMsg.PlayerID

	case shared.MessageTypeWorldState:
		var worldState shared.WorldStateMessage
		if err := json.Unmarshal(msg.Data, &worldState); err != nil {
			log.Printf("Error unmarshaling world state: %v", err)
			return
		}

		n.updateRemotePlayers(worldState.Players)

	case shared.MessageTypeNewPlayer:
		var newPlayer shared.NewPlayerMessage
		if err := json.Unmarshal(msg.Data, &newPlayer); err != nil {
			log.Printf("Error unmarshaling new player message: %v", err)
			return
		}

		log.Printf("New player joined: %s", newPlayer.Nickname)

	case shared.MessageTypeLeave:
		var leaveMsg shared.LeaveMessage
		if err := json.Unmarshal(msg.Data, &leaveMsg); err != nil {
			log.Printf("Error unmarshaling leave message: %v", err)
			return
		}

		log.Printf("Player left: %s (%s)", leaveMsg.PlayerID, leaveMsg.Reason)
		n.removeRemotePlayer(leaveMsg.PlayerID, false)

	case shared.MessageTypePlayerLeft:
		var playerLeft shared.PlayerLeftMessage
		if err := json.Unmarshal(msg.Data, &playerLeft); err != nil {
			log.Printf("Error unmarshaling player left message: %v", err)
			return
		}

		log.Printf("Player left: %s (%s)", playerLeft.Nickname, playerLeft.Reason)
		n.removeRemotePlayer(playerLeft.PlayerID, true)

	case shared.MessageTypeHeartbeat:
		var hbMsg shared.HeartbeatMessage
		if err := json.Unmarshal(msg.Data, &hbMsg); err != nil {
			log.Printf("Error unmarshaling heartbeat message: %v", err)
			return
		}
		n.lastPong = time.Now()

	case shared.MessageTypeError:
		var errorMsg shared.ErrorMessage
		if err := json.Unmarshal(msg.Data, &errorMsg); err != nil {
			log.Printf("Error unmarshaling error message: %v", err)
			return
		}

		log.Printf("Server error: %s (code: %d)", errorMsg.Message, errorMsg.Code)
		n.container.NetworkError = errorMsg.Message

		if errorMsg.Code >= 400 {
			n.container.GameState = GameStateMenu
		}

	default:
		log.Printf("Unknown message type: %d", msg.Type)
	}
}

func (n *Network) updateRemotePlayers(players []shared.PlayerState) {
	n.container.Mu.Lock()
	defer n.container.Mu.Unlock()

	foundPlayers := make(map[string]bool)

	for _, player := range players {
		if player.PlayerID == n.container.PlayerID {
			continue
		}

		foundPlayers[player.PlayerID] = true

		if existing, exists := n.container.Players[player.PlayerID]; exists {
			existing.Position = player.Position
			existing.Rotation = player.Rotation
			existing.Animation = player.Animation
			existing.Model = player.Model
			existing.LastUpdate = time.Now()
			existing.IsActive = true
		} else {
			newPlayer := player
			newPlayer.LastUpdate = time.Now()
			newPlayer.IsActive = true
			n.container.Players[player.PlayerID] = &newPlayer
			log.Printf("Remote player added: %s", player.Nickname)
		}
	}

	// Ищем и удаляем игроков, которых нет в списке сервера
	for id := range n.container.Players {
		if !foundPlayers[id] {
			log.Printf("Player %s not found in server list, marking for removal", id)
			n.container.Players[id].IsActive = false
		}
	}
}

func (n *Network) removeRemotePlayer(playerID string, notify bool) {
	n.container.Mu.Lock()
	defer n.container.Mu.Unlock()

	if player, exists := n.container.Players[playerID]; exists {
		if notify {
			log.Printf("Server notified player left: %s", player.Nickname)
		} else {
			log.Printf("Player left locally: %s", player.Nickname)
		}

		// Устанавливаем флаг неактивности и удаляем сразу
		player.IsActive = false
		delete(n.container.Players, playerID)
	}
}

func (n *Network) sendCloseMessage() {
	n.mu.Lock()
	n.isClosing = true
	n.mu.Unlock()

	leaveMsg := shared.LeaveMessage{
		PlayerID: n.container.PlayerID,
		Reason:   "Client disconnected",
	}

	data, _ := json.Marshal(leaveMsg)
	message := shared.Message{
		Type: shared.MessageTypeLeave,
		Data: data,
	}

	msgData, _ := json.Marshal(message)
	n.conn.WriteMessage(websocket.TextMessage, msgData)

	n.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (n *Network) closeConnection() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.isConnected {
		n.isConnected = false
		n.isClosing = true

		if n.heartbeatTicker != nil {
			n.heartbeatTicker.Stop()
		}

		if n.inputSendTicker != nil {
			n.inputSendTicker.Stop()
		}

		close(n.sendChan)
		close(n.receiveChan)
		n.conn.Close()

		log.Println("Network connection closed")
	}
}

func (n *Network) Close() {
	n.closeConnection()
}

func (n *Network) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.isConnected
}

func (n *Network) GetLatency() time.Duration {
	return n.latency
}
