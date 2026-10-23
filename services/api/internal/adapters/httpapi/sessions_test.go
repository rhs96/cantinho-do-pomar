package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app/auth"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/player"
)

type fakeAuth struct {
	token  string
	player player.Player
	err    error
}

func (f fakeAuth) StartGuest(context.Context) (string, player.Player, error) {
	return f.token, f.player, f.err
}

func (f fakeAuth) Authenticate(_ context.Context, token string) (player.Player, error) {
	if f.err != nil {
		return player.Player{}, f.err
	}
	if token != f.token {
		return player.Player{}, auth.ErrUnauthenticated
	}
	return f.player, nil
}

var ana = player.Player{ID: "p1", DisplayName: "Ana", Coins: 200}

func newSessionRouter(a Authenticator) http.Handler {
	return NewRouter(Deps{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:      fakePinger{},
		Auth:    a,
		Session: SessionConfig{TTL: time.Hour, SecureCookie: true},
	})
}

func do(t *testing.T, h http.Handler, method, path string, setup func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	if setup != nil {
		setup(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStartGuestSetsCookie(t *testing.T) {
	h := newSessionRouter(fakeAuth{token: "secret", player: ana})

	rec := do(t, h, http.MethodPost, "/v1/sessions/guest", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != sessionCookie || c.Value != "secret" || !c.HttpOnly || !c.Secure || c.MaxAge != 3600 {
		t.Errorf("cookie = %+v", c)
	}

	var body struct {
		Player playerResponse `json:"player"`
		Token  *string        `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Player.ID != ana.ID || body.Player.Coins != ana.Coins {
		t.Errorf("player = %+v", body.Player)
	}
	if body.Token != nil {
		t.Error("token must not be exposed in the response body")
	}
}

func TestStartGuestFailure(t *testing.T) {
	h := newSessionRouter(fakeAuth{err: errors.New("db down")})

	rec := do(t, h, http.MethodPost, "/v1/sessions/guest", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("cookie set on failure")
	}
}

func TestMe(t *testing.T) {
	h := newSessionRouter(fakeAuth{token: "secret", player: ana})

	tests := []struct {
		name  string
		setup func(*http.Request)
		want  int
	}{
		{"no credentials", nil, http.StatusUnauthorized},
		{"cookie", func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "secret"})
		}, http.StatusOK},
		{"bearer", func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer secret")
		}, http.StatusOK},
		{"wrong token", func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "guess"})
		}, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, "/v1/me", tt.setup)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if tt.want != http.StatusOK {
				return
			}
			var body struct {
				Player playerResponse `json:"player"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Player.DisplayName != "Ana" {
				t.Errorf("player = %+v", body.Player)
			}
		})
	}
}

func TestMeAuthBackendError(t *testing.T) {
	h := newSessionRouter(fakeAuth{err: errors.New("db down")})

	rec := do(t, h, http.MethodGet, "/v1/me", func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer secret")
	})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
