package redirect

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"domain-redirector/internal/config"
)

// Service инкапсулирует правила вычисления конечного URL для редиректа.
type Service struct {
	routes map[string]config.Route
}

// Resolution описывает итоговое решение по конкретному запросу.
type Resolution struct {
	Destination string
	StatusCode  int
	Canonical   bool
}

// NewService создает сервис редиректов.
func NewService(routes map[string]config.Route) *Service {
	copiedRoutes := make(map[string]config.Route, len(routes))
	for source, route := range routes {
		copiedRoutes[strings.ToLower(strings.TrimSpace(source))] = route
	}

	return &Service{
		routes: copiedRoutes,
	}
}

// Resolve определяет конечный URL и политику ответа по входящему HTTP-запросу.
// Если поддомен не найден в map, метод возвращает ok=false.
func (s *Service) Resolve(r *http.Request) (Resolution, bool) {
	host, ok := normalizeHost(r.Host)
	if !ok {
		return Resolution{}, false
	}

	if route, ok := s.routes[host]; ok {
		return s.buildResolution(r, host, route)
	}

	firstLabel, ok := firstLabel(host)
	if !ok {
		return Resolution{}, false
	}

	route, ok := s.resolveRoute(host, firstLabel)
	if !ok {
		return Resolution{}, false
	}

	return s.buildResolution(r, host, route)
}

func (s *Service) buildResolution(r *http.Request, host string, route config.Route) (Resolution, bool) {
	if isAbsoluteURL(route.Destination) {
		destination, ok := mergeDestinationURL(route.Destination, r.URL.RawQuery)
		if !ok {
			return Resolution{}, false
		}

		return Resolution{
			Destination: destination,
			StatusCode:  route.RedirectStatus,
			Canonical:   route.CanonicalHeader,
		}, true
	}

	targetHost, ok := parentHost(host)
	if !ok {
		return Resolution{}, false
	}

	targetURL := url.URL{
		Scheme:   detectRequestScheme(r),
		Host:     targetHost,
		Path:     route.Destination,
		RawQuery: r.URL.RawQuery,
	}

	return Resolution{
		Destination: targetURL.String(),
		StatusCode:  route.RedirectStatus,
		Canonical:   route.CanonicalHeader,
	}, true
}

func (s *Service) resolveRoute(host, firstLabel string) (config.Route, bool) {
	if route, ok := s.routes[host]; ok {
		return route, true
	}

	if route, ok := s.routes[firstLabel]; ok {
		return route, true
	}

	return config.Route{}, false
}

// normalizeHost очищает host от порта, регистра и завершающей точки.
func normalizeHost(hostport string) (string, bool) {
	host := strings.TrimSpace(hostport)
	if host == "" {
		return "", false
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return "", false
	}

	return host, true
}

// firstLabel возвращает первый label host.
// Например:
// - promo.example.com -> promo
// - localhost -> localhost
func firstLabel(host string) (string, bool) {
	parts := strings.Split(host, ".")
	if len(parts) == 0 {
		return "", false
	}

	firstLabel := strings.TrimSpace(parts[0])
	if firstLabel == "" {
		return "", false
	}

	return firstLabel, true
}

// parentHost отбрасывает первый label хоста.
// Например promo.example.com -> example.com.
func parentHost(host string) (string, bool) {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", false
	}

	base := strings.Join(parts[1:], ".")
	if base == "" {
		return "", false
	}

	return base, true
}

func isAbsoluteURL(destination string) bool {
	return strings.HasPrefix(destination, "http://") || strings.HasPrefix(destination, "https://")
}

func mergeDestinationURL(rawDestination, rawQuery string) (string, bool) {
	destinationURL, err := url.Parse(rawDestination)
	if err != nil {
		return "", false
	}

	if rawQuery == "" {
		return destinationURL.String(), true
	}

	if destinationURL.RawQuery == "" {
		destinationURL.RawQuery = rawQuery
	} else {
		destinationURL.RawQuery = destinationURL.RawQuery + "&" + rawQuery
	}

	return destinationURL.String(), true
}

// detectRequestScheme старается не угадывать схему, а взять ее из фактического запроса.
// За reverse proxy приоритет отдается X-Forwarded-Proto.
func detectRequestScheme(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		first = strings.ToLower(strings.TrimSpace(first))
		if first == "http" || first == "https" {
			return first
		}
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}
