package component

import "fmt"

// ValidationError 组件验证错误
type ValidationError struct {
	Component string
	Field     string
	Message   string
	Err       error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s.%s] %s", e.Component, e.Field, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Component, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError 创建验证错误
func NewValidationError(component, field, message string, err error) *ValidationError {
	return &ValidationError{
		Component: component,
		Field:     field,
		Message:   message,
		Err:       err,
	}
}
