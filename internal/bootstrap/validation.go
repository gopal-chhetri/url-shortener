package bootstrap

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type validate struct {
	validator *validator.Validate
}

func NewValidator() echo.Validator {
	val := validator.New()

	val.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return validate{validator: val}
}

func (v validate) Validate(i interface{}) error {
	return v.validator.Struct(i)
}
