package binding

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// TestBasicTypes 测试基础类型
func TestBasicTypes(t *testing.T) {
	type QueryParams struct {
		Name    string  `query:"name"`
		Age     int     `query:"age"`
		Height  float64 `query:"height"`
		Active  bool    `query:"active"`
		Page    uint    `query:"page"`
	}

	tests := []struct {
		name      string
		query     string
		want      QueryParams
		wantError bool
	}{
		{
			name:  "all fields set",
			query: "name=john&age=25&height=175.5&active=true&page=1",
			want: QueryParams{
				Name:   "john",
				Age:    25,
				Height: 175.5,
				Active: true,
				Page:   1,
			},
			wantError: false,
		},
		{
			name:  "partial fields",
			query: "name=alice&age=30",
			want: QueryParams{
				Name:   "alice",
				Age:    30,
				Height: 0,
				Active: false,
				Page:   0,
			},
			wantError: false,
		},
		{
			name:      "invalid integer",
			query:     "age=invalid",
			wantError: true,
		},
		{
			name:      "invalid boolean",
			query:     "active=maybe",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if (err != nil) != tt.wantError {
				t.Errorf("Query() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && params != tt.want {
				t.Errorf("Query() got = %+v, want %+v", params, tt.want)
			}
		})
	}
}

// TestDefaultValues 测试默认值
func TestDefaultValues(t *testing.T) {
	type QueryParams struct {
		Page     int    `query:"page" default:"1"`
		PageSize int    `query:"page_size" default:"10"`
		Sort     string `query:"sort" default:"created_at"`
		Order    string `query:"order" default:"desc"`
	}

	tests := []struct {
		name  string
		query string
		want  QueryParams
	}{
		{
			name:  "no query params - all defaults",
			query: "",
			want: QueryParams{
				Page:     1,
				PageSize: 10,
				Sort:     "created_at",
				Order:    "desc",
			},
		},
		{
			name:  "partial params - mix of values and defaults",
			query: "page=2&sort=name",
			want: QueryParams{
				Page:     2,
				PageSize: 10, // default
				Sort:     "name",
				Order:    "desc", // default
			},
		},
		{
			name:  "all params provided - no defaults used",
			query: "page=3&page_size=20&sort=updated_at&order=asc",
			want: QueryParams{
				Page:     3,
				PageSize: 20,
				Sort:     "updated_at",
				Order:    "asc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			if params != tt.want {
				t.Errorf("Query() got = %+v, want %+v", params, tt.want)
			}
		})
	}
}

