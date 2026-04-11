package dto

import (
	"fmt"
	"regexp"
	"strings"

	apperrors "github.com/vatsalpatel/hostbox/internal/errors"

	"github.com/go-playground/validator/v10"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// NewValidator creates and configures a validator instance with custom rules.
func NewValidator() *validator.Validate {
	v := validator.New()

	// Register custom "slug" tag
	v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		return slugRegex.MatchString(fl.Field().String())
	})

	return v
}

// ValidateStruct validates a struct and returns []FieldError if invalid.
func ValidateStruct(v *validator.Validate, s interface{}) []apperrors.FieldError {
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	var fieldErrors []apperrors.FieldError
	for _, e := range err.(validator.ValidationErrors) {
		field := strings.ToLower(e.Field())
		fieldErrors = append(fieldErrors, apperrors.FieldError{
			Field:   field,
			Message: formatValidationMessage(e),
		})
	}
	return fieldErrors
}

func formatValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", e.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", e.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", e.Param())
	case "url":
		return "must be a valid URL"
	case "fqdn":
		return "must be a valid domain name"
	case "slug":
		return "must be lowercase alphanumeric with hyphens, 1-63 chars"
	default:
		return fmt.Sprintf("failed %s validation", e.Tag())
	}
}
