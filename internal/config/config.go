package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultPort = 8080

// Config хранит конфигурацию приложения.
type Config struct {
	ListenAddress         string
	Routes                map[string]Route
	EnableCanonicalHeader bool
	RedirectStatus        int
}

// Route описывает одно правило редиректа и возможные per-route переопределения.
type Route struct {
	Destination          string
	RedirectStatus       int
	CanonicalHeader      bool
	CanonicalHeaderIsSet bool
}

// Load читает конфигурацию приложения из переменных окружения.
//
// Обязательная переменная:
// - ROUTES=promo=>/promo,docs.example.com=>https://docs.other.host/landing
//
// Формат ROUTES:
// - разделители записей: запятая, точка с запятой или перевод строки;
// - формат записи: source=>destination|key=value|key=value;
// - source: либо первый label, либо полный host;
// - destination: либо путь, либо полный URL.
func Load() (Config, error) {
	port := getIntEnv("PORT", defaultPort)
	redirectStatus := getRedirectStatusCode("REDIRECT_STATUS_CODE", 307)
	enableCanonicalHeader := getBoolEnv("ENABLE_CANONICAL_HEADER", false)
	routes, err := parseRoutes(os.Getenv("ROUTES"))
	if err != nil {
		return Config{}, err
	}

	applyRouteDefaults(routes, redirectStatus, enableCanonicalHeader)

	return Config{
		ListenAddress:         fmt.Sprintf(":%d", port),
		Routes:                routes,
		EnableCanonicalHeader: enableCanonicalHeader,
		RedirectStatus:        redirectStatus,
	}, nil
}

func getIntEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getRedirectStatusCode(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	if !isSupportedRedirectStatus(parsed) {
		return fallback
	}

	return parsed
}

func parseRoutes(raw string) (map[string]Route, error) {
	normalized := strings.NewReplacer("\r\n", "\n", "\r", "\n", ";", ",", "\n", ",").Replace(raw)
	entries := strings.Split(normalized, ",")

	routes := make(map[string]Route)

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		source, route, err := parseRouteEntry(entry)
		if err != nil {
			return nil, err
		}
		if source == "" {
			return nil, fmt.Errorf("обнаружен пустой source в записи %q", entry)
		}

		source = strings.ToLower(strings.TrimSpace(source))
		if strings.Contains(source, "://") {
			return nil, fmt.Errorf("source %q должен быть host или первым label без схемы", source)
		}

		if _, exists := routes[source]; exists {
			return nil, fmt.Errorf("дублирующийся source %q в ROUTES", source)
		}

		routes[source] = route
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("переменная ROUTES пуста: укажите хотя бы одно соответствие в формате promo=>/promo")
	}

	return routes, nil
}

func parseRouteEntry(entry string) (string, Route, error) {
	source, rest, ok := splitRouteEntry(entry)
	if !ok {
		return "", Route{}, fmt.Errorf("некорректная запись маршрута %q: ожидается формат source=>destination|key=value", entry)
	}

	source = strings.TrimSpace(source)
	rest = strings.TrimSpace(rest)
	if source == "" {
		return "", Route{}, fmt.Errorf("обнаружен пустой source в записи %q", entry)
	}
	if rest == "" {
		return "", Route{}, fmt.Errorf("обнаружен пустой destination в записи %q", entry)
	}

	parts := strings.Split(rest, "|")
	route := Route{
		Destination: normalizeDestination(parts[0]),
	}

	for _, option := range parts[1:] {
		option = strings.TrimSpace(option)
		if !ok {
			continue
		}
		if option == "" {
			continue
		}

		key, value, ok := strings.Cut(option, "=")
		if !ok {
			return "", Route{}, fmt.Errorf("некорректная опция %q в записи %q: ожидается key=value", option, entry)
		}

		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch key {
		case "status", "redirect", "code":
			route.RedirectStatus = parseRedirectStatusValue(value)
			if route.RedirectStatus == 0 {
				return "", Route{}, fmt.Errorf("некорректный status %q в записи %q", value, entry)
			}
		case "canonical":
			parsed, ok := parseBoolValue(value)
			if !ok {
				return "", Route{}, fmt.Errorf("некорректный canonical %q в записи %q", value, entry)
			}
			route.CanonicalHeader = parsed
			route.CanonicalHeaderIsSet = true
		default:
			return "", Route{}, fmt.Errorf("неподдерживаемая опция %q в записи %q", key, entry)
		}
	}

	return source, route, nil
}

func splitRouteEntry(entry string) (string, string, bool) {
	if left, right, ok := strings.Cut(entry, "=>"); ok {
		return left, right, true
	}

	left, right, ok := strings.Cut(entry, "=")
	return left, right, ok
}

func applyRouteDefaults(routes map[string]Route, defaultStatus int, defaultCanonical bool) {
	for source, route := range routes {
		if route.RedirectStatus == 0 {
			route.RedirectStatus = defaultStatus
		}
		if !route.CanonicalHeaderIsSet {
			route.CanonicalHeader = defaultCanonical
		}
		routes[source] = route
	}
}

func parseRedirectStatusValue(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}

	if !isSupportedRedirectStatus(parsed) {
		return 0
	}

	return parsed
}

func isSupportedRedirectStatus(status int) bool {
	switch status {
	case 301, 302, 307, 308:
		return true
	default:
		return false
	}
}

func parseBoolValue(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func normalizeDestination(destination string) string {
	trimmed := strings.TrimSpace(destination)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}

	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	return fmt.Sprintf("/%s", trimmed)
}
