package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	Logger  *slog.Logger
	DB      Pinger
	Auth    Authenticator
	Session SessionConfig
	Farming Farming
	Shop    Shop
	Now     func() time.Time
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(d.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz(d.DB))

	if d.Now == nil {
		d.Now = time.Now
	}

	sessions := &sessionHandlers{auth: d.Auth, cfg: d.Session, logger: d.Logger}
	farms := &farmHandlers{farming: d.Farming, now: d.Now, logger: d.Logger}
	shops := &shopHandlers{shop: d.Shop, logger: d.Logger}

	r.Route("/v1", func(r chi.Router) {
		r.Post("/sessions/guest", sessions.startGuest)

		r.Group(func(r chi.Router) {
			r.Use(requireSession(d.Auth, d.Logger))
			r.Get("/me", sessions.me)
			farms.routes(r)
			shops.routes(r)
		})
	})

	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyz(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
