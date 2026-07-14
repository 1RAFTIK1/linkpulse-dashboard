package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"

	analyticsv1 "github.com/1RAFTIK1/linkpulse-contracts/gen/go/analytics/v1"
)

// writeTimeout — потолок на запись одного сообщения в WS.
const writeTimeout = 5 * time.Second

// TokenValidator проверяет токен из первого WS-сообщения; nil — dev-заглушка
// (принимается любой непустой токен). Реализация — gRPC ValidateToken к Auth.
type TokenValidator interface {
	Validate(ctx context.Context, token string) (userID string, valid bool, err error)
}

// Server принимает WS-соединения и мостит их к gRPC-стримам Analytics.
type Server struct {
	analytics analyticsv1.AnalyticsServiceClient
	auth      TokenValidator // nil = заглушка
	log       *slog.Logger

	mu    sync.Mutex
	conns map[*conn]struct{} // активные соединения — для shutdown-рассылки
}

func NewServer(analytics analyticsv1.AnalyticsServiceClient, auth TokenValidator, log *slog.Logger) *Server {
	return &Server{analytics: analytics, auth: auth, log: log, conns: make(map[*conn]struct{})}
}

// conn — одно WS-соединение: авторизация, подписки, сериализация записи.
type conn struct {
	sock    *websocket.Conn
	writeMu sync.Mutex // websocket.Conn не допускает конкурентный Write

	mu     sync.Mutex
	subs   map[int64]context.CancelFunc // link_id → отмена gRPC-стрима
	authed bool
}

// Handle — HTTP-handler эндпоинта /ws.
func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	sock, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Dev-режим: принимаем любой Origin (SPA живёт на другом порту).
		// В проде здесь белый список доменов дашборда.
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.log.Warn("ws accept", "error", err)
		return
	}

	c := &conn{sock: sock, subs: make(map[int64]context.CancelFunc)}
	s.register(c)
	defer s.unregister(c)

	s.log.Info("ws подключение открыто", "remote", r.RemoteAddr)
	defer s.log.Info("ws подключение закрыто", "remote", r.RemoteAddr)

	// ctx соединения: отмена (обрыв, shutdown) валит и все gRPC-подписки.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer c.unsubscribeAll()

	s.readLoop(ctx, c)
	_ = sock.CloseNow()
}

func (s *Server) readLoop(ctx context.Context, c *conn) {
	for {
		var msg clientMessage
		if err := readJSON(ctx, c.sock, &msg); err != nil {
			return // обрыв или закрытие — выходим молча
		}

		switch msg.Type {
		case "auth":
			if msg.Token == "" {
				c.send(ctx, s.log, serverMessage{Type: "error", Error: "пустой токен"})
				continue
			}
			// Реальная проверка через Auth service (спека §8, шаг 2);
			// без AUTH_ADDR (s.auth == nil) — dev-заглушка.
			if s.auth != nil {
				vctx, cancel := context.WithTimeout(ctx, 3*time.Second)
				userID, valid, err := s.auth.Validate(vctx, msg.Token)
				cancel()
				if err != nil {
					s.log.Error("validate token", "error", err)
					c.send(ctx, s.log, serverMessage{Type: "error", Error: "auth временно недоступен"})
					continue
				}
				if !valid {
					c.send(ctx, s.log, serverMessage{Type: "error", Error: "невалидный токен"})
					continue
				}
				s.log.Info("ws авторизован", "user_id", userID)
			}
			c.mu.Lock()
			c.authed = true
			c.mu.Unlock()
			c.send(ctx, s.log, serverMessage{Type: "auth_ok"})

		case "subscribe":
			if !c.isAuthed() {
				c.send(ctx, s.log, serverMessage{Type: "error", Error: "сначала auth"})
				continue
			}
			linkID, err := strconv.ParseInt(msg.LinkID, 10, 64)
			if err != nil || linkID <= 0 {
				c.send(ctx, s.log, serverMessage{Type: "error", Error: "некорректный link_id"})
				continue
			}
			s.subscribe(ctx, c, linkID)

		case "unsubscribe":
			linkID, err := strconv.ParseInt(msg.LinkID, 10, 64)
			if err != nil {
				continue
			}
			c.unsubscribe(linkID)

		default:
			c.send(ctx, s.log, serverMessage{Type: "error", Error: "неизвестный тип: " + msg.Type})
		}
	}
}

