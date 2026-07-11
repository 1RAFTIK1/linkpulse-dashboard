// Package ws — WebSocket-сервер live-дашборда: JSON-протокол поверх WS,
// мост к gRPC-стриму Analytics.
package ws

import (
	"strconv"
	"time"

	eventsv1 "github.com/1RAFTIK1/linkpulse-contracts/gen/go/events/v1"
)

// Входящие сообщения клиента (спека §8).
//
//	{"type":"auth","token":"<jwt>"}
//	{"type":"subscribe","link_id":"123"}
//	{"type":"unsubscribe","link_id":"123"}
//
// link_id передаётся строкой: int64 не влезает в JS Number без потери
// точности (Number.MAX_SAFE_INTEGER = 2^53-1, Snowflake ID больше).
type clientMessage struct {
	Type   string `json:"type"`
	Token  string `json:"token,omitempty"`
	LinkID string `json:"link_id,omitempty"`
}

// Исходящие сообщения сервера.
type serverMessage struct {
	Type   string     `json:"type"` // auth_ok | error | click | shutdown
	Error  string     `json:"error,omitempty"`
	LinkID string     `json:"link_id,omitempty"`
	Data   *clickData `json:"data,omitempty"`
}

// clickData — событие клика в JSON-представлении для браузера.
// ip_hash намеренно не отдаём: фронтенду он не нужен.
type clickData struct {
	EventID     string    `json:"event_id"`
	LinkID      string    `json:"link_id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	ClickedAt   time.Time `json:"clicked_at"`
	Referrer    string    `json:"referrer"`
	Country     string    `json:"country"`
	UserAgent   string    `json:"user_agent"`
}

func toClickData(ev *eventsv1.ClickEvent) *clickData {
	return &clickData{
		EventID:     strconv.FormatInt(ev.GetEventId(), 10),
		LinkID:      strconv.FormatInt(ev.GetLinkId(), 10),
		ShortCode:   ev.GetShortCode(),
		OriginalURL: ev.GetOriginalUrl(),
		ClickedAt:   ev.GetClickedAt().AsTime(),
		Referrer:    ev.GetReferrer(),
		Country:     ev.GetCountry(),
		UserAgent:   ev.GetUserAgent(),
	}
}
