package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolation = "23505"

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == uniqueViolation
}
