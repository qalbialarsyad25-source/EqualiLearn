package rest

import (
	"testing"

	"EquiliLearn/internal/model"

	"github.com/go-playground/validator/v10"
)

func TestFormatValidationError_RequiredAndEmail(t *testing.T) {
	validate := validator.New()

	// 1. Test missing required fields
	req := model.UserRegister{
		Name:     "",
		Email:    "invalid-email",
		Password: "123", // too short (< 8)
	}

	err := validate.Struct(req)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	msg, fieldErrors := FormatValidationError(err)
	if msg != "Validation failed" {
		t.Errorf("expected message 'Validation failed', got %q", msg)
	}

	if len(fieldErrors) == 0 {
		t.Fatal("expected fieldErrors map to have entries")
	}

	if nameErr, ok := fieldErrors["name"]; !ok || nameErr != "name is required" {
		t.Errorf("expected 'name is required', got %q", nameErr)
	}

	if emailErr, ok := fieldErrors["email"]; !ok || emailErr != "email must be a valid email address" {
		t.Errorf("expected email error, got %q", emailErr)
	}

	if passErr, ok := fieldErrors["password"]; !ok || passErr != "password must be at least 8 characters long" {
		t.Errorf("expected password min length error, got %q", passErr)
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ConfirmPassword", "confirm_password"},
		{"UserID", "user_id"},
		{"Email", "email"},
		{"UserRegister", "user_register"},
	}

	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		if got != tt.expected {
			t.Errorf("toSnakeCase(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
