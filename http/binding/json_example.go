package binding

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// 这个文件包含 JSON 解码 Option 的使用示例

// ExampleBasicBind 基础绑定示例
func ExampleBasicBind() {
	var r *http.Request
	var user struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Score int64  `json:"score"`
	}

	// 基础用法，不使用任何选项
	binding := jsonBinding{}
	err := binding.Bind(r, &user)
	if err != nil {
		fmt.Println(err)
	}
}

// ExampleUseNumber 使用 UseNumber 选项示例
func ExampleUseNumber() {
	var r *http.Request

	// 当需要处理大整数时，使用 UseNumber 避免精度丢失
	// 例如：JSON 中的 9007199254740992 (2^53) 会在 float64 中丢失精度
	var data struct {
		ID     json.Number `json:"id"` // 使用 json.Number 类型
		Amount json.Number `json:"amount"`
	}

	binding := jsonBinding{}
	err := binding.Bind(r, &data, WithUseNumber())
	if err != nil {
		fmt.Println(err)
		return
	}

	// json.Number 可以转换为不同类型
	id, _ := data.ID.Int64()
	amount, _ := data.Amount.Float64()
	fmt.Printf("ID: %d, Amount: %f\n", id, amount)
}

// ExampleDisallowUnknownFields 不允许未知字段示例
func ExampleDisallowUnknownFields() {
	var r *http.Request
	var user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// 启用严格模式，如果 JSON 中包含 user 结构体没有定义的字段，将返回错误
	// 例如：JSON 中有 {"name": "Alice", "email": "alice@example.com", "age": 30}
	// 会返回错误，因为 age 字段未定义
	binding := jsonBinding{}
	err := binding.Bind(r, &user, WithDisallowUnknownFields())
	if err != nil {
		fmt.Println("JSON 包含未知字段:", err)
		return
	}
}

// ExampleCombinedOptions 组合多个选项示例
func ExampleCombinedOptions() {
	var r *http.Request
	var transaction struct {
		ID     json.Number `json:"id"`
		Amount json.Number `json:"amount"`
		Status string      `json:"status"`
	}

	// 组合使用多个选项
	binding := jsonBinding{}
	err := binding.Bind(r, &transaction,
		WithUseNumber(),             // 使用 Number 类型保持精度
		WithDisallowUnknownFields(), // 严格模式，提高安全性
	)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// ExampleFinancialData 金融数据处理示例
func ExampleFinancialData() {
	var r *http.Request

	// 金融数据需要精确的数字表示
	var payment struct {
		TransactionID json.Number `json:"transaction_id"` // 大整数 ID
		Amount        json.Number `json:"amount"`         // 精确金额
		Currency      string      `json:"currency"`
		UserID        json.Number `json:"userId"`
	}

	// 金融数据处理建议：
	// 1. 使用 UseNumber 保持数字精度
	// 2. 使用 DisallowUnknownFields 提高安全性
	binding := jsonBinding{}
	err := binding.Bind(r, &payment,
		WithUseNumber(),
		WithDisallowUnknownFields(),
	)
	if err != nil {
		fmt.Println("绑定失败:", err)
		return
	}

	// 处理金融数据
	amount, err := payment.Amount.Float64()
	if err != nil {
		fmt.Println("金额解析失败:", err)
		return
	}

	fmt.Printf("交易金额: %.2f %s\n", amount, payment.Currency)
}

// ExampleAPIIntegration API 集成示例
func ExampleAPIIntegration() {
	var r *http.Request

	// 对于第三方 API 响应，可能包含额外字段
	// 不使用 DisallowUnknownFields，保持兼容性
	var apiResponse struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}

	binding := jsonBinding{}
	err := binding.Bind(r, &apiResponse)
	// 不使用 DisallowUnknownFields，允许额外字段
	if err != nil {
		fmt.Println(err)
		return
	}
}

// ExampleStrictMode 严格模式示例
func ExampleStrictMode() {
	var r *http.Request

	// 对于内部 API，使用严格模式确保数据准确性
	var request struct {
		Action string `json:"action"`
		Params string `json:"params"`
	}

	binding := jsonBinding{}
	err := binding.Bind(r, &request, WithDisallowUnknownFields())
	if err != nil {
		// 如果客户端发送了未定义的字段，立即返回错误
		fmt.Println("请求格式错误:", err)
		return
	}
}

// ExampleLargeInteger 大整数处理示例
func ExampleLargeInteger() {
	var r *http.Request

	// JavaScript 的 Number.MAX_SAFE_INTEGER 是 2^53 - 1 (9007199254740991)
	// 超过这个值的整数在 float64 中会丢失精度
	var data struct {
		SnowflakeID json.Number `json:"snowflake_id"` // Twitter Snowflake ID
		Timestamp   json.Number `json:"timestamp"`    // Unix 纳秒时间戳
	}

	binding := jsonBinding{}
	err := binding.Bind(r, &data, WithUseNumber())
	if err != nil {
		fmt.Println(err)
		return
	}

	// 安全地将 json.Number 转换为 int64
	snowflakeID, err := data.SnowflakeID.Int64()
	if err != nil {
		fmt.Println("ID 解析失败:", err)
		return
	}

	fmt.Printf("Snowflake ID: %d\n", snowflakeID)
}
