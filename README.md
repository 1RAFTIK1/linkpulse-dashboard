# linkpulse-dashboard

Dashboard service проекта **LinkPulse**: WebSocket-сервер live-дашборда и
React SPA. Браузер держит одно WS-соединение; на каждую подписку сервис
открывает server-streaming gRPC к Analytics и пересылает события кликов.

```
браузер ──WS──► dashboard ──gRPC StreamLiveClicks──► analytics ──Kafka──► ...
```

## WS-протокол (спека §8)

| Направление | Сообщение | Описание |
|---|---|---|
| → | `{"type":"auth","token":"<jwt>"}` | первое сообщение; фаза 4 — заглушка, фаза 5 — ValidateToken |
| ← | `{"type":"auth_ok"}` | авторизация принята |
| → | `{"type":"subscribe","link_id":"123"}` | подписка на live-клики ссылки |
| ← | `{"type":"click","data":{...}}` | событие клика |
| → | `{"type":"unsubscribe","link_id":"123"}` | отписка |
| ← | `{"type":"shutdown"}` | сервер останавливается, клиент переподключается |

Все id — строки: Snowflake int64 превышает Number.MAX_SAFE_INTEGER.

## Ключевые решения

- **coder/websocket** — поддерживаемый преемник nhooyr.io/websocket,
  context-first API. Запись сериализована мьютексом: в соединение пишут
  readLoop и горутины подписок.
- Один gRPC-стрим на пару (соединение, link_id); отписка/обрыв WS отменяет
  context стрима — утечек горутин нет.
- При SIGTERM всем клиентам рассылается `shutdown` и close-фрейм
  `StatusGoingAway` — фронтенд отличает плановый рестарт от сбоя.
- SPA: React 19 + Vite 7 + TypeScript strict, без чартовых библиотек —
  live-график рисуется SVG вручную. В dev SPA отдаёт vite (с прокси /api и
  /ws), в проде — сам сервис (`WEB_DIST=web/dist`).

## Конфигурация (env)

| Переменная | Дефолт | Описание |
|---|---|---|
| `HTTP_ADDR` | `:8082` | адрес WS/HTTP сервера |
| `ANALYTICS_ADDR` | `localhost:50051` | gRPC-адрес Analytics |
| `WEB_DIST` | `""` | путь к собранной SPA; пусто — статика не отдаётся |

## Запуск локально

```bash
# бэкенды: linkpulse-infra (make up), linkpulse-link, linkpulse-analytics
make run          # WS-сервер на :8082
make web-install  # один раз
make web-dev      # SPA на http://localhost:5173
```

## Зависимости и версии

| Компонент | Версия |
|---|---|
| coder/websocket | 1.8.15 |
| google.golang.org/grpc | 1.82.0 |
| React / Vite / TypeScript | 19.2.7 / 7.3.6 / 5.9.3 |

Go 1.26, Node >= 20.19 для сборки SPA.
