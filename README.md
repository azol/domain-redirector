# domain-redirector

Универсальный Go-сервис для редиректов по host/subdomain.

Сервис принимает входящий `Host`, находит для него правило в `ROUTES` и делает redirect на путь или полный URL из env-конфигурации.

Примеры:

- `promo.example.com` -> `https://example.com/promo`
- `docs.example.com` -> `https://example.com/documentation`
- `docs.eu.example.com` -> `https://external.example.org/eu-docs`

Если destination задан как путь, приложение само берет схему из входящего запроса и строит target host, отбрасывая первый label. Если destination задан как полный URL, редирект выполняется прямо на него. За reverse proxy приоритет отдается `X-Forwarded-Proto`.

## Возможности

- HTTP-сервер на Go с `chi`
- настраиваемый redirect status
- сохранение query string
- healthcheck на `/healthz`
- маршруты через env
- глобальный и per-route `Link: rel="canonical"`
- тестируемая бизнес-логика без привязки к конкретному домену

## Как работает редирект

Для входящего `promo.example.com`:

1. сервис пытается найти точное совпадение по полному host;
2. если не нашел, пробует short-alias по первому label;
3. берет destination из route;
4. если destination это путь, строит URL из схемы запроса и parent host;
5. возвращает redirect с route-specific или глобальным status code.

Если route не найден, сервис отвечает `404`.

## Конфигурация

Обязательные и опциональные переменные:

- `PORT` - порт приложения, по умолчанию `8080`
- `ROUTES` - обязательная таблица соответствий `source=>destination|option=value`
- `REDIRECT_STATUS_CODE` - глобальный код редиректа по умолчанию: `301`, `302`, `307`, `308`
- `ENABLE_CANONICAL_HEADER` - глобальный default для `Link: rel="canonical"`, по умолчанию `false`

Формат `ROUTES`:

```text
source=>destination|status=302|canonical=true
```

Разделители записей:

- запятая
- `;`
- перевод строки

Примеры:

```env
ROUTES=promo=>/promo,docs=>/documentation,help=>/support
```

```env
ROUTES=promo.example.com=>https://landing.example.org/promo|status=302|canonical=true
```

```env
ROUTES=docs.eu.example.com=>https://external.example.org/eu-docs|status=302,docs=>/documentation
```

Рекомендация по формату:

- в `source` лучше использовать host или alias, а не полный URL;
- схема в `source` пользы почти не дает, потому что матчинг идет по `Host`;
- если нужен редирект на другой домен, это лучше выражать через полный URL в `destination`.

Файл-пример с комментариями: [.env.example](/Users/az/git/github.com/azol/wmecte-redirect/.env.example).

## Локальный запуск

Требования:

- Go `1.26+`

Пример:

```bash
export PORT=8080
export REDIRECT_STATUS_CODE=307
export ENABLE_CANONICAL_HEADER=false
export ROUTES='promo=>/promo,docs=>/documentation,docs.eu.example.com=>https://external.example.org/eu-docs|status=302|canonical=true'
go run ./cmd/redirector
```

Проверка:

```bash
curl -i \
  -H 'Host: promo.example.com' \
  -H 'X-Forwarded-Proto: https' \
  'http://localhost:8080/?utm_source=test'
```

Ожидаемый результат:

- статус `307 Temporary Redirect`
- `Location: https://example.com/promo?utm_source=test`

Если `ENABLE_CANONICAL_HEADER=true`, дополнительно будет:

- `Link: <https://example.com/promo?utm_source=test>; rel="canonical"`

Healthcheck:

```bash
curl http://localhost:8080/healthz
```

## Нужен ли canonical на redirect-ответе

Обычно основной канонический сигнал дает сам редирект и каноникал на целевой HTML-странице. Поэтому в проекте canonical-заголовок отключен по умолчанию и включается только явно через `ENABLE_CANONICAL_HEADER=true`.

## Когда отдельное приложение вообще не нужно

Если редиректы простые и статичные, ту же задачу часто удобнее решать средствами веб-сервера или reverse proxy:

1. `nginx`
2. `caddy`
3. `traefik`
4. edge-правила у CDN

Но отдельный сервис полезен, когда нужны:

- единая логика в коде;
- тесты;
- развитие в сторону более сложных правил.
