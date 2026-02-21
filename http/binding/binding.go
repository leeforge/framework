package binding

import (
	"fmt"
	"io"
	"net/http"

	validatorV10 "github.com/go-playground/validator/v10"
	"github.com/leeforge/framework/json"
)

type BindInterface interface {
	Name() string
	Bind(*http.Request, any) error
}

const (
	InvalidRequestBodyError = "invalid request body"
)

type BindError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e BindError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: field '%s' %s", e.Type, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

type ValidationErrors []BindError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s", ve[0].Error())
}

// Query 使用默认的查询参数解析器绑定查询参数到结构体
func Query(r *http.Request, v any) error {
	parser := NewQueryParser()
	return QueryWithParser(r, v, parser)
}

func JSON(r *http.Request, v any) error {
	if r.Body == nil {
		return &BindError{
			Type:    "bind_error",
			Message: "request body is empty",
		}
	}

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return &BindError{
			Type:    "bind_error",
			Message: "failed to read request body: " + err.Error(),
		}
	}

	if len(body) == 0 {
		return &BindError{
			Type:    "bind_error",
			Message: "request body is empty",
		}
	}

	if err := json.Unmarshal(body, v); err != nil {
		return &BindError{
			Type:    "json_error",
			Message: "failed to unmarshal JSON: " + err.Error(),
		}
	}

	if err := validator.Struct(v); err != nil {
		if validationErrors, ok := err.(validatorV10.ValidationErrors); ok {
			var bindErrors ValidationErrors
			for _, ve := range validationErrors {
				bindErrors = append(bindErrors, BindError{
					Type:    "validation_error",
					Field:   ve.Field(),
					Message: getValidationMessage(ve),
				})
			}
			return bindErrors
		}
		return &BindError{
			Type:    "validation_error",
			Message: err.Error(),
		}
	}

	return nil
}
