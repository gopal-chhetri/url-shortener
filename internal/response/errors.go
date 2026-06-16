package response

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

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
