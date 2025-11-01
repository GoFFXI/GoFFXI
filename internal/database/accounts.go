package database

import (
	"context"
	"errors"
	"time"
)

const (
	ConstraintAccountsNameUnique = "accounts_name_unique"
)

var ErrAccountNameNotUnique = errors.New("account name not unique")

type Account struct {
	ID       uint   `bun:"id,pk,autoincrement,type:int(10) unsigned"`
	Username string `bun:"type:varchar(16),notnull,unique"`
	Password string `bun:"type:varchar(64),notnull"`

	CreatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"type:timestamp,notnull,default:current_timestamp"`
}

type AccountQueries interface {
	CreateAccount(ctx context.Context, account *Account) (Account, error)
}

func (q *queriesImpl) CreateAccount(ctx context.Context, account *Account) (Account, error) {
	_, err := q.db.NewInsert().Model(&account).Exec(ctx)
	if err != nil {
		if isViolationOfConstraint(err, ConstraintAccountsNameUnique) {
			return Account{}, ErrAccountNameNotUnique
		}

		return Account{}, err
	}

	return *account, err
}
