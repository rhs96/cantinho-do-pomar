package pgstore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rhs96/cantinho-do-pomar/services/api/internal/adapters/pgstore"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/app"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/catalog"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/domain/farm"
	"github.com/rhs96/cantinho-do-pomar/services/api/internal/platform/postgres/pgtest"
)

func TestStoreDoRollsBackOnError(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	owner := createPlayer(t, db)
	store := pgstore.NewStore(db, catalog.Default())
	ctx := t.Context()
	boom := errors.New("boom")

	err := store.Do(ctx, func(r app.Repos) error {
		f, _ := farm.New(owner, 2, 2)
		if _, err := r.Farms().Create(ctx, f); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Do() = %v, want boom", err)
	}

	if _, err := pgstore.NewFarms(db, catalog.Default()).Get(ctx, owner); !errors.Is(err, farm.ErrNotFound) {
		t.Errorf("farm survived rollback: Get() = %v", err)
	}
}

func TestStoreDoCommits(t *testing.T) {
	db := pgtest.NewMigratedDB(t)
	owner := createPlayer(t, db)
	store := pgstore.NewStore(db, catalog.Default())
	ctx := context.Background()

	err := store.Do(ctx, func(r app.Repos) error {
		if _, err := r.Players().GetForUpdate(ctx, owner); err != nil {
			return err
		}
		f, _ := farm.New(owner, 2, 2)
		_, err := r.Farms().Create(ctx, f)
		return err
	})
	if err != nil {
		t.Fatalf("Do() = %v", err)
	}

	if _, err := pgstore.NewFarms(db, catalog.Default()).Get(ctx, owner); err != nil {
		t.Errorf("Get() after commit = %v", err)
	}
}
