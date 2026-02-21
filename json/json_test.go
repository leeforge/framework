package json

import (
	"bytes"
	stdjson "encoding/json"
	"strings"
	"testing"
)

type testUser struct {
	ID      int    `json:"id,omitempty" default:"1"`
	Name    string `json:"name" default:"Anonymous"`
	Age     int    `json:"age" default:"18"`
	Enabled bool   `json:"enabled" default:"true"`
}

func TestMarshalAppliesDefaults(t *testing.T) {
	user := &testUser{ // only set Name, expect defaults on others
		Name: "Alice",
	}

	data, err := Marshal(user)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	// Marshal should populate defaults on the original struct too
	if user.ID != 1 {
		t.Fatalf("expected default ID=1, got %d", user.ID)
	}
	if user.Age != 18 {
		t.Fatalf("expected default Age=18, got %d", user.Age)
	}
	if !user.Enabled {
		t.Fatalf("expected default Enabled=true, got false")
	}

	// Verify encoded JSON contains populated defaults
	var decoded testUser
	if err := stdjson.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("encoded JSON should be valid, got error: %v", err)
	}
	if decoded != *user {
		t.Fatalf("expected marshaled JSON to match struct with defaults applied, got %+v", decoded)
	}
}

func TestUnmarshalAppliesDefaultsForMissingFields(t *testing.T) {
	input := []byte(`{"name":"Bob"}`)

	var user testUser
	if err := Unmarshal(input, &user); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if user.ID != 1 {
		t.Fatalf("expected default ID=1, got %d", user.ID)
	}
	if user.Age != 18 {
		t.Fatalf("expected default Age=18, got %d", user.Age)
	}
	if !user.Enabled {
		t.Fatalf("expected default Enabled=true, got false")
	}
	if user.Name != "Bob" {
		t.Fatalf("expected Name from JSON to be Bob, got %s", user.Name)
	}
}

func TestUnmarshalPreservesExplicitZeroValues(t *testing.T) {
	input := []byte(`{"age":0,"enabled":false,"id":0,"name":"Charlie"}`)

	var user testUser
	if err := Unmarshal(input, &user); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if user.ID != 0 {
		t.Fatalf("expected explicit ID=0 to be preserved, got %d", user.ID)
	}
	if user.Age != 0 {
		t.Fatalf("expected explicit Age=0 to be preserved, got %d", user.Age)
	}
	if user.Enabled {
		t.Fatalf("expected explicit Enabled=false to be preserved")
	}
}

// TestDecoderDisallowUnknownFields 测试嵌入的 DisallowUnknownFields 方法
func TestDecoderDisallowUnknownFields(t *testing.T) {
	// 测试带有未知字段的 JSON
	input := `{"name":"Alice","age":25,"unknown_field":"should_fail"}`
	decoder := NewDecoder(bytes.NewReader([]byte(input)))

	// 调用嵌入的 jsoniter.Decoder 方法
	decoder.DisallowUnknownFields()

	var user testUser
	err := decoder.Decode(&user)

	// 应该返回错误，因为有未知字段
	if err == nil {
		t.Fatal("expected error for unknown field, but got none")
	}

	t.Logf("correctly rejected unknown field: %v", err)
}

// TestDecoderUseNumber 测试嵌入的 UseNumber 方法
func TestDecoderUseNumber(t *testing.T) {
	type NumberTest struct {
		ID stdjson.Number `json:"id"`
	}

	input := `{"id":999999999999999999}`
	decoder := NewDecoder(bytes.NewReader([]byte(input)))

	// 调用嵌入的 jsoniter.Decoder 方法
	decoder.UseNumber()

	var result NumberTest
	err := decoder.Decode(&result)
	if err != nil {
		t.Fatalf("Decode with UseNumber failed: %v", err)
	}

	// 验证 ID 是 Number 类型
	if result.ID == "" {
		t.Fatal("expected ID to be populated")
	}

	// 验证可以转换为整数
	_, err = result.ID.Int64()
	if err != nil {
		t.Fatalf("expected ID to be convertible to int64, got error: %v", err)
	}

	t.Logf("correctly preserved number as json.Number: %v", result.ID)
}

// TestEncoderSetIndent 测试嵌入的 SetIndent 方法
func TestEncoderSetIndent(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)

	// 调用嵌入的 jsoniter.Encoder 方法
	encoder.SetIndent("", "  ")

	user := testUser{
		ID:      1,
		Name:    "Alice",
		Age:     25,
		Enabled: true,
	}

	err := encoder.Encode(&user)
	if err != nil {
		t.Fatalf("Encode with SetIndent failed: %v", err)
	}

	output := buf.String()

	// 验证输出包含缩进（至少有两个空格的缩进）
	if !strings.Contains(output, "  ") {
		t.Fatalf("expected indented output, got: %s", output)
	}

	t.Logf("correctly formatted with indent:\n%s", output)
}

// TestEncoderSetEscapeHTML 测试嵌入的 SetEscapeHTML 方法
func TestEncoderSetEscapeHTML(t *testing.T) {
	type testHTML struct {
		Content string `json:"content"`
	}

	var buf bytes.Buffer
	encoder := NewEncoder(&buf)

	// 调用嵌入的 jsoniter.Encoder 方法，禁用 HTML 转义
	encoder.SetEscapeHTML(false)

	data := testHTML{
		Content: "<html>&test</html>",
	}

	err := encoder.Encode(&data)
	if err != nil {
		t.Fatalf("Encode with SetEscapeHTML failed: %v", err)
	}

	output := buf.String()

	// 验证 HTML 标签没有被转义
	if !strings.Contains(output, "<html>") {
		t.Fatalf("expected unescaped HTML, got: %s", output)
	}

	t.Logf("correctly preserved HTML without escaping: %s", output)
}
