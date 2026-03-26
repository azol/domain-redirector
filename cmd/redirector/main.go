package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"domain-redirector/internal/app"
	"domain-redirector/internal/config"
)

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("ошибка загрузки конфигурации: %v", err)
	}

	server := app.NewServer(cfg, logger)

	errCh := make(chan error, 1)

	go func() {
		logger.Printf("HTTP сервер запущен на %s", cfg.ListenAddress)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logger.Println("получен сигнал остановки, начинаю graceful shutdown")
	case serveErr := <-errCh:
		logger.Fatalf("сервер завершился с ошибкой: %v", serveErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("ошибка graceful shutdown: %v", err)
	}

	logger.Println("сервер остановлен")
}
