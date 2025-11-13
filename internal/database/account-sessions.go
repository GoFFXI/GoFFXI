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

func (m *AccountSession) BeforeUpdate(_ context.Context, _ *bun.UpdateQuery) error {
	m.UpdatedAt = time.Now()
	return nil
}

type AccountSessionQueries interface {
	GetAccountSessionBySessionKey(ctx context.Context, sessionKey string) (AccountSession, error)
	GetAccountSessionByCharacterID(ctx context.Context, characterID uint32) (AccountSession, error)
	CreateAccountSession(ctx context.Context, accountSession *AccountSession) (AccountSession, error)
	DeleteAccountSessions(ctx context.Context, accountID uint32) error
	UpdateAccountSession(ctx context.Context, accountID, characterID uint32, clientIP string) error
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

func (q *queriesImpl) GetAccountSessionByCharacterID(ctx context.Context, characterID uint32) (AccountSession, error) {
	var accountSession AccountSession

	err := q.db.NewSelect().Model(&accountSession).Where("character_id = ?", characterID).Scan(ctx)
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

func (q *queriesImpl) UpdateAccountSession(ctx context.Context, accountID, characterID uint32, clientIP string) error {
	_, err := q.db.NewUpdate().Model((*AccountSession)(nil)).
		Set("character_id = ?, client_ip = ?, updated_at = ?", characterID, clientIP, time.Now()).
		Where("account_id = ?", accountID).
		Exec(ctx)

	return err
}
