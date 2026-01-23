package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP service address")
	flag.Parse()

	srv := server.NewServer()

	// Запускаем сервер в отдельной горутине
	go srv.Run()

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.ServeWs(srv, w, r)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Создаем HTTP сервер
	httpServer := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           http.DefaultServeMux,
	}

	// Канал для сигналов
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Горутина для graceful shutdown
	go func() {
		sig := <-sigs
		log.Printf("Received signal: %v", sig)

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Останавливаем сервер
		srv.Stop()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on %s", *addr)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", *addr)
	log.Printf("Health check: http://localhost%s/health", *addr)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("Server stopped gracefully")
}
