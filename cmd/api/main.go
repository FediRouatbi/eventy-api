package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"eventy-api/internal/app"
	"eventy-api/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file first, before config.Load()
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env vars")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("build app: %v", err)
	}
	defer application.Close()

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           application.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("eventy api listening on :%s", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("start server: %v", err)
	}
}
