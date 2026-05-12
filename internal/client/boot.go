package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"game/internal/shared"

	"github.com/gorilla/websocket"
)

type discoverResponse struct {
	Servers []serverEndpoint `json:"servers"`
}

type serverEndpoint struct {
	InstanceID  string `json:"instance_id"`
	Address     string `json:"address"`
	Port        uint32 `json:"port"`
	Protocol    string `json:"protocol"`
	PlayerCount uint32 `json:"player_count"`
	MaxPlayers  uint32 `json:"max_players"`
}

func GameBoot(container *DataContainer, nickname, model string) {
	const (
		gatewayURL = "http://localhost:8080"
		gameID     = 1
	)

	container.GameState = GameStateConnecting
	container.PlayerNickname = nickname
	container.PlayerModel = model

	log.Println("Discovering game server via gateway...")

	server, err := discoverServer(gatewayURL, gameID)
	if err != nil {
		log.Printf("Failed to discover server: %v", err)
		container.NetworkError = "Failed to discover server: " + err.Error()
		container.GameState = GameStateMenu
		return
	}

	wsURL := fmt.Sprintf("ws://%s:%d/ws", server.Address, server.Port)
	log.Printf("Connecting to server at %s (players: %d/%d)...", wsURL, server.PlayerCount, server.MaxPlayers)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		container.NetworkError = "Failed to connect to server: " + err.Error()
		container.GameState = GameStateMenu
		return
	}

	container.Network = NewNetwork(conn, container)
	time.Sleep(500 * time.Millisecond)

	if !container.Network.IsConnected() {
		container.NetworkError = "Failed to establish connection"
		container.GameState = GameStateMenu
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
		container.GameState = GameStateMenu
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
		container.GameState = GameStateMenu
		container.Network.Close()
		return
	}

	err = container.Network.Send(msgData)
	if err != nil {
		log.Printf("Failed to send join message: %v", err)
		container.NetworkError = "Failed to send join message"
		container.GameState = GameStateMenu
		container.Network.Close()
		return
	}

	time.Sleep(1000 * time.Millisecond)

	if container.NetworkError != "" {
		container.GameState = GameStateMenu
		container.Network.Close()
	} else {
		container.GameState = GameStateRunning
		log.Println("Successfully connected to server")
	}
}

func discoverServer(gatewayURL string, gameID int64) (*serverEndpoint, error) {
	url := fmt.Sprintf("%s/api/v1/games/%d/discover", gatewayURL, gameID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("discovery request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(body))
	}

	var discResp discoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&discResp); err != nil {
		return nil, fmt.Errorf("failed to decode discovery response: %w", err)
	}

	if len(discResp.Servers) == 0 {
		return nil, fmt.Errorf("no servers available for game %d", gameID)
	}

	for _, s := range discResp.Servers {
		if s.Protocol == "PROTOCOL_WEBSOCKET" && s.PlayerCount < s.MaxPlayers {
			return &s, nil
		}
	}

	return &discResp.Servers[0], nil
}
