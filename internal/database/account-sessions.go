package database

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

const (
	ConstraintAccountSessionsAccountIDUnique = "account_sessions_account_id_unique"
)

var ErrAccountSessionNotUnique = errors.New("account session not unique")

type AccountSession struct {
	AccountID     uint32 `bun:"type:int unsigned"`
	CharacterID   uint32 `bun:"type:int unsigned,notnull"`
	SessionKey    string `bun:"type:varchar(32),notnull"`
	ClientAddress uint32 `bun:"type:int unsigned,notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

func (m *AccountSession) BeforeUpdate(_ context.Context, _ bun.Query) error {
	m.UpdatedAt = time.Now()
	return nil
}

type AccountSessionQueries interface {
	GetAccountSessionBySessionHash(ctx context.Context, sessionHash [16]byte) (AccountSession, error)
	CreateAccountSession(ctx context.Context, accountSession *AccountSession) (AccountSession, error)
	DeleteAccountSession(ctx context.Context, id uint) error
}

func (q *queriesImpl) GetAccountSessionBySessionHash(ctx context.Context, sessionHash [16]byte) (AccountSession, error) {
	var accountSession AccountSession

	err := q.db.NewSelect().Model(&accountSession).Where("session_key = ?", sessionHash).Scan(ctx)
	if err != nil {
		return AccountSession{}, err
	}

	return accountSession, nil
}

func (q *queriesImpl) CreateAccountSession(ctx context.Context, accountSession *AccountSession) (AccountSession, error) {
	_, err := q.db.NewInsert().Model(accountSession).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintAccountSessionsAccountIDUnique) {
			return AccountSession{}, ErrAccountSessionNotUnique
		}

		return AccountSession{}, err
	}

	return *accountSession, nil
}

func (q *queriesImpl) DeleteAccountSession(ctx context.Context, id uint) error {
	_, err := q.db.NewDelete().Model((*AccountSession)(nil)).Where("account_id = ?", id).Exec(ctx)
	return err
}
