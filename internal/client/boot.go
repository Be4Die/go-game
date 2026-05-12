package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"game/internal/shared"

	"github.com/gorilla/websocket"
)

// discoverResponse соответствует формату из гайда Orchestrator
type discoverResponse struct {
	Instances []gameInstance `json:"instances"`
}

type gameInstance struct {
	ID               string                 `json:"id"`
	ServerAddress    string                 `json:"server_address"`
	Protocol         string                 `json:"protocol"`
	PlayerCount      uint32                 `json:"player_count"`
	MaxPlayers       uint32                 `json:"max_players"`
	Status           string                 `json:"status"`
	DeveloperPayload map[string]interface{} `json:"developer_payload,omitempty"`
}

// queueStatusResponse — ответ от Orchestrator на запрос статуса очереди
type queueStatusResponse struct {
	Position            int64  `json:"position"`
	EstimatedWaitSeconds int64 `json:"estimated_wait_seconds"`
	ReservedInstanceID  string `json:"reserved_instance_id"`
}

// GameBoot выполняет полный цикл подключения: Discovery -> [Queue] -> WebSocket -> Join
func GameBoot(container *DataContainer, nickname, model string) {
	cfg := container.Config
	if cfg == nil {
		cfg = LoadConfig()
		container.Config = cfg
	}

	orchestratorURL := cfg.OrchestratorURL
	if orchestratorURL == "" {
		orchestratorURL = "http://localhost:8080"
	}
	gameID := cfg.GameID
	if gameID == 0 {
		gameID = 1
	}

	container.GameState = GameStateConnecting
	container.PlayerNickname = nickname
	container.PlayerModel = model
	container.NetworkError = ""
	container.ConnectionStatus = "Discovering server..."

	log.Printf("Discovering game server via orchestrator at %s...", orchestratorURL)

	instance, err := discoverServer(orchestratorURL, gameID)
	if err != nil {
		if cfg.UseServerSideQueue {
			container.ConnectionStatus = "No servers available. Joining queue..."
			log.Println("No servers available, joining queue...")
			instance, err = joinQueueAndWait(orchestratorURL, gameID, container)
			if err != nil {
				log.Printf("Queue failed: %v", err)
				container.NetworkError = "Queue failed: " + err.Error()
				container.ConnectionStatus = ""
				container.GameState = GameStateError
				return
			}
		} else {
			log.Printf("Failed to discover server: %v", err)
			container.NetworkError = "Failed to discover server: " + err.Error()
			container.ConnectionStatus = ""
			container.GameState = GameStateError
			return
		}
	}

	container.ConnectionStatus = fmt.Sprintf("Server found! Connecting to %s...", instance.ServerAddress)
	log.Printf("Connecting to server at %s (instance: %s, players: %d/%d, status: %s)...",
		instance.ServerAddress, instance.ID, instance.PlayerCount, instance.MaxPlayers, instance.Status)

	wsURL := fmt.Sprintf("ws://%s/ws", instance.ServerAddress)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		container.NetworkError = "Failed to connect to server: " + err.Error()
		container.ConnectionStatus = ""
		container.GameState = GameStateError
		return
	}

	container.ConnectionStatus = "Connected! Joining game..."
	container.Network = NewNetwork(conn, container)
	time.Sleep(500 * time.Millisecond)

	if !container.Network.IsConnected() {
		container.NetworkError = "Failed to establish connection"
		container.ConnectionStatus = ""
		container.GameState = GameStateError
		return
	}

	joinMsg := shared.JoinMessage{
		Nickname: nickname,
		Model:    model,
	}

	data, err := json.Marshal(joinMsg)
	if err != nil {
		log.Printf("Failed to marshal join message: %v", err)
		container.NetworkError = "Failed to prepare join message"
		container.ConnectionStatus = ""
		container.GameState = GameStateError
		container.Network.Close()
		return
	}

	message := shared.Message{
		Type: shared.MessageTypeJoin,
		Data: data,
	}

	msgData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		container.NetworkError = "Failed to prepare message"
		container.ConnectionStatus = ""
		container.GameState = GameStateError
		container.Network.Close()
		return
	}

	err = container.Network.Send(msgData)
	if err != nil {
		log.Printf("Failed to send join message: %v", err)
		container.NetworkError = "Failed to send join message"
		container.ConnectionStatus = ""
		container.GameState = GameStateError
		container.Network.Close()
		return
	}

	time.Sleep(1000 * time.Millisecond)

	if container.NetworkError != "" {
		container.ConnectionStatus = ""
		container.GameState = GameStateError
		container.Network.Close()
	} else {
		container.GameState = GameStateRunning
		container.ConnectionStatus = ""
		log.Println("Successfully connected to server")
	}
}

