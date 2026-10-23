package player

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type ID string

const (
	StartingCoins = 200
	MaxNameLength = 32
)

var (
	ErrNotFound    = errors.New("player not found")
	ErrInvalidName = errors.New("display name must have between 1 and 32 characters")
)

type Player struct {
	ID          ID
	DisplayName string
	Coins       int64
	XP          int64
	CreatedAt   time.Time
}

// NewGuest returns a player that has not been persisted yet, so it has no ID.
func NewGuest(name string, now time.Time) (Player, error) {
	name, err := NormalizeName(name)
	if err != nil {
		return Player{}, err
	}
	return Player{
		DisplayName: name,
		Coins:       StartingCoins,
		CreatedAt:   now,
	}, nil
}

func NormalizeName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if n := utf8.RuneCountInString(name); n == 0 || n > MaxNameLength {
		return "", ErrInvalidName
	}
	return name, nil
}

var (
	ErrNotEnoughCoins = errors.New("not enough coins")
	ErrInvalidAmount  = errors.New("amount must be positive")
)

func (p *Player) Spend(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if p.Coins < amount {
		return ErrNotEnoughCoins
	}
	p.Coins -= amount
	return nil
}

func (p *Player) Earn(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	p.Coins += amount
	return nil
}
