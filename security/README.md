# security — 安全工具

提供 AES-GCM 加密/解密、HMAC 签名、密码强度验证与 API Key 生成工具。

## 功能

| 工具 | 说明 |
|---|---|
| `CryptoTool` | AES-256-GCM 加密/解密、HMAC-SHA256 签名 |
| `EncryptionManager` | 面向字符串的加密管理器（自动处理 Key 长度） |
| `PasswordValidator` | 密码强度校验（长度、大小写、数字、特殊字符） |
| `APIKeyGenerator` | 带前缀的 API Key 生成器（32 字节随机） |
| `SHA256Hash` / `HMACHash` | 哈希算法接口实现 |

## 快速开始

### AES 加密

```go
import "github.com/leeforge/framework/security"

// 密钥必须为 16/24/32 字节（AES-128/192/256）
crypto := security.NewCryptoTool("your-32-byte-secret-key-here!!!")

// 加密字符串
encrypted, err := crypto.EncryptString("敏感数据")

// 解密
plaintext, err := crypto.DecryptString(encrypted)
```

### HMAC 签名

```go
crypto := security.NewCryptoTool("secret-key")

// 生成签名（用于 Webhook 签名验证等）
sig := crypto.GenerateSignature(payload)

// 验证签名（常数时间比较，防时序攻击）
valid := crypto.VerifySignature(payload, sig)
```

### 密码强度验证

```go
validator := security.NewPasswordValidator()

valid, err := validator.Validate("MyP@ssw0rd")
// 要求：≥8 位，包含大写、小写、数字、特殊字符

// 快捷函数
if !security.IsSecurePassword(password) {
    return errors.New("密码强度不足")
}
```

### API Key 生成

```go
generator := security.NewAPIKeyGenerator("lf") // 前缀

apiKey, err := generator.Generate()
// 生成格式：lf_<64位十六进制随机字符串>
```

### 加密管理器

```go
// 自动处理密钥长度（不足 32 字节补 0，超出截断）
manager := security.NewEncryptionManager("my-secret")

encrypted, err := manager.EncryptString("data to encrypt")
plaintext, err := manager.DecryptString(encrypted)
```

### 日志脱敏

```go
// 将敏感字符串脱敏后输出到日志
masked := security.MaskString("sk-abc123xyz") // 返回 "sk****yz"
logger.Info("API Key", zap.String("key", masked))
```

## 安全注意事项

- **密码存储**：`HashPassword` 当前使用 HMAC-SHA256（简化实现），生产环境**必须**替换为 `bcrypt` 或 `argon2`
- **AES 密钥**：必须安全生成并存储在环境变量中，不要硬编码在代码里
- **API Key**：创建后只展示一次（`response.Success` 返回），之后仅存储哈希值
- **签名验证**：使用 `hmac.Equal` 进行常数时间比较，避免时序攻击