// subscribe открывает gRPC-стрим к Analytics и пересылает события в WS.
// Один стрим на (соединение, link_id); повторная подписка — no-op.
func (s *Server) subscribe(ctx context.Context, c *conn, linkID int64) {
	c.mu.Lock()
	if _, exists := c.subs[linkID]; exists {
		c.mu.Unlock()
		return
	}
	subCtx, cancel := context.WithCancel(ctx)
	c.subs[linkID] = cancel
	c.mu.Unlock()

	go func() {
		defer c.unsubscribe(linkID)

		stream, err := s.analytics.StreamLiveClicks(subCtx,
			&analyticsv1.StreamLiveClicksRequest{LinkId: linkID})
		if err != nil {
			s.log.Error("открытие gRPC-стрима", "link_id", linkID, "error", err)
			c.send(subCtx, s.log, serverMessage{Type: "error", LinkID: strconv.FormatInt(linkID, 10), Error: "стрим недоступен"})
			return
		}
		s.log.Info("подписка открыта", "link_id", linkID)

		for {
			ev, err := stream.Recv()
			if err != nil {
				// Отмена (unsubscribe/обрыв WS) или падение Analytics.
				if subCtx.Err() == nil && !errors.Is(err, context.Canceled) {
					s.log.Warn("gRPC-стрим прерван", "link_id", linkID, "error", err)
				}
				return
			}
			c.send(subCtx, s.log, serverMessage{Type: "click", Data: toClickData(ev)})
		}
	}()
}

func (c *conn) unsubscribe(linkID int64) {
	c.mu.Lock()
	if cancel, ok := c.subs[linkID]; ok {
		cancel()
		delete(c.subs, linkID)
	}
	c.mu.Unlock()
}

func (c *conn) unsubscribeAll() {
	c.mu.Lock()
	for id, cancel := range c.subs {
		cancel()
		delete(c.subs, id)
	}
	c.mu.Unlock()
}

func (c *conn) isAuthed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.authed
}

// send сериализует запись (websocket.Conn не потокобезопасен на запись:
// в соединение пишут и readLoop, и горутины подписок).
func (c *conn) send(ctx context.Context, log *slog.Logger, msg serverMessage) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := writeJSON(wctx, c.sock, msg); err != nil {
		log.Debug("ws write", "error", err)
	}
}

// Shutdown рассылает {"type":"shutdown"} всем клиентам и закрывает соединения —
// фронтенд отличает плановую остановку от обрыва и переподключается сам
// (спека §11, специфика Dashboard).
func (s *Server) Shutdown(ctx context.Context) {
	s.mu.Lock()
	conns := make([]*conn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	s.log.Info("рассылаем shutdown клиентам", "connections", len(conns))
	for _, c := range conns {
		c.send(ctx, s.log, serverMessage{Type: "shutdown"})
		_ = c.sock.Close(websocket.StatusGoingAway, "server shutting down")
	}
}

// ActiveConnections — gauge для метрик (фаза 6).
func (s *Server) ActiveConnections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

func (s *Server) register(c *conn) {
	s.mu.Lock()
	s.conns[c] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) unregister(c *conn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
}

// readJSON/writeJSON — маленькие обёртки над Reader/Writer вместо wsjson:
// меньше зависимости от вспомогательного пакета, явный контроль таймаутов.
func readJSON(ctx context.Context, sock *websocket.Conn, v any) error {
	_, data, err := sock.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(ctx context.Context, sock *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return sock.Write(ctx, websocket.MessageText, data)
}
