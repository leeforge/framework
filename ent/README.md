# ent — Ent ORM 生成代码

> ⚠️ 此目录为自动生成代码，**禁止手动修改**。
>
> 修改方式：编辑 `ent/schema/` 中的 Schema 定义，然后运行代码生成命令。

## 生成的实体

| 实体 | 说明 |
|---|---|
| `CasbinPolicy` | Casbin RBAC 策略规则存储（`auth` 模块使用） |
| `Media` | 媒体文件记录（文件名、大小、MIME 类型、URL 等）|
| `MediaFormat` | 媒体文件的各种格式/尺寸变体（缩略图、小图等）|

## 代码生成

```bash
# 在 framework 目录下执行
go generate ./ent/...

# 或使用 Makefile
make generate
```

## Schema 定义位置

```
ent/
├── schema/
│   ├── casbinpolicy.go   # CasbinPolicy Schema
│   ├── media.go          # Media Schema
│   └── mediaformat.go    # MediaFormat Schema
├── generate.go           # go:generate 指令
└── ...（生成代码）
```

## 基础 Mixin

所有 Schema 均继承自 `entities` 模块提供的 Mixin（参见 [entities/README.md](../entities/README.md)）：

- `GlobalEntitySchema`：全局实体，含标准审计字段与 UUID v7 主键
- `BaseEntitySchema`：域隔离实体，额外包含 `ownerDomainId`

## 添加新实体

1. 在 `ent/schema/` 目录创建新的 Schema 文件（如 `my_entity.go`）
2. 选择合适的 Mixin（`GlobalEntitySchema` / `BaseEntitySchema` / `TenantEntitySchema`）
3. 定义业务字段与 Edge
4. 运行 `go generate ./ent/...` 生成代码
5. 如涉及外键 Edge，必须显式声明 `entities.IDField("id")` 防止主键回退

## 注意事项

- `ent/generate.go` 中配置了生成选项（Feature Flag、注解等），修改前请了解 Ent 文档
- Schema 变更后需同步运行数据库迁移
- 生成代码已通过 `.gitignore` 设置提交到仓库（方便 CI 直接使用），无需每次生成
