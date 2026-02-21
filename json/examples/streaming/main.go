package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/leeforge/framework/json"
)

type LogEntry struct {
	Level     string `json:"level" default:"info"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source" default:"unknown"`
}

func main() {
	fmt.Println("=== JSON 流式处理示例 ===")

	// 1. 编码器示例
	fmt.Println("1. 使用编码器写入多个对象:")
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	logs := []LogEntry{
		{Level: "error", Message: "Database connection failed", Timestamp: "2024-01-01T10:00:00Z"},
		{Message: "User login successful", Timestamp: "2024-01-01T10:01:00Z"}, // 使用默认 level
		{Level: "warn", Message: "High memory usage detected", Timestamp: "2024-01-01T10:02:00Z"},
	}

	for i, entry := range logs {
		err := encoder.Encode(entry)
		if err != nil {
			log.Fatal("编码失败:", err)
		}
		fmt.Printf("  编码第 %d 个对象完成\n", i+1)
	}

	fmt.Printf("缓冲区内容:\n%s\n", buf.String())

	// 2. 解码器示例
	fmt.Println("2. 使用解码器读取多个对象:")
	decoder := json.NewDecoder(&buf)

	var decodedLogs []LogEntry
	for {
		var entry LogEntry
		err := decoder.Decode(&entry)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Fatal("解码失败:", err)
		}
		decodedLogs = append(decodedLogs, entry)
		fmt.Printf("  解码对象: %+v\n", entry)
	}

	// 3. 批量处理示例
	fmt.Println("\n3. 批量处理日志数据:")
	jsonLines := `{"level":"error","message":"Critical error","timestamp":"2024-01-01T11:00:00Z"}
{"message":"Info message","timestamp":"2024-01-01T11:01:00Z"}
{"level":"debug","message":"Debug info","timestamp":"2024-01-01T11:02:00Z","source":"app.go"}`

	lines := strings.Split(jsonLines, "\n")
	var processedLogs []LogEntry

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry LogEntry
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			log.Printf("第 %d 行解析失败: %v", i+1, err)
			continue
		}

		processedLogs = append(processedLogs, entry)
		fmt.Printf("  处理第 %d 行: level=%s, source=%s\n", i+1, entry.Level, entry.Source)
	}

	// 4. 输出处理结果
	fmt.Println("\n4. 最终处理结果:")
	result, err := json.MarshalIndent(processedLogs, "", "  ")
	if err != nil {
		log.Fatal("最终序列化失败:", err)
	}
	fmt.Printf("%s\n", result)

	// 5. 展示嵌入特性 - SetIndent
	fmt.Println("\n5. 使用 Encoder 的原生方法 SetIndent:")
	var prettyBuf bytes.Buffer
	prettyEncoder := json.NewEncoder(&prettyBuf)

	// 直接调用 jsoniter.Encoder 的方法，无需手动包装！
	prettyEncoder.SetIndent("", "    ") // 4 个空格缩进
	prettyEncoder.SetEscapeHTML(false)  // 不转义 HTML

	sampleLog := LogEntry{
		Level:     "info",
		Message:   "使用 <SetIndent> & <SetEscapeHTML> 方法",
		Timestamp: "2024-01-01T12:00:00Z",
		Source:    "example.go",
	}

	err = prettyEncoder.Encode(sampleLog)
	if err != nil {
		log.Fatal("格式化编码失败:", err)
	}
	fmt.Printf("格式化输出:\n%s", prettyBuf.String())

	// 6. 展示嵌入特性 - Buffered
	fmt.Println("\n6. 使用 Decoder 的原生方法 Buffered:")
	multiJSON := `{"level":"error","message":"First"}
{"level":"warn","message":"Second"}
{"level":"info","message":"Third"}`

	bufferedDecoder := json.NewDecoder(strings.NewReader(multiJSON))

	var firstLog LogEntry
	err = bufferedDecoder.Decode(&firstLog)
	if err != nil {
		log.Fatal("解码第一个对象失败:", err)
	}
	fmt.Printf("   解码第一个对象: %+v\n", firstLog)

	// 使用 Buffered() 方法检查缓冲区剩余内容
	remaining := bufferedDecoder.Buffered()
	remainingBytes, _ := io.ReadAll(remaining)
	fmt.Printf("   缓冲区剩余内容:\n%s\n", string(remainingBytes))
}
