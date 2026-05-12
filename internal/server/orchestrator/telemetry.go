package orchestrator

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// TelemetryPayload соответствует формату из гайда
type TelemetryPayload struct {
	InstanceID  int64  `json:"instance_id"`
	PlayerCount uint32 `json:"player_count"`
	QueueSize   uint32 `json:"queue_size"`
	MaxPlayers  uint32 `json:"max_players"`
}

// Reporter отправляет telemetry на Game Server Node
type Reporter struct {
	reportURL  string
	instanceID int64
	client     *http.Client
	ticker     *time.Ticker
	stopChan   chan bool
	getStats   func() (playerCount, maxPlayers uint32)
}

// NewReporter создает новый telemetry reporter
func NewReporter(getStats func() (playerCount, maxPlayers uint32)) *Reporter {
	reportURL := os.Getenv("GAME_SERVER_NODE_REPORT_URL")
	instanceIDStr := os.Getenv("GAME_SERVER_NODE_INSTANCE_ID")

	// Fallback для локальной разработки
	if reportURL == "" {
		reportURL = "http://localhost:44045/v1/report"
	}

	var instanceID int64 = 1 // fallback
	if instanceIDStr != "" {
		if v, err := strconv.ParseInt(instanceIDStr, 10, 64); err == nil {
			instanceID = v
		}
	}

	return &Reporter{
		reportURL:  reportURL,
		instanceID: instanceID,
		client:     &http.Client{Timeout: 5 * time.Second},
		stopChan:   make(chan bool),
		getStats:   getStats,
	}
}

// Start запускает периодическую отправку telemetry каждые 10 секунд
func (r *Reporter) Start() {
	if r.getStats == nil {
		log.Println("Telemetry reporter: getStats not set, skipping")
		return
	}

	r.ticker = time.NewTicker(10 * time.Second)
	go r.loop()
	log.Printf("Telemetry reporter started: url=%s instance=%d", r.reportURL, r.instanceID)
}

// Stop останавливает reporter
func (r *Reporter) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
		close(r.stopChan)
	}
}

func (r *Reporter) loop() {
	// Отправляем сразу при старте
	r.report()

	for {
		select {
		case <-r.ticker.C:
			r.report()
		case <-r.stopChan:
			return
		}
	}
}

func (r *Reporter) report() {
	playerCount, maxPlayers := r.getStats()

	payload := TelemetryPayload{
		InstanceID:  r.instanceID,
		PlayerCount: playerCount,
		QueueSize:   0, // Очередь на уровне orchestrator, не на сервере
		MaxPlayers:  maxPlayers,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Telemetry marshal error: %v", err)
		return
	}

	resp, err := r.client.Post(r.reportURL, "application/json", bytes.NewReader(data))
	if err != nil {
		// Не логируем как фатальную ошибку — сервер может работать без orchestrator
		log.Printf("Telemetry report failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("Telemetry report returned status: %d", resp.StatusCode)
	}
}
