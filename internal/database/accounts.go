package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

const (
	ConstraintAccountsUsernameUnique = "accounts_username_unique"
)

var ErrAccountNameNotUnique = errors.New("account username not unique")

type Account struct {
	ID       uint32 `bun:"id,pk,autoincrement,type:int unsigned"`
	Username string `bun:"type:varchar(16),notnull,unique"`
	Password string `bun:"type:varchar(64),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

func (m *Account) BeforeUpdate(_ context.Context, _ bun.Query) error {
	m.UpdatedAt = time.Now()
	return nil
}

type AccountQueries interface {
	GetAccountByID(ctx context.Context, id uint) (Account, error)
	GetAccountByUsername(ctx context.Context, username string) (Account, error)
	CreateAccount(ctx context.Context, account *Account) (Account, error)
	UpdateAccount(ctx context.Context, account *Account) (Account, error)
	AccountExists(ctx context.Context, username string) (bool, error)
}

func (q *queriesImpl) GetAccountByID(ctx context.Context, id uint) (Account, error) {
	var account Account

	err := q.db.NewSelect().Model(&account).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrNotFound
		}

		return Account{}, err
	}

	return account, nil
}

func (q *queriesImpl) GetAccountByUsername(ctx context.Context, username string) (Account, error) {
	var account Account

	err := q.db.NewSelect().Model(&account).Where("username = ?", username).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrNotFound
		}

		return Account{}, err
	}

	return account, nil
}

func (q *queriesImpl) CreateAccount(ctx context.Context, account *Account) (Account, error) {
	_, err := q.db.NewInsert().Model(account).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintAccountsUsernameUnique) {
			return Account{}, ErrAccountNameNotUnique
		}

		return Account{}, err
	}

	return *account, nil
}

func (q *queriesImpl) UpdateAccount(ctx context.Context, account *Account) (Account, error) {
	res, err := q.db.NewUpdate().Model(account).Where("id = ?", account.ID).Exec(ctx)
	if err != nil {
		return Account{}, err
	}

	return *account, notFoundErrIfNoRowsAffected(res)
}

func (q *queriesImpl) AccountExists(ctx context.Context, username string) (bool, error) {
	count, err := q.db.NewSelect().Model((*Account)(nil)).Where("username = ?", username).Count(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return count > 0, nil
}