// TestArrays 测试数组类型
func TestArrays(t *testing.T) {
	type QueryParams struct {
		Tags   []string `query:"tags"`
		IDs    []int    `query:"ids"`
		Scores []float64 `query:"scores"`
	}

	tests := []struct {
		name  string
		query string
		want  QueryParams
	}{
		{
			name:  "multiple values",
			query: "tags=go&tags=rust&tags=python",
			want: QueryParams{
				Tags:   []string{"go", "rust", "python"},
				IDs:    nil,
				Scores: nil,
			},
		},
		{
			name:  "comma separated",
			query: "tags=go,rust,python",
			want: QueryParams{
				Tags:   []string{"go", "rust", "python"},
				IDs:    nil,
				Scores: nil,
			},
		},
		{
			name:  "integer array",
			query: "ids=1&ids=2&ids=3",
			want: QueryParams{
				Tags:   nil,
				IDs:    []int{1, 2, 3},
				Scores: nil,
			},
		},
		{
			name:  "float array with comma",
			query: "scores=9.5,8.7,10.0",
			want: QueryParams{
				Tags:   nil,
				IDs:    nil,
				Scores: []float64{9.5, 8.7, 10.0},
			},
		},
		{
			name:  "mixed arrays",
			query: "tags=go&tags=rust&ids=1,2,3&scores=9.5&scores=8.7",
			want: QueryParams{
				Tags:   []string{"go", "rust"},
				IDs:    []int{1, 2, 3},
				Scores: []float64{9.5, 8.7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			if !slicesEqual(params.Tags, tt.want.Tags) ||
				!intSlicesEqual(params.IDs, tt.want.IDs) ||
				!floatSlicesEqual(params.Scores, tt.want.Scores) {
				t.Errorf("Query() got = %+v, want %+v", params, tt.want)
			}
		})
	}
}

// TestNestedStructs 测试嵌套结构体
func TestNestedStructs(t *testing.T) {
	type User struct {
		Name string `query:"name"`
		Age  int    `query:"age"`
	}

	type QueryParams struct {
		User User `query:"user"`
	}

	tests := []struct {
		name  string
		query string
		want  QueryParams
	}{
		{
			name:  "nested struct with dot notation",
			query: "user.name=john&user.age=25",
			want: QueryParams{
				User: User{
					Name: "john",
					Age:  25,
				},
			},
		},
		{
			name:  "partial nested fields",
			query: "user.name=alice",
			want: QueryParams{
				User: User{
					Name: "alice",
					Age:  0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			if params != tt.want {
				t.Errorf("Query() got = %+v, want %+v", params, tt.want)
			}
		})
	}
}

// TestPointerTypes 测试指针类型
func TestPointerTypes(t *testing.T) {
	type QueryParams struct {
		Name     *string  `query:"name"`
		Age      *int     `query:"age"`
		Active   *bool    `query:"active"`
		MinPrice *float64 `query:"min_price"`
	}

	tests := []struct {
		name  string
		query string
		check func(t *testing.T, params QueryParams)
	}{
		{
			name:  "all pointer fields set",
			query: "name=john&age=25&active=true&min_price=9.99",
			check: func(t *testing.T, params QueryParams) {
				if params.Name == nil || *params.Name != "john" {
					t.Errorf("Name = %v, want john", params.Name)
				}
				if params.Age == nil || *params.Age != 25 {
					t.Errorf("Age = %v, want 25", params.Age)
				}
				if params.Active == nil || *params.Active != true {
					t.Errorf("Active = %v, want true", params.Active)
				}
				if params.MinPrice == nil || *params.MinPrice != 9.99 {
					t.Errorf("MinPrice = %v, want 9.99", params.MinPrice)
				}
			},
		},
		{
			name:  "no fields set - all nil",
			query: "",
			check: func(t *testing.T, params QueryParams) {
				if params.Name != nil {
					t.Errorf("Name should be nil, got %v", *params.Name)
				}
				if params.Age != nil {
					t.Errorf("Age should be nil, got %v", *params.Age)
				}
				if params.Active != nil {
					t.Errorf("Active should be nil, got %v", *params.Active)
				}
				if params.MinPrice != nil {
					t.Errorf("MinPrice should be nil, got %v", *params.MinPrice)
				}
			},
		},
		{
			name:  "partial fields set",
			query: "name=alice&age=30",
			check: func(t *testing.T, params QueryParams) {
				if params.Name == nil || *params.Name != "alice" {
					t.Errorf("Name = %v, want alice", params.Name)
				}
				if params.Age == nil || *params.Age != 30 {
					t.Errorf("Age = %v, want 30", params.Age)
				}
				if params.Active != nil {
					t.Errorf("Active should be nil, got %v", *params.Active)
				}
				if params.MinPrice != nil {
					t.Errorf("MinPrice should be nil, got %v", *params.MinPrice)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			tt.check(t, params)
		})
	}
}

// TestValidation 测试验证功能
func TestValidation(t *testing.T) {
	type QueryParams struct {
		Page     int    `query:"page" validate:"required,min=1"`
		PageSize int    `query:"page_size" validate:"required,min=1,max=100"`
		Email    string `query:"email" validate:"omitempty,email"`
		Sort     string `query:"sort" validate:"omitempty,oneof=asc desc"`
	}

	tests := []struct {
		name      string
		query     string
		wantError bool
		errorType string
	}{
		{
			name:      "valid params",
			query:     "page=1&page_size=10",
			wantError: false,
		},
		{
			name:      "page too small",
			query:     "page=0&page_size=10",
			wantError: true,
			errorType: "validation_error",
		},
		{
			name:      "page_size too large",
			query:     "page=1&page_size=200",
			wantError: true,
			errorType: "validation_error",
		},
		{
			name:      "invalid email",
			query:     "page=1&page_size=10&email=invalid",
			wantError: true,
			errorType: "validation_error",
		},
		{
			name:      "valid email",
			query:     "page=1&page_size=10&email=test@example.com",
			wantError: false,
		},
		{
			name:      "invalid sort value",
			query:     "page=1&page_size=10&sort=invalid",
			wantError: true,
			errorType: "validation_error",
		},
		{
			name:      "valid sort value",
			query:     "page=1&page_size=10&sort=asc",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if (err != nil) != tt.wantError {
				t.Errorf("Query() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && tt.errorType != "" {
				if bindErr, ok := err.(*BindError); ok {
					if bindErr.Type != tt.errorType {
						t.Errorf("Error type = %s, want %s", bindErr.Type, tt.errorType)
					}
				} else if validationErrs, ok := err.(ValidationErrors); ok {
					if len(validationErrs) > 0 && validationErrs[0].Type != tt.errorType {
						t.Errorf("Error type = %s, want %s", validationErrs[0].Type, tt.errorType)
					}
				}
			}
		})
	}
}

// TestJSONTag 测试使用 json 标签
func TestJSONTag(t *testing.T) {
	type QueryParams struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	req := createRequest("name=john&age=25")
	var params QueryParams

	err := Query(req, &params)
	if err != nil {
		t.Errorf("Query() error = %v", err)
		return
	}

	if params.Name != "john" || params.Age != 25 {
		t.Errorf("Query() got = %+v, want {Name:john Age:25}", params)
	}
}

// TestMixedTags 测试混合使用 query 和 json 标签
func TestMixedTags(t *testing.T) {
	type QueryParams struct {
		Name string `query:"username"`       // 使用 query 标签
		Age  int    `json:"age"`              // 使用 json 标签
		City string                           // 没有标签，使用字段名小写
	}

	req := createRequest("username=john&age=25&city=beijing")
	var params QueryParams

	err := Query(req, &params)
	if err != nil {
		t.Errorf("Query() error = %v", err)
		return
	}

	if params.Name != "john" || params.Age != 25 || params.City != "beijing" {
		t.Errorf("Query() got = %+v, want {Name:john Age:25 City:beijing}", params)
	}
}

// TestIgnoredFields 测试忽略字段
func TestIgnoredFields(t *testing.T) {
	type QueryParams struct {
		Name     string `query:"name"`
		Password string `query:"-"`           // 忽略
		Internal string `json:"-"`            // 忽略
		Age      int    `query:"age"`
	}

	req := createRequest("name=john&password=secret&internal=data&age=25")
	var params QueryParams

	err := Query(req, &params)
	if err != nil {
		t.Errorf("Query() error = %v", err)
		return
	}

	if params.Name != "john" || params.Age != 25 {
		t.Errorf("Query() got = %+v, want {Name:john Age:25}", params)
	}

	if params.Password != "" || params.Internal != "" {
		t.Errorf("Ignored fields should be empty, got Password=%s, Internal=%s", params.Password, params.Internal)
	}
}

// CustomType 自定义类型用于测试 QueryUnmarshaler
type CustomType struct {
	Value string
}

func (ct *CustomType) UnmarshalQuery(s string) error {
	if s == "" {
		return fmt.Errorf("empty value")
	}
	ct.Value = "custom:" + s
	return nil
}

// TestCustomUnmarshaler 测试自定义解析器
func TestCustomUnmarshaler(t *testing.T) {
	type QueryParams struct {
		Name   string     `query:"name"`
		Custom CustomType `query:"custom"`
	}

	req := createRequest("name=john&custom=test")
	var params QueryParams

	err := Query(req, &params)
	if err != nil {
		t.Errorf("Query() error = %v", err)
		return
	}

	if params.Name != "john" {
		t.Errorf("Name = %s, want john", params.Name)
	}

	if params.Custom.Value != "custom:test" {
		t.Errorf("Custom.Value = %s, want custom:test", params.Custom.Value)
	}
}

// TestInvalidInput 测试无效输入
func TestInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantError bool
	}{
		{
			name:      "nil pointer",
			input:     (*struct{})(nil),
			wantError: true,
		},
		{
			name:      "not a pointer",
			input:     struct{}{},
			wantError: true,
		},
		{
			name: "pointer to non-struct",
			input: func() interface{} {
				i := 42
				return &i
			}(),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest("test=value")
			err := Query(req, tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("Query() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestArrayStrategy 测试不同的数组解析策略
func TestArrayStrategy(t *testing.T) {
	type QueryParams struct {
		Tags []string `query:"tags"`
	}

	tests := []struct {
		name     string
		query    string
		strategy ArrayStrategy
		want     []string
	}{
		{
			name:     "multiple strategy - multiple values",
			query:    "tags=go&tags=rust",
			strategy: ArrayStrategyMultiple,
			want:     []string{"go", "rust"},
		},
		{
			name:     "comma strategy - comma separated",
			query:    "tags=go,rust,python",
			strategy: ArrayStrategyComma,
			want:     []string{"go", "rust", "python"},
		},
		{
			name:     "both strategy - prefers multiple",
			query:    "tags=go&tags=rust",
			strategy: ArrayStrategyBoth,
			want:     []string{"go", "rust"},
		},
		{
			name:     "both strategy - uses comma when single value",
			query:    "tags=go,rust,python",
			strategy: ArrayStrategyBoth,
			want:     []string{"go", "rust", "python"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			parser := NewQueryParser()
			parser.SetArrayStrategy(tt.strategy)

			err := QueryWithParser(req, &params, parser)
			if err != nil {
				t.Errorf("QueryWithParser() error = %v", err)
				return
			}

			if !slicesEqual(params.Tags, tt.want) {
				t.Errorf("Tags = %v, want %v", params.Tags, tt.want)
			}
		})
	}
}

// TestDefaultWithValidation 测试默认值与验证的配合
func TestDefaultWithValidation(t *testing.T) {
	type QueryParams struct {
		Page     int    `query:"page" default:"1" validate:"min=1"`
		PageSize int    `query:"page_size" default:"10" validate:"min=1,max=100"`
		Sort     string `query:"sort" default:"created_at"`
	}

	tests := []struct {
		name      string
		query     string
		wantError bool
		check     func(t *testing.T, params QueryParams)
	}{
		{
			name:      "no params - use defaults and pass validation",
			query:     "",
			wantError: false,
			check: func(t *testing.T, params QueryParams) {
				if params.Page != 1 || params.PageSize != 10 || params.Sort != "created_at" {
					t.Errorf("Default values not applied correctly: %+v", params)
				}
			},
		},
		{
			name:      "override default with valid value",
			query:     "page=2&page_size=20",
			wantError: false,
			check: func(t *testing.T, params QueryParams) {
				if params.Page != 2 || params.PageSize != 20 {
					t.Errorf("Values not overridden correctly: %+v", params)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.query)
			var params QueryParams

			err := Query(req, &params)
			if (err != nil) != tt.wantError {
				t.Errorf("Query() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.check != nil {
				tt.check(t, params)
			}
		})
	}
}

// Helper functions

func createRequest(query string) *http.Request {
	req := &http.Request{
		URL: &url.URL{},
	}
	if query != "" {
		values, _ := url.ParseQuery(query)
		req.URL.RawQuery = values.Encode()
	}
	return req
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func floatSlicesEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
