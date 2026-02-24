# entities — Ent Schema 基础 Mixin

提供所有业务实体共用的 Ent ORM Mixin，统一主键格式、审计字段与多租户隔离字段。

## Mixin 列表

| Mixin | 用途 | 额外字段 |
|---|---|---|
| `AuditedEntitySchema` | 基础审计字段 | 无额外字段，仅包含标准审计字段 |
| `BaseEntitySchema` | 领域隔离实体（推荐） | `ownerDomainId` |
| `GlobalEntitySchema` | 全局/平台级实体（无域隔离） | 无 |
| `TenantEntitySchema` | 多租户实体 | `tenantId`（NOT NULL） |

### 共用审计字段

所有 Mixin 均包含以下字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | UUID v7 | 主键，不可变 |
| `createdById` | UUID | 创建者 ID |
| `createdAt` | Time | 创建时间 |
| `updatedById` | UUID | 更新者 ID |
| `updatedAt` | Time | 更新时间 |
| `deletedById` | UUID | 软删除者 ID |
| `deletedAt` | Time | 软删除时间 |
| `publishedAt` | Time | 发布时间 |
| `archivedAt` | Time | 归档时间 |

## 使用方式

```go
package schema

import (
    "entgo.io/ent"
    "github.com/leeforge/framework/entities"
)

// 继承 BaseEntitySchema（推荐）
type Article struct {
    entities.BaseEntitySchema
}

func (Article) Fields() []ent.Field {
    // 当使用外键 Edge 时，必须显式定义 id 字段
    return []ent.Field{
        entities.IDField("id"),
    }
}

func (Article) Edges() []ent.Edge {
    return []ent.Edge{
        // 定义外键关系
    }
}
```

## UUID v7 生成

框架提供 `NewUUID()` 函数生成有序的 UUID v7（时间戳有序，天然支持数据库索引优化）：

```go
import "github.com/leeforge/framework/entities"

id := entities.NewUUID() // 生成 UUID v7

// 在 Schema 中使用
field.UUID("id", uuid.UUID{}).Default(entities.NewUUID).Immutable()
```

**禁止**直接调用第三方 UUID 库，必须使用此 `NewUUID()`，保证全局 ID 格式一致。

## 状态枚举

```go
// 内容发布状态
entities.Draft     // "draft"
entities.Published // "published"
entities.Archived  // "archived"

// 激活状态
entities.StatusActive   // "active"
entities.StatusInactive // "inactive"
```

## 索引

`AuditedEntitySchema` 自动创建以下索引：
- `id`（主键）
- `deleted_at`（软删除过滤）
- `created_at` / `updated_at` / `published_at`（时间排序）

## 注意事项

- 使用 `BaseEntitySchema` 且涉及 Edge 外键时，**必须**在子 Schema 中显式调用 `entities.IDField("id")` 定义主键，否则 Ent 可能回退到 int 类型
- 软删除逻辑由业务层自行实现（框架不自动过滤 `deleted_at != null`），建议配合 Ent Interceptor 使用
