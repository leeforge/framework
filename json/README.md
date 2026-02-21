# JSON 包

高性能的 JSON 序列化/反序列化包，基于 `jsoniter` 库实现，集成默认值设置功能。

## 功能特性

- 🚀 高性能 JSON 处理（基于 jsoniter）
- 🔧 自动默认值设置（基于 defaults 标签）
- 📦 支持流式编码/解码
- 🎯 标准库兼容的 API
- 💫 支持缩进格式化
- 📝 支持字符串输出

## 安装

```bash
go get github.com/JsonLee12138/headless-cms/core/json
```

## 快速开始

### 基本序列化/反序列化

```go
package main

import (
    "fmt"
    "github.com/JsonLee12138/headless-cms/core/json"
)

type User struct {
    ID   int    `json:"id" default:"1"`
    Name string `json:"name" default:"Anonymous"`
    Age  int    `json:"age" default:"18"`
}

func main() {
    // 序列化
    user := User{Name: "Alice", Age: 25}
    data, err := json.Marshal(user)
    if err != nil {
        panic(err)
    }
    fmt.Printf("JSON: %s\n", data)

    // 反序列化
    var newUser User
    err = json.Unmarshal(data, &newUser)
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %+v\n", newUser)
}
```

### 流式处理

```go
package main

import (
    "bytes"
    "fmt"
    "github.com/JsonLee12138/headless-cms/core/json"
)

func main() {
    // 编码器
    var buf bytes.Buffer
    encoder := json.NewEncoder(&buf)
    
    user := User{Name: "Bob", Age: 30}
    err := encoder.Encode(user)
    if err != nil {
        panic(err)
    }
    
    // 解码器
    decoder := json.NewDecoder(&buf)
    var decoded User
    err = decoder.Decode(&decoded)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Decoded: %+v\n", decoded)
}
```

## API 参考

### 序列化函数

#### `Marshal(v any) ([]byte, error)`
将值序列化为 JSON 字节数组。

```go
data, err := json.Marshal(user)
```

#### `MarshalIndent(v any, prefix, indent string) ([]byte, error)`
将值序列化为带缩进的 JSON 字节数组。

```go
data, err := json.MarshalIndent(user, "", "  ")
```

#### `MarshalToString(v any) (string, error)`
将值序列化为 JSON 字符串。

```go
str, err := json.MarshalToString(user)
```

### 反序列化函数

#### `Unmarshal(data []byte, v any) error`
将 JSON 字节数组反序列化为值。

```go
err := json.Unmarshal(data, &user)
```

### 流式处理

#### `NewEncoder(w io.Writer) *Encoder`
创建 JSON 编码器。

```go
encoder := json.NewEncoder(os.Stdout)
```

#### `NewDecoder(r io.Reader) *Decoder`
创建 JSON 解码器。

```go
decoder := json.NewDecoder(os.Stdin)
```

## 默认值功能

本包集成了 `github.com/creasty/defaults` 库，支持通过结构体标签设置默认值。

### 支持的默认值类型

```go
type Config struct {
    // 基本类型
    Name    string  `default:"example"`
    Port    int     `default:"8080"`
    Enabled bool    `default:"true"`
    Rate    float64 `default:"1.5"`
    
    // 切片
    Tags []string `default:"tag1,tag2"`
    
    // 嵌套结构体
    Database struct {
        Host string `default:"localhost"`
        Port int    `default:"5432"`
    }
}
```

### 默认值设置时机

- **序列化时**: 在序列化前设置默认值
- **反序列化时**: 在 JSON 解析后设置默认值（不会覆盖已解析的值）

## 性能特性

基于 `jsoniter` 库，性能显著优于标准库：

- 序列化性能提升 2-3 倍
- 反序列化性能提升 2-3 倍
- 内存分配更少
- 支持流式处理大文件

## 完整示例

### 配置文件处理

```go
package main

import (
    "fmt"
    "os"
    "github.com/JsonLee12138/headless-cms/core/json"
)

type ServerConfig struct {
    Host     string `json:"host" default:"localhost"`
    Port     int    `json:"port" default:"8080"`
    Debug    bool   `json:"debug" default:"false"`
    Database struct {
        Driver   string `json:"driver" default:"mysql"`
        Host     string `json:"host" default:"localhost"`
        Port     int    `json:"port" default:"3306"`
        Database string `json:"database" default:"app"`
    } `json:"database"`
    Redis struct {
        Host string `json:"host" default:"localhost"`
        Port int    `json:"port" default:"6379"`
        DB   int    `json:"db" default:"0"`
    } `json:"redis"`
}

func main() {
    // 从文件读取配置
    data, err := os.ReadFile("config.json")
    if err != nil {
        // 文件不存在时使用默认配置
        data = []byte("{}")
    }
    
    var config ServerConfig
    err = json.Unmarshal(data, &config)
    if err != nil {
        panic(err)
    }
    
    // 输出完整配置（包含默认值）
    output, _ := json.MarshalIndent(config, "", "  ")
    fmt.Printf("Final config:\n%s\n", output)
}
```

### HTTP API 响应

```go
package main

import (
    "net/http"
    "github.com/JsonLee12138/headless-cms/core/json"
)

type APIResponse struct {
    Success bool        `json:"success" default:"true"`
    Code    int         `json:"code" default:"200"`
    Message string      `json:"message" default:"OK"`
    Data    interface{} `json:"data"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    response := APIResponse{
        Data: map[string]string{
            "name": "John",
            "role": "admin",
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    encoder := json.NewEncoder(w)
    encoder.Encode(response)
}
```

## 错误处理

```go
// 序列化错误处理
data, err := json.Marshal(complexStruct)
if err != nil {
    log.Printf("序列化失败: %v", err)
    return
}

// 反序列化错误处理
var result MyStruct
err = json.Unmarshal(jsonData, &result)
if err != nil {
    log.Printf("反序列化失败: %v", err)
    return
}
```

## 最佳实践

### 1. 合理使用默认值

```go
type User struct {
    ID       int       `json:"id"`
    Name     string    `json:"name"`
    Status   string    `json:"status" default:"active"`     // 合理的默认值
    CreateAt time.Time `json:"created_at"`                  // 不设置默认值，让业务逻辑处理
}
```

### 2. 流式处理大数据

```go
func processLargeJSON(r io.Reader, w io.Writer) error {
    decoder := json.NewDecoder(r)
    encoder := json.NewEncoder(w)
    
    for {
        var item Item
        if err := decoder.Decode(&item); err == io.EOF {
            break
        } else if err != nil {
            return err
        }
        
        // 处理 item
        processedItem := processItem(item)
        
        if err := encoder.Encode(processedItem); err != nil {
            return err
        }
    }
    return nil
}
```

### 3. 性能敏感场景

```go
// 预分配切片容量
items := make([]Item, 0, expectedCount)

// 重用 buffer
var buf bytes.Buffer
encoder := json.NewEncoder(&buf)

for _, item := range items {
    buf.Reset() // 重用 buffer
    if err := encoder.Encode(item); err != nil {
        return err
    }
    // 使用 buf.Bytes()
}
```

## 依赖

- `github.com/json-iterator/go` - 高性能 JSON 库
- `github.com/creasty/defaults` - 默认值设置库

## 许可证

本项目遵循主项目许可证。