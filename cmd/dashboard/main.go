// Dashboard service — WebSocket-сервер live-дашборда. Браузер подключается
// по WS, сервис проксирует live-события кликов из gRPC-стрима Analytics.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	analyticsv1 "github.com/1RAFTIK1/linkpulse-contracts/gen/go/analytics/v1"

	"github.com/1RAFTIK1/linkpulse-dashboard/internal/authclient"
	"github.com/1RAFTIK1/linkpulse-dashboard/internal/config"
	"github.com/1RAFTIK1/linkpulse-dashboard/internal/ws"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("сервис завершился с ошибкой", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	// gRPC-клиент к Analytics. insecure — внутренний трафик в docker-сети;
	// mTLS между сервисами — вне скоупа проекта.
	grpcConn, err := grpc.NewClient(cfg.AnalyticsAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	defer func() {
		if err := grpcConn.Close(); err != nil {
			log.Warn("закрытие grpc", "error", err)
		}
	}()

	// Авторизация WS: реальная через Auth service, если задан AUTH_ADDR.
	var validator ws.TokenValidator
	if cfg.AuthAddr != "" {
		authClient, err := authclient.New(cfg.AuthAddr)
		if err != nil {
			return err
		}
		defer func() {
			if err := authClient.Close(); err != nil {
				log.Warn("закрытие auth-клиента", "error", err)
			}
		}()
		validator = authClient
		log.Info("ws-авторизация включена", "auth_addr", cfg.AuthAddr)
	} else {
		log.Warn("AUTH_ADDR пуст — ws-авторизация ЗАГЛУШКА (любой непустой токен)")
	}

	wsServer := ws.NewServer(analyticsv1.NewAnalyticsServiceClient(grpcConn), validator, log)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", wsServer.Handle)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Прод-режим: сервис сам отдаёт собранную SPA (web/dist) — один контейнер
	// на дашборд. В dev статику отдаёт vite со своим прокси.
	if cfg.WebDist != "" {
		mux.Handle("/", http.FileServer(http.Dir(cfg.WebDist)))
		log.Info("отдаём SPA", "dir", cfg.WebDist)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("dashboard запущен", "addr", cfg.HTTPAddr, "analytics", cfg.AnalyticsAddr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("получен сигнал, останавливаемся", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Сначала предупреждаем WS-клиентов (спека §11), потом гасим HTTP.
	wsServer.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Info("сервис остановлен корректно")
	return nil
}
