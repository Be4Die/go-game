package client

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
)

// ClientConfig хранит настройки клиента для подключения к Orchestrator
type ClientConfig struct {
	OrchestratorURL    string `json:"orchestrator_url"`
	GameID             int64  `json:"game_id"`
	UseServerSideQueue bool   `json:"use_server_side_queue"`
}

// LoadConfig загружает конфиг из файла client_config.json или env-переменных
func LoadConfig() *ClientConfig {
	cfg := &ClientConfig{
		OrchestratorURL:    "http://localhost:8080",
		GameID:             1,
		UseServerSideQueue: false,
	}

	// Пытаемся загрузить из файла
	if data, err := os.ReadFile("client_config.json"); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			log.Printf("Failed to parse client_config.json: %v, using defaults", err)
		} else {
			log.Println("Loaded client config from client_config.json")
		}
	} else {
		log.Println("client_config.json not found, using defaults/env")
	}

	// Env-переменные имеют приоритет
	if v := os.Getenv("ORCHESTRATOR_URL"); v != "" {
		cfg.OrchestratorURL = v
	}
	if v := os.Getenv("GAME_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.GameID = id
		}
	}
	if v := os.Getenv("USE_SERVER_SIDE_QUEUE"); v != "" {
		cfg.UseServerSideQueue = v == "1" || v == "true" || v == "yes"
	}

	return cfg
}
