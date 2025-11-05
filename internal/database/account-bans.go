package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type AccountBan struct {
	AccountID    uint32    `bun:"type:int unsigned,pk"`
	TimeBanned   time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	TimeUnbanned time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	Reason       string    `bun:"type:varchar(512),notnull"`
}

type AccountBanQueries interface {
	GetLastAccountBan(ctx context.Context, accountID uint32) (AccountBan, error)
	IsAccountBanned(ctx context.Context, accountID uint32) (bool, error)
	CreateAccountBan(ctx context.Context, accountBan *AccountBan) (AccountBan, error)
}

func (q *queriesImpl) GetLastAccountBan(ctx context.Context, accountID uint32) (AccountBan, error) {
	var accountBan AccountBan

	err := q.db.NewSelect().Model(&accountBan).
		Where("account_id = ?", accountID).
		Order("time_banned DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccountBan{}, ErrNotFound
		}

		return AccountBan{}, err
	}

	return accountBan, nil
}

func (q *queriesImpl) IsAccountBanned(ctx context.Context, accountID uint32) (bool, error) {
	var count int

	count, err := q.db.NewSelect().Model((*AccountBan)(nil)).
		Where("account_id = ? AND time_unbanned > ?", accountID, time.Now()).
		Count(ctx)
	if err != nil {
		return false, err
	}

	return (count > 0), nil
}

func (q *queriesImpl) CreateAccountBan(ctx context.Context, accountBan *AccountBan) (AccountBan, error) {
	_, err := q.db.NewInsert().Model(accountBan).Exec(ctx)
	if err != nil {
		return AccountBan{}, err
	}

	return *accountBan, nil
}
