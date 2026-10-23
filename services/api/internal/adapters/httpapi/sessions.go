package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/auth"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

const sessionCookie = "pomar_session"

type Authenticator interface {
	StartGuest(ctx context.Context) (string, player.Player, error)
	Authenticate(ctx context.Context, token string) (player.Player, error)
}

type SessionConfig struct {
	TTL          time.Duration
	SecureCookie bool
}

type sessionHandlers struct {
	auth   Authenticator
	cfg    SessionConfig
	logger *slog.Logger
}

type playerResponse struct {
	ID          player.ID `json:"id"`
	DisplayName string    `json:"displayName"`
	Coins       int64     `json:"coins"`
	XP          int64     `json:"xp"`
	Level       int       `json:"level"`
	// Total XP at which the current level started and the next one starts,
	// so the client can draw a progress bar.
	LevelXP     int64 `json:"levelXp"`
	NextLevelXP int64 `json:"nextLevelXp"`
}

func toPlayerResponse(p player.Player) playerResponse {
	level := p.Level()
	return playerResponse{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		Coins:       p.Coins,
		XP:          p.XP,
		Level:       level,
		LevelXP:     player.XPForLevel(level),
		NextLevelXP: player.XPForLevel(level + 1),
	}
}

func (h *sessionHandlers) startGuest(w http.ResponseWriter, r *http.Request) {
	token, p, err := h.auth.StartGuest(r.Context())
	if err != nil {
		internalError(w, r, h.logger, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.cfg.TTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"player": toPlayerResponse(p)})
}

func (h *sessionHandlers) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"player": toPlayerResponse(currentPlayer(r.Context()))})
}

type playerKey struct{}

func currentPlayer(ctx context.Context) player.Player {
	p, _ := ctx.Value(playerKey{}).(player.Player)
	return p
}

func requireSession(a Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := a.Authenticate(r.Context(), sessionToken(r))
			if errors.Is(err, auth.ErrUnauthenticated) {
				writeError(w, http.StatusUnauthorized, "unauthenticated", "session missing or expired")
				return
			}
			if err != nil {
				internalError(w, r, logger, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), playerKey{}, p)))
		})
	}
}

// sessionToken accepts a bearer token for non-browser clients and falls back
// to the session cookie.
func sessionToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if token, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(token)
		}
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		return c.Value
	}
	return ""
}
