# Configuration System (Frame Core)

基于 `spf13/viper` 封装的现代化配置加载组件，支持多环境配置文件、环境变量注入及热更新。

## 🌟 特性 (Features)

- **多环境支持**: 自动根据 `APP_ENV` (dev/test/prod) 加载对应的配置文件。
- **环境变量注入**: 支持通过环境变量覆盖 YAML 配置，支持自定义前缀（如 `LEEFORGE_`）。
- **智能映射**: 自动将环境变量中的下划线 `_` 转换为配置层级分隔符 `.`。
- **结构化绑定**: 支持 `mapstructure` 标签，直接将配置绑定到 Go Struct。
- **热更新 (Watch)**: 支持配置文件变更监听（开发模式下默认开启）。

## 🚀 快速开始 (Usage)

### 1. 基础用法

```go
opts := config.DefaultConfigOptions()
opts.EnvPrefix = "MYAPP" // 设置环境变量前缀

cfg, err := config.NewConfig(opts)
if err != nil {
    panic(err)
}

var myConfig MyConfigStruct
cfg.Bind(&myConfig)
```

### 2. 环境变量规则

当设置了 `EnvPrefix` 时，环境变量的映射规则如下：

`[PREFIX]_[SECTION]_[KEY]` -> `section.key`

例如，当 `EnvPrefix = "MYAPP"` 时：

| 环境变量 | 对应的 YAML/配置项 |
| :--- | :--- |
| `MYAPP_SERVER_PORT` | `server.port` |
| `MYAPP_DATABASE_URL` | `database.url` |
| `MYAPP_INIT_SECRET_KEY` | `init.secret_key` |

### 3. 配置选项 (Options)

```go
type ConfigOptions struct {
    BasePath  string // 配置文件目录，默认为 ./configs 或 env:CONFIG_PATH
    FileName  string // 文件名，默认为 config
    FileType  string // 文件类型，默认为 yaml
    EnvPrefix string // 环境变量前缀 (自动转大写)
    WatchAble bool   // 是否开启热更新监听
    LoadAll   bool   // 是否加载目录下所有配置文件
}
```
