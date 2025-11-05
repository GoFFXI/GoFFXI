package database

import (
	"context"

	"github.com/uptrace/bun"
)

type Queries interface {
	AccountBanQueries
	AccountIPRecordQueries
	AccountQueries
	AccountSessionQueries
	CharacterJobsQueries
	CharacterLooksQueries
	CharacterQueries
	CharacterStatsQueries
}

type Tx interface {
	Queries
}

type DB interface {
	Queries
	RunInTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}

type queriesImpl struct {
	db bun.IDB
}

type DBImpl struct {
	db *bun.DB
	queriesImpl
}

func (d *DBImpl) BunDB() *bun.DB {
	return d.db
}

type TXImpl struct {
	tx *bun.Tx
	queriesImpl
}

func (d *DBImpl) RunInTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error {
	return d.db.RunInTx(ctx, nil, func(ctx context.Context, bunTx bun.Tx) error {
		tx := &TXImpl{
			tx:          &bunTx,
			queriesImpl: queriesImpl{db: bunTx},
		}
		return fn(ctx, tx)
	})
}

func NewDB(db *bun.DB) *DBImpl {
	return &DBImpl{
		db:          db,
		queriesImpl: queriesImpl{db: db},
	}
}
