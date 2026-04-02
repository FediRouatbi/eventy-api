package events

import (
	"errors"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
)

func isUniqueViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}

	return false
}

func isForeignKeyViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1452
	}

	return false
}

func hasConstraint(err error, constraint string) bool {
	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(constraint))
}
