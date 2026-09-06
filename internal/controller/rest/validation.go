package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// FormatValidationError converts validator.ValidationErrors or binding errors into clean, user-friendly maps.
func FormatValidationError(err error) (string, map[string]string) {
	if err == nil {
		return "", nil
	}

	var valErrors validator.ValidationErrors
	if errors.As(err, &valErrors) {
		fieldErrors := make(map[string]string)
		for _, fe := range valErrors {
			fieldName := toSnakeCase(fe.Field())
			fieldErrors[fieldName] = formatFieldError(fieldName, fe.Tag(), fe.Param())
		}
		return "Validation failed", fieldErrors
	}

	// Handle JSON unmarshal / syntax errors
	var jsonErr *json.UnmarshalTypeError
	if errors.As(err, &jsonErr) {
		fieldName := toSnakeCase(jsonErr.Field)
		return "Invalid payload format", map[string]string{
			fieldName: fmt.Sprintf("%s must be a valid %s", fieldName, jsonErr.Type.String()),
		}
	}

	// General error string fallback
	cleanMsg := err.Error()
	if strings.Contains(cleanMsg, "EOF") {
		cleanMsg = "Request body cannot be empty"
	}

	return cleanMsg, nil
}

func formatFieldError(fieldName, tag, param string) string {
	friendlyName := strings.ReplaceAll(fieldName, "_", " ")

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", friendlyName)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", friendlyName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", friendlyName, param)
	case "max":
		return fmt.Sprintf("%s cannot exceed %s characters", friendlyName, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", friendlyName, strings.ReplaceAll(param, " ", ", "))
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", friendlyName)
	case "eqfield":
		return fmt.Sprintf("%s must match %s", friendlyName, strings.ReplaceAll(toSnakeCase(param), "_", " "))
	default:
		return fmt.Sprintf("%s is invalid", friendlyName)
	}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// RespondValidationError sends a standardized 400 Bad Request with field-level details.
func RespondValidationError(c *gin.Context, err error) {
	msg, fieldErrors := FormatValidationError(err)
	if len(fieldErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  msg,
			"errors": fieldErrors,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": msg,
	})
}

// RespondError sends a standardized error response with the given status code.
func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}
