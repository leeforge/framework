package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/leeforge/framework/json"
)

type User struct {
	ID      int    `json:"id" default:"1"`
	Name    string `json:"name" default:"Anonymous"`
	Age     int    `json:"age" default:"18"`
	Email   string `json:"email" default:"user@example.com"`
	Status  string `json:"status" default:"active"`
	IsAdmin bool   `json:"is_admin" default:"false"`
}

func main() {
	fmt.Println("=== JSON 包基本使用示例 ===")

	// 1. 创建用户实例（部分字段有值，部分字段使用默认值）
	user := User{
		Name: "Alice",
		Age:  25,
	}

	fmt.Printf("1. 原始用户数据: %+v\n", user)

	// 2. 序列化为 JSON
	data, err := json.Marshal(user)
	if err != nil {
		log.Fatal("序列化失败:", err)
	}
	fmt.Printf("2. 序列化结果: %s\n", data)

	// 3. 使用 MarshalIndent 格式化输出
	indentData, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		log.Fatal("格式化序列化失败:", err)
	}
	fmt.Printf("3. 格式化序列化结果:\n%s\n", indentData)

	// 4. 使用 MarshalToString 输出字符串
	str, err := json.MarshalToString(user)
	if err != nil {
		log.Fatal("字符串序列化失败:", err)
	}
	fmt.Printf("4. 字符串序列化结果: %s\n", str)

	// 5. 反序列化测试
	jsonStr := `{"name":"Bob","age":30}`
	var newUser User
	err = json.Unmarshal([]byte(jsonStr), &newUser)
	if err != nil {
		log.Fatal("反序列化失败:", err)
	}
	fmt.Printf("5. 反序列化结果（注意默认值的应用）: %+v\n", newUser)

	// 6. 空 JSON 对象测试（全部使用默认值）
	emptyJSON := `{}`
	var emptyUser User
	err = json.Unmarshal([]byte(emptyJSON), &emptyUser)
	if err != nil {
		log.Fatal("空对象反序列化失败:", err)
	}
	fmt.Printf("6. 空对象反序列化结果（全部默认值）: %+v\n", emptyUser)

	// 7. 部分字段覆盖测试
	partialJSON := `{"name":"Charlie","is_admin":true}`
	var partialUser User
	err = json.Unmarshal([]byte(partialJSON), &partialUser)
	if err != nil {
		log.Fatal("部分字段反序列化失败:", err)
	}
	fmt.Printf("7. 部分字段反序列化结果: %+v\n", partialUser)

	// 8. 测试嵌入带来的额外功能 - DisallowUnknownFields
	fmt.Println("\n=== 嵌入特性演示 ===")
	fmt.Println("8. 使用 DisallowUnknownFields（jsoniter.Decoder 的原生方法）:")
	unknownFieldJSON := `{"name":"David","age":28,"unknown_field":"test"}`
	decoder := json.NewDecoder(strings.NewReader(unknownFieldJSON))

	// 直接调用 jsoniter.Decoder 的方法，无需手动包装！
	decoder.DisallowUnknownFields()

	var strictUser User
	err = decoder.Decode(&strictUser)
	if err != nil {
		fmt.Printf("   ✓ 检测到未知字段，错误: %v\n", err)
	} else {
		fmt.Printf("   解析成功: %+v\n", strictUser)
	}

	// 9. 使用 UseNumber 保留数字精度
	fmt.Println("\n9. 使用 UseNumber（jsoniter.Decoder 的原生方法）:")
	bigNumberJSON := `{"id":999999999999999999,"name":"Test"}`
	numDecoder := json.NewDecoder(strings.NewReader(bigNumberJSON))
	numDecoder.UseNumber() // 使用 Number 类型而不是 float64

	var result map[string]interface{}
	err = numDecoder.Decode(&result)
	if err != nil {
		log.Fatal("UseNumber 解析失败:", err)
	}
	fmt.Printf("   解析结果: %+v\n", result)
	fmt.Printf("   ID 类型: %T\n", result["id"])
}
