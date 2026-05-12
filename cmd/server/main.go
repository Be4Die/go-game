package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"game/internal/server/network"
	"game/internal/server/orchestrator"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP service address")
	maxPlayers := flag.Uint("max-players", 32, "Maximum number of players")
	flag.Parse()

	// Переопределяем max-players из env, если задано
	if envMax := os.Getenv("MAX_PLAYERS"); envMax != "" {
		if v, err := strconv.ParseUint(envMax, 10, 32); err == nil {
			*maxPlayers = uint(v)
		}
	}

	mgr := network.NewManager()
	mgr.SetMaxPlayers(uint32(*maxPlayers))

	// Запускаем telemetry reporter для Game Server Node
	telemetryReporter := orchestrator.NewReporter(func() (uint32, uint32) {
		return mgr.PlayerCount(), mgr.MaxPlayers()
	})
	telemetryReporter.Start()
	defer telemetryReporter.Stop()

	// Run manager in a separate goroutine
	go mgr.Run()

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		network.ServeWs(mgr, w, r)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Встроенный endpoint /v1/report для совместимости с Game Server Node
	http.HandleFunc("/v1/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var report orchestrator.TelemetryPayload
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		log.Printf("Received telemetry report: instance=%s players=%d/%d",
			report.InstanceID, report.PlayerCount, report.MaxPlayers)
		w.WriteHeader(http.StatusOK)
	})

	// Stub Discovery endpoint — позволяет серверу выступать как orchestrator для локального тестирования
	http.HandleFunc("/api/v1/games/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}
		// Ожидаем путь /api/v1/games/{game_id}/discover
		// Или /api/v1/games/{game_id}/discover/
		path := r.URL.Path
		if len(path) < len("/api/v1/games/1/discover") || path[len(path)-len("/discover"):] != "/discover" {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		externalAddr := os.Getenv("EXTERNAL_ADDRESS")
		if externalAddr == "" {
			// Если адрес вида ":8080", превращаем в "localhost:8080"
			if (*addr)[0] == ':' {
				externalAddr = "localhost" + *addr
			} else {
				externalAddr = *addr
			}
		}
		resp := map[string]interface{}{
			"instances": []map[string]interface{}{
				{
					"id":                os.Getenv("GAME_SERVER_NODE_INSTANCE_ID"),
					"server_address":    externalAddr,
					"protocol":          "websocket",
					"player_count":      mgr.PlayerCount(),
					"max_players":       mgr.MaxPlayers(),
					"status":            "running",
					"developer_payload": map[string]string{"map": "default", "mode": "sandbox"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           http.DefaultServeMux,
	}

	// Channel for signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine for graceful shutdown
	go func() {
		sig := <-sigs
		log.Printf("Received signal: %v", sig)

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Stop manager
		mgr.Stop()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on %s", *addr)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", *addr)
	log.Printf("Health check: http://localhost%s/health", *addr)
	log.Printf("Max players: %d", *maxPlayers)
	log.Printf("Instance ID: %s", os.Getenv("GAME_SERVER_NODE_INSTANCE_ID"))

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("Server stopped gracefully")
}
