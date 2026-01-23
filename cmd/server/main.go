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

	"game/internal/server/network"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP service address")
	flag.Parse()

	mgr := network.NewManager()

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

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("Server stopped gracefully")
}
