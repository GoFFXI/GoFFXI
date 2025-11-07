package database

import (
	"context"
	"database/sql"
	"errors"
)

type AccountTOTP struct {
	AccountID    uint32 `bun:"type:int unsigned,notnull,pk"`
	Secret       string `bun:"type:varchar(32),notnull"`
	RecoveryCode string `bun:"type:varchar(32),notnull"`
	Validated    bool   `bun:"type:boolean,notnull,default:false"`
}

type AccountTOTPQueries interface {
	AccountHasTOTPEnabled(ctx context.Context, accountID uint32) (bool, error)
	GetAccountTOTPByAccountID(ctx context.Context, accountID uint32) (AccountTOTP, error)
	CreateAccountTOTP(ctx context.Context, accountTOTP *AccountTOTP) (AccountTOTP, error)
	UpdateAccountTOTP(ctx context.Context, accountTOTP *AccountTOTP) (AccountTOTP, error)
	DeleteAccountTOTP(ctx context.Context, accountID uint32) error
}

func (q *queriesImpl) AccountHasTOTPEnabled(ctx context.Context, accountID uint32) (bool, error) {
	count, err := q.db.NewSelect().Model((*AccountTOTP)(nil)).Where("account_id = ?", accountID).Count(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return count > 0, nil
}

func (q *queriesImpl) GetAccountTOTPByAccountID(ctx context.Context, accountID uint32) (AccountTOTP, error) {
	var accountTOTP AccountTOTP

	err := q.db.NewSelect().Model(&accountTOTP).Where("account_id = ?", accountID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccountTOTP{}, ErrNotFound
		}

		return AccountTOTP{}, err
	}

	return accountTOTP, nil
}

func (q *queriesImpl) CreateAccountTOTP(ctx context.Context, accountTOTP *AccountTOTP) (AccountTOTP, error) {
	_, err := q.db.NewInsert().Model(accountTOTP).Exec(ctx)
	if err != nil {
		return AccountTOTP{}, err
	}

	return *accountTOTP, nil
}

func (q *queriesImpl) UpdateAccountTOTP(ctx context.Context, accountTOTP *AccountTOTP) (AccountTOTP, error) {
	_, err := q.db.NewUpdate().Model(accountTOTP).Where("account_id = ?", accountTOTP.AccountID).Exec(ctx)
	if err != nil {
		return AccountTOTP{}, err
	}

	return *accountTOTP, nil
}

func (q *queriesImpl) DeleteAccountTOTP(ctx context.Context, accountID uint32) error {
	_, err := q.db.NewDelete().Model((*AccountTOTP)(nil)).Where("account_id = ?", accountID).Exec(ctx)
	return err
}
