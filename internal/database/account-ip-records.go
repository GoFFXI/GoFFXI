package database

import (
	"context"
	"time"
)

type AccountIPRecord struct {
	LoginTime   time.Time `bun:"type:timestamp,notnull,default:current_timestamp,pk"`
	AccountID   uint32    `bun:"type:int unsigned,notnull,pk"`
	CharacterID uint32    `bun:"type:int unsigned,notnull"`
	ClientIP    string    `bun:"notnull"`
}

type AccountIPRecordQueries interface {
	CreateAccountIPRecord(ctx context.Context, record *AccountIPRecord) (AccountIPRecord, error)
}

func (q *queriesImpl) CreateAccountIPRecord(ctx context.Context, record *AccountIPRecord) (AccountIPRecord, error) {
	_, err := q.db.NewInsert().Model(record).Exec(ctx)
	if err != nil {
		return AccountIPRecord{}, err
	}

	return *record, nil
}
