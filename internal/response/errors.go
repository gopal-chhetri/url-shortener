package response

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation is the Postgres SQLSTATE for a unique constraint violation.
const pgUniqueViolation = "23505"

type AppError struct {
	Message string
}

type DuplicateData struct {
	Model string
}

type InvalidOperation struct {
	Message string
}

type PermissionDeniedError struct {
	Message string
}

type NotFoundError struct {
	Model string
}

// UnauthorizedError means the caller's credentials are missing or invalid.
type UnauthorizedError struct {
	Message string
}

func (e UnauthorizedError) Error() string {
	return e.Message
}

func (e AppError) Error() string {
	return e.Message
}

func (e InvalidOperation) Error() string {
	return e.Message
}

func (e NotFoundError) Error() string {
	return e.Model + " not found"
}

func (e PermissionDeniedError) Error() string {
	return e.Message
}

func (e DuplicateData) Error() string {
	return e.Model + " already exists"
}

func NewAppError(message string) error {
	return AppError{
		Message: message,
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// TranslateDBError maps database errors onto the app's typed errors: a missing
// row becomes NotFoundError and a unique-constraint violation DuplicateData.
func TranslateDBError(err error, model string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return NotFoundError{Model: model}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return DuplicateData{Model: model}
	}
	return err
}
