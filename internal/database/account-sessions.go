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
	AccountSessionKeyLength                  = 20
)

var ErrAccountSessionNotUnique = errors.New("account session not unique")

type AccountSession struct {
	AccountID   uint32 `bun:"type:int unsigned,unique"`
	CharacterID uint32 `bun:"type:int unsigned,notnull,pk"`
	SessionKey  []byte `bun:"type:binary(20),notnull,unique"`
	ClientIP    string `bun:"type:varchar(15),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

func (m *AccountSession) BeforeUpdate(_ context.Context, _ *bun.UpdateQuery) error {
	m.UpdatedAt = time.Now()
	return nil
}

type AccountSessionQueries interface {
	GetAccountSessionBySessionKey(ctx context.Context, sessionKey []byte) (AccountSession, error)
	GetAccountSessionByCharacterID(ctx context.Context, characterID uint32) (AccountSession, error)
	CreateAccountSession(ctx context.Context, accountSession *AccountSession) (AccountSession, error)
	DeleteAccountSessions(ctx context.Context, accountID uint32) error
	UpdateAccountSession(ctx context.Context, accountID, characterID uint32, clientIP string, sessionKey []byte) error
}

func (q *queriesImpl) GetAccountSessionBySessionKey(ctx context.Context, sessionKey []byte) (AccountSession, error) {
	var accountSession AccountSession

	err := q.db.NewSelect().Model(&accountSession).Where("session_key = ?", normalizeSessionKey(sessionKey)).Scan(ctx)
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
	accountSession.SessionKey = normalizeSessionKey(accountSession.SessionKey)

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

func (q *queriesImpl) UpdateAccountSession(ctx context.Context, accountID, characterID uint32, clientIP string, sessionKey []byte) error {
	query := q.db.NewUpdate().Model((*AccountSession)(nil)).
		Set("character_id = ?", characterID).
		Set("client_ip = ?", clientIP).
		Set("updated_at = ?", time.Now()).
		Where("account_id = ?", accountID)

	if len(sessionKey) == AccountSessionKeyLength {
		query = query.Set("session_key = ?", normalizeSessionKey(sessionKey))
	}

	_, err := query.Exec(ctx)

	return err
}

func normalizeSessionKey(key []byte) []byte {
	normalized := make([]byte, AccountSessionKeyLength)
	if len(key) == 0 {
		return normalized
	}

	copy(normalized, key)
	return normalized
}
