package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

const (
	ConstraintAccountSessionsAccountIDUnique = "account_sessions_account_id_unique"
)

var ErrAccountSessionNotUnique = errors.New("account session not unique")

type AccountSession struct {
	AccountID   uint32 `bun:"type:int unsigned,unique"`
	CharacterID uint32 `bun:"type:int unsigned,notnull,pk"`
	SessionKey  string `bun:"type:varchar(16),notnull,unique"`
	ClientIP    string `bun:"type:varchar(15),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

func (m *AccountSession) BeforeUpdate(_ context.Context, _ bun.Query) error {
	m.UpdatedAt = time.Now()
	return nil
}

type AccountSessionQueries interface {
	GetAccountSessionBySessionKey(ctx context.Context, sessionKey string) (AccountSession, error)
	CreateAccountSession(ctx context.Context, accountSession *AccountSession) (AccountSession, error)
	DeleteAccountSessions(ctx context.Context, accountID uint32) error
}

func (q *queriesImpl) GetAccountSessionBySessionKey(ctx context.Context, sessionKey string) (AccountSession, error) {
	var accountSession AccountSession

	err := q.db.NewSelect().Model(&accountSession).Where("session_key = ?", sessionKey).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccountSession{}, ErrNotFound
		}

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

func (q *queriesImpl) DeleteAccountSessions(ctx context.Context, accountID uint32) error {
	_, err := q.db.NewDelete().Model((*AccountSession)(nil)).Where("account_id = ?", accountID).Exec(ctx)
	return err
}
