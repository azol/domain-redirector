package app

import (
	"log"
	"net/http"

	"domain-redirector/internal/config"
	"domain-redirector/internal/domain/redirect"
	"domain-redirector/internal/http/handlers"
	"domain-redirector/internal/http/router"
)

// NewServer собирает все зависимости приложения и возвращает готовый HTTP сервер.
func NewServer(cfg config.Config, logger *log.Logger) *http.Server {
	redirectService := redirect.NewService(cfg.Routes)
	redirectHandler := handlers.NewRedirectHandler(redirectService, logger)
	httpRouter := router.New(redirectHandler, logger)

	return &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           httpRouter,
		ReadHeaderTimeout: 5,
	}
}
