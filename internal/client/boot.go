package client

import (
	"encoding/json"
	"log"
	"time"

	"game/internal/shared"

	"github.com/gorilla/websocket"
)

func GameBoot(container *DataContainer, nickname, model string) {
	container.GameState = GameStateConnecting
	container.PlayerNickname = nickname
	container.PlayerModel = model
	// Players больше не используется, удаляем инициализацию

	// Показываем сообщение о подключении
	log.Println("Connecting to server...")

	// Подключаемся к серверу с таймаутом
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		container.NetworkError = "Failed to connect to server: " + err.Error()
		container.GameState = GameStateMenu
		return
	}

	container.Network = NewNetwork(conn, container)

	// Ждем приветственного сообщения
	time.Sleep(500 * time.Millisecond)

	if !container.Network.IsConnected() {
		container.NetworkError = "Failed to establish connection"
		container.GameState = GameStateMenu
		return
	}

	// Отправляем сообщение о присоединении
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

	// Ждем подтверждения
	time.Sleep(1000 * time.Millisecond)

	if container.NetworkError != "" {
		// Ошибка при подключении
		container.GameState = GameStateMenu
		container.Network.Close()
	} else {
		// Успешное подключение
		container.GameState = GameStateRunning
		log.Println("Successfully connected to server")
	}
}
