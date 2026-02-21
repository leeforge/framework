package binding

import (
	"net/http"
	"net/url"
	"testing"
)

// BenchmarkQueryBasicTypes 基准测试：基础类型
func BenchmarkQueryBasicTypes(b *testing.B) {
	type QueryParams struct {
		Name    string  `query:"name"`
		Age     int     `query:"age"`
		Height  float64 `query:"height"`
		Active  bool    `query:"active"`
		Page    uint    `query:"page"`
	}

	req := createRequest("name=john&age=25&height=175.5&active=true&page=1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryWithDefaults 基准测试：带默认值
func BenchmarkQueryWithDefaults(b *testing.B) {
	type QueryParams struct {
		Page     int    `query:"page" default:"1"`
		PageSize int    `query:"page_size" default:"10"`
		Sort     string `query:"sort" default:"created_at"`
		Order    string `query:"order" default:"desc"`
		Keyword  string `query:"keyword"`
	}

	req := createRequest("page=2&keyword=test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryArrays 基准测试：数组
func BenchmarkQueryArrays(b *testing.B) {
	type QueryParams struct {
		Tags []string `query:"tags"`
		IDs  []int    `query:"ids"`
	}

	req := createRequest("tags=go&tags=rust&tags=python&ids=1&ids=2&ids=3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryArraysComma 基准测试：逗号分隔数组
func BenchmarkQueryArraysComma(b *testing.B) {
	type QueryParams struct {
		Tags []string `query:"tags"`
		IDs  []int    `query:"ids"`
	}

	req := createRequest("tags=go,rust,python&ids=1,2,3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryNested 基准测试：嵌套结构体
func BenchmarkQueryNested(b *testing.B) {
	type User struct {
		Name string `query:"name"`
		Age  int    `query:"age"`
		City string `query:"city"`
	}

	type QueryParams struct {
		User   User   `query:"user"`
		Active bool   `query:"active"`
		Page   int    `query:"page"`
	}

	req := createRequest("user.name=john&user.age=25&user.city=beijing&active=true&page=1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryPointers 基准测试：指针类型
func BenchmarkQueryPointers(b *testing.B) {
	type QueryParams struct {
		Name     *string  `query:"name"`
		Age      *int     `query:"age"`
		Active   *bool    `query:"active"`
		MinPrice *float64 `query:"min_price"`
	}

	req := createRequest("name=john&age=25&active=true&min_price=9.99")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryValidation 基准测试：带验证
func BenchmarkQueryValidation(b *testing.B) {
	type QueryParams struct {
		Page     int    `query:"page" validate:"required,min=1"`
		PageSize int    `query:"page_size" validate:"required,min=1,max=100"`
		Email    string `query:"email" validate:"omitempty,email"`
		Sort     string `query:"sort" validate:"omitempty,oneof=asc desc"`
	}

	req := createRequest("page=1&page_size=10&email=test@example.com&sort=asc")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryComplex 基准测试：复杂场景
func BenchmarkQueryComplex(b *testing.B) {
	type Filter struct {
		MinPrice *float64 `query:"min_price"`
		MaxPrice *float64 `query:"max_price"`
		Category string   `query:"category"`
	}

	type QueryParams struct {
		Page     int      `query:"page" default:"1" validate:"min=1"`
		PageSize int      `query:"page_size" default:"10" validate:"min=1,max=100"`
		Sort     string   `query:"sort" default:"created_at"`
		Order    string   `query:"order" default:"desc" validate:"oneof=asc desc"`
		Keywords []string `query:"keywords"`
		Tags     []string `query:"tags"`
		Filter   Filter   `query:"filter"`
		Active   *bool    `query:"active"`
	}

	req := createRequest("page=2&page_size=20&keywords=golang&keywords=web&tags=backend,api&filter.min_price=10.5&filter.category=tech&active=true")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryEmpty 基准测试：空查询参数（全部使用默认值）
func BenchmarkQueryEmpty(b *testing.B) {
	type QueryParams struct {
		Page     int    `query:"page" default:"1"`
		PageSize int    `query:"page_size" default:"10"`
		Sort     string `query:"sort" default:"created_at"`
		Order    string `query:"order" default:"desc"`
	}

	req := createRequest("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryManyFields 基准测试：多字段
func BenchmarkQueryManyFields(b *testing.B) {
	type QueryParams struct {
		Field1  string `query:"field1"`
		Field2  string `query:"field2"`
		Field3  string `query:"field3"`
		Field4  int    `query:"field4"`
		Field5  int    `query:"field5"`
		Field6  int    `query:"field6"`
		Field7  bool   `query:"field7"`
		Field8  bool   `query:"field8"`
		Field9  float64 `query:"field9"`
		Field10 float64 `query:"field10"`
		Field11 string `query:"field11"`
		Field12 string `query:"field12"`
		Field13 int    `query:"field13"`
		Field14 int    `query:"field14"`
		Field15 bool   `query:"field15"`
	}

	req := createRequest("field1=val1&field2=val2&field3=val3&field4=1&field5=2&field6=3&field7=true&field8=false&field9=1.5&field10=2.5&field11=val11&field12=val12&field13=13&field14=14&field15=true")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkQueryParserReuse 基准测试：复用解析器
func BenchmarkQueryParserReuse(b *testing.B) {
	type QueryParams struct {
		Name string `query:"name"`
		Age  int    `query:"age"`
		Page int    `query:"page" default:"1"`
	}

	parser := NewQueryParser()
	req := createRequest("name=john&age=25")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = QueryWithParser(req, &params, parser)
	}
}

// BenchmarkQueryCustomUnmarshaler 基准测试：自定义解析器
func BenchmarkQueryCustomUnmarshaler(b *testing.B) {
	type QueryParams struct {
		Name   string     `query:"name"`
		Custom CustomType `query:"custom"`
	}

	req := createRequest("name=john&custom=test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params QueryParams
		_ = Query(req, &params)
	}
}

// BenchmarkParallel 并行基准测试
func BenchmarkQueryParallel(b *testing.B) {
	type QueryParams struct {
		Page     int    `query:"page" default:"1"`
		PageSize int    `query:"page_size" default:"10"`
		Keyword  string `query:"keyword"`
		Sort     string `query:"sort" default:"created_at"`
	}

	req := createRequest("page=2&keyword=test&sort=name")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var params QueryParams
			_ = Query(req, &params)
		}
	})
}

// Comparison benchmarks

// BenchmarkURLParse 基准测试：标准库 URL 解析（对比基准）
func BenchmarkURLParse(b *testing.B) {
	rawQuery := "name=john&age=25&height=175.5&active=true&page=1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values, _ := url.ParseQuery(rawQuery)
		_ = values.Get("name")
		_ = values.Get("age")
		_ = values.Get("height")
		_ = values.Get("active")
		_ = values.Get("page")
	}
}

// BenchmarkManualParse 基准测试：手动解析（对比基准）
func BenchmarkManualParse(b *testing.B) {
	req := &http.Request{
		URL: &url.URL{},
	}
	values, _ := url.ParseQuery("name=john&age=25&height=175.5&active=true&page=1")
	req.URL.RawQuery = values.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		type QueryParams struct {
			Name   string
			Age    int
			Height float64
			Active bool
			Page   uint
		}
		var params QueryParams
		queryValues := req.URL.Query()
		params.Name = queryValues.Get("name")
		// 手动类型转换会更慢，这里只是演示
		_ = params
	}
}
