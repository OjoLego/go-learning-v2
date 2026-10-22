package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateStruct validates a struct and returns field-level errors.
func ValidateStruct(s any) []FieldError {
	if err := validate.Struct(s); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			out := make([]FieldError, 0, len(errs))
			for _, e := range errs {
				out = append(out, FieldError{
					Field:   strings.ToLower(e.Field()),
					Message: formatError(e),
				})
			}
			return out
		}
		return []FieldError{{Field: "unknown", Message: err.Error()}}
	}
	return nil
}

func formatError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "this field is required"
	case "gt":
		return fmt.Sprintf("must be greater than %s", e.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", e.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", e.Param())
	default:
		return fmt.Sprintf("failed validation: %s", e.Tag())
	}
}
