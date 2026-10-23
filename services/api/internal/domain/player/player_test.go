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

func TestSpendAndEarn(t *testing.T) {
	p := Player{Coins: 50}

	if err := p.Spend(30); err != nil || p.Coins != 20 {
		t.Fatalf("Spend(30) = %v, coins %d", err, p.Coins)
	}
	if err := p.Spend(21); !errors.Is(err, ErrNotEnoughCoins) || p.Coins != 20 {
		t.Errorf("Spend(21) = %v, coins %d; want ErrNotEnoughCoins and unchanged", err, p.Coins)
	}
	if err := p.Earn(5); err != nil || p.Coins != 25 {
		t.Errorf("Earn(5) = %v, coins %d", err, p.Coins)
	}

	for _, amount := range []int64{0, -1} {
		if err := p.Spend(amount); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("Spend(%d) = %v, want ErrInvalidAmount", amount, err)
		}
		if err := p.Earn(amount); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("Earn(%d) = %v, want ErrInvalidAmount", amount, err)
		}
	}
}
