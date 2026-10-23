package player

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewGuest(t *testing.T) {
	now := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

	p, err := NewGuest("Fazendeira 0042", now)
	if err != nil {
		t.Fatalf("NewGuest() = %v", err)
	}
	if p.ID != "" {
		t.Errorf("ID = %q, want empty before persisting", p.ID)
	}
	if p.Coins != StartingCoins || p.XP != 0 {
		t.Errorf("Coins, XP = %d, %d; want %d, 0", p.Coins, p.XP, StartingCoins)
	}
	if !p.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", p.CreatedAt, now)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"Ana", "Ana", false},
		{"  João   da  Roça ", "João da Roça", false},
		{strings.Repeat("ã", MaxNameLength), strings.Repeat("ã", MaxNameLength), false},
		{"", "", true},
		{"   ", "", true},
		{strings.Repeat("a", MaxNameLength+1), "", true},
	}

	for _, tt := range tests {
		got, err := NormalizeName(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("NormalizeName(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if err != nil && !errors.Is(err, ErrInvalidName) {
			t.Errorf("NormalizeName(%q) err = %v, want ErrInvalidName", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
