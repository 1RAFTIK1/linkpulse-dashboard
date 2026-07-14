package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	analyticsv1 "github.com/1RAFTIK1/linkpulse-contracts/gen/go/analytics/v1"
)

// StatsHandler — REST-мост к Analytics.GetLinkStats: исторические агрегаты
// для дашборда. Live-лента по WS показывает только события текущей сессии,
// а этот эндпоинт даёт накопленную статистику (переживает перезагрузку
// страницы и переключение ссылок).
type StatsHandler struct {
	analytics analyticsv1.AnalyticsServiceClient
	auth      TokenValidator // nil = dev-заглушка, как в WS
	log       *slog.Logger
}

func NewStatsHandler(analytics analyticsv1.AnalyticsServiceClient, auth TokenValidator, log *slog.Logger) *StatsHandler {
	return &StatsHandler{analytics: analytics, auth: auth, log: log}
}

type hourlyBucketJSON struct {
	Hour       time.Time `json:"hour"`
	ClickCount int64     `json:"click_count"`
}

type statsResponse struct {
	LinkID      string             `json:"link_id"` // строкой: int64 vs JS Number
	TotalClicks int64              `json:"total_clicks"`
	Hourly      []hourlyBucketJSON `json:"hourly"`
}

// ServeHTTP — GET /stats/{link_id}: вся накопленная статистика ссылки.
func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.auth != nil {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, `{"error":"требуется Bearer-токен"}`, http.StatusUnauthorized)
			return
		}
		_, valid, err := h.auth.Validate(r.Context(), token)
		if err != nil {
			h.log.ErrorContext(r.Context(), "auth недоступен", "error", err)
			http.Error(w, `{"error":"auth временно недоступен"}`, http.StatusServiceUnavailable)
			return
		}
		if !valid {
			http.Error(w, `{"error":"невалидный токен"}`, http.StatusUnauthorized)
			return
		}
	}

	linkID, err := strconv.ParseInt(r.PathValue("link_id"), 10, 64)
	if err != nil || linkID <= 0 {
		http.Error(w, `{"error":"некорректный link_id"}`, http.StatusBadRequest)
		return
	}

	// Вся история: агрегаты почасовые, объём на ссылку мал.
	resp, err := h.analytics.GetLinkStats(r.Context(), &analyticsv1.GetLinkStatsRequest{
		LinkId: linkID,
		From:   timestamppb.New(time.Unix(0, 0)),
		To:     timestamppb.Now(),
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "get link stats", "link_id", linkID, "error", err)
		http.Error(w, `{"error":"analytics недоступен"}`, http.StatusBadGateway)
		return
	}

	out := statsResponse{
		LinkID:      strconv.FormatInt(resp.GetLinkId(), 10),
		TotalClicks: resp.GetTotalClicks(),
		Hourly:      make([]hourlyBucketJSON, 0, len(resp.GetHourlyBreakdown())),
	}
	for _, b := range resp.GetHourlyBreakdown() {
		out.Hourly = append(out.Hourly, hourlyBucketJSON{
			Hour:       b.GetHour().AsTime(),
			ClickCount: b.GetClickCount(),
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}
