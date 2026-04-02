package auth

import (
	"errors"

	mysql "github.com/go-sql-driver/mysql"
)

func isUniqueViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}

	return false
}
