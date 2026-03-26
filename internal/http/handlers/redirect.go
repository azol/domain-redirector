package handlers

import (
	"fmt"
	"log"
	"net/http"

	"domain-redirector/internal/domain/redirect"
)

// RedirectHandler обрабатывает все входящие запросы и
// перенаправляет пользователя в заранее определенную точку назначения.
type RedirectHandler struct {
	service *redirect.Service
	logger  *log.Logger
}

// NewRedirectHandler создает обработчик редиректов.
func NewRedirectHandler(service *redirect.Service, logger *log.Logger) *RedirectHandler {
	return &RedirectHandler{
		service: service,
		logger:  logger,
	}
}

// ServeHTTP вычисляет URL назначения по host заголовку.
// Если поддомен неизвестен, возвращается 404, чтобы не делать неявных редиректов.
func (h *RedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resolution, ok := h.service.Resolve(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if resolution.Canonical {
		// Заголовок canonical на redirect-ответе является вспомогательным сигналом
		// и потому включается только по явной настройке.
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", resolution.Destination))
	}

	h.logger.Printf(
		"редирект: method=%s host=%s uri=%s destination=%s status=%d canonical=%t remote_addr=%s",
		r.Method,
		r.Host,
		r.RequestURI,
		resolution.Destination,
		resolution.StatusCode,
		resolution.Canonical,
		r.RemoteAddr,
	)

	http.Redirect(w, r, resolution.Destination, resolution.StatusCode)
}