func discoverServer(orchestratorURL string, gameID int64) (*gameInstance, error) {
	url := fmt.Sprintf("%s/api/v1/games/%d/discover", orchestratorURL, gameID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("discovery request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("orchestrator returned %d: %s", resp.StatusCode, string(body))
	}

	var discResp discoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&discResp); err != nil {
		return nil, fmt.Errorf("failed to decode discovery response: %w", err)
	}

	if len(discResp.Instances) == 0 {
		return nil, fmt.Errorf("no instances available for game %d", gameID)
	}

	// Ищем первый подходящий инстанс: running, websocket и не заполненный
	for _, inst := range discResp.Instances {
		if inst.Status == "running" && (inst.Protocol == "websocket" || inst.Protocol == "ws") {
			if inst.PlayerCount < inst.MaxPlayers {
				return &inst, nil
			}
		}
	}

	// Если не нашли идеальный, берем первый running с websocket
	for _, inst := range discResp.Instances {
		if inst.Status == "running" && (inst.Protocol == "websocket" || inst.Protocol == "ws") {
			return &inst, nil
		}
	}

	// Последний fallback — просто первый инстанс
	return &discResp.Instances[0], nil
}

// joinQueueAndWait реализует server-side очередь через API Orchestrator
func joinQueueAndWait(orchestratorURL string, gameID int64, container *DataContainer) (*gameInstance, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// 1. Join queue
	joinURL := fmt.Sprintf("%s/api/v1/games/%d/queue/join", orchestratorURL, gameID)
	joinBody := map[string]interface{}{
		"player_id":   container.PlayerNickname,
		"region":      "default",
		"party_size":  1,
	}
	bodyData, _ := json.Marshal(joinBody)
	resp, err := httpClient.Post(joinURL, "application/json", bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("queue join failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("queue join returned %d", resp.StatusCode)
	}

	// 2. Heartbeat + status poll
	heartbeatURL := fmt.Sprintf("%s/api/v1/games/%d/queue/heartbeat", orchestratorURL, gameID)
	statusURL := fmt.Sprintf("%s/api/v1/games/%d/queue/status", orchestratorURL, gameID)

	heartbeatTicker := time.NewTicker(5 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-heartbeatTicker.C:
			// Отправляем heartbeat
			_, _ = httpClient.Post(heartbeatURL, "application/json", bytes.NewReader([]byte("{}")))

			// Проверяем статус
			resp, err := httpClient.Get(statusURL)
			if err != nil {
				continue
			}

			var status queueStatusResponse
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			if status.ReservedInstanceID != "" && status.ReservedInstanceID != "0" {
				// Место зарезервировано! Получаем инстанс через Discovery
				container.ConnectionStatus = "Server reserved! Connecting..."
				return discoverServer(orchestratorURL, gameID)
			}

			container.ConnectionStatus = fmt.Sprintf("Queue position: %d (wait ~%ds)",
				status.Position, status.EstimatedWaitSeconds)
			log.Printf("Queue status: position=%d wait=%ds", status.Position, status.EstimatedWaitSeconds)
		}
	}
}
