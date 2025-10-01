package core

import "fmt"

type UsrsxError struct {
	Message string
	Cause   error
}

func (e *UsrsxError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *UsrsxError) Unwrap() error {
	return e.Cause
}

type ConfigurationError struct {
	*UsrsxError
}

func NewConfigurationError(message string, cause error) *ConfigurationError {
	return &ConfigurationError{
		UsrsxError: &UsrsxError{Message: message, Cause: cause},
	}
}

type NetworkError struct {
	*UsrsxError
}

func NewNetworkError(message string, cause error) *NetworkError {
	return &NetworkError{
		UsrsxError: &UsrsxError{Message: message, Cause: cause},
	}
}

type DataError struct {
	*UsrsxError
}

func NewDataError(message string, cause error) *DataError {
	return &DataError{
		UsrsxError: &UsrsxError{Message: message, Cause: cause},
	}
}

type ValidationError struct {
	*UsrsxError
}

func NewValidationError(message string, cause error) *ValidationError {
	return &ValidationError{
		UsrsxError: &UsrsxError{Message: message, Cause: cause},
	}
}

type SchemaValidationError struct {
	*DataError
}

func NewSchemaValidationError(message string, cause error) *SchemaValidationError {
	return &SchemaValidationError{
		DataError: &DataError{
			UsrsxError: &UsrsxError{Message: message, Cause: cause},
		},
	}
}
