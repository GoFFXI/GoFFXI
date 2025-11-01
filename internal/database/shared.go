package database

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const (
	ErrCodeDuplicateEntry       = 1062
	ErrCodeForeignKeyConstraint = 1452
)

func isViolationOfConstraint(err error, constraintName string) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		// MySQL error code 1062 is for duplicate entry (unique constraint violation)
		// MySQL error code 1452 is for foreign key constraint violation
		if mysqlErr.Number == ErrCodeDuplicateEntry || mysqlErr.Number == ErrCodeForeignKeyConstraint {
			// Check if the error message contains the constraint name
			if strings.Contains(mysqlErr.Message, constraintName) {
				return true
			}
		}
	}

	return false
}
