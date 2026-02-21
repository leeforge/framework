package entities

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
	"github.com/leeforge/framework/utils"
)

// NewUUID generates a new UUID v7.
// It panics if the generation fails, as UUID v7 generation should not fail in normal circumstances.
func NewUUID() uuid.UUID {
	v7, err := uuid.NewV7()
	if err != nil {
		panic("failed to create UUID v7: " + err.Error())
	}
	return v7
}

type BaseEntitySchema struct {
	AuditedEntitySchema
}

// AuditedEntitySchema defines shared id/audit fields without tenant scoping.
type AuditedEntitySchema struct {
	mixin.Schema
}

// GlobalEntitySchema defines global/platform-scoped entities.
type GlobalEntitySchema struct {
	AuditedEntitySchema
}

// TenantEntitySchema defines tenant-scoped entities.
type TenantEntitySchema struct {
	AuditedEntitySchema
}

// IDField creates a UUID field with standard configuration.
// fieldName should be the database field name in snake_case (e.g., "id", "created_by_id").
// The JSON tag will be converted to lowerCamelCase (e.g., "id", "createdById").
func IDField(fieldName string) ent.Field {
	jsonTag := utils.LowerCamelCase(fieldName)
	return field.UUID(fieldName, uuid.UUID{}).
		Default(NewUUID).
		SchemaType(map[string]string{
			dialect.Postgres: "uuid",
			dialect.MySQL:    "char(36)",
			dialect.SQLite:   "text",
		}).
		Immutable().
		StructTag(`json:"` + jsonTag + `"`)
}

func (AuditedEntitySchema) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(NewUUID).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Immutable().
			Comment("唯一标识").
			StructTag(`json:"id"`),
		field.UUID("created_by_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Immutable().
			Optional().
			Comment("创建者ID").
			StructTag(`json:"createdById,omitempty"`),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Optional().
			Comment("创建时间").
			StructTag(`json:"createdAt,omitempty"`),
		field.UUID("updated_by_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Optional().
			Comment("更新者ID").
			StructTag(`json:"updatedById,omitempty"`),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Optional().
			Comment("更新时间").
			StructTag(`json:"updatedAt,omitempty"`),
		field.UUID("deleted_by_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Optional().
			Comment("删除者ID").
			StructTag(`json:"deletedById,omitempty"`),
		field.Time("deleted_at").
			Optional().
			Comment("删除时间").
			StructTag(`json:"deletedAt,omitempty"`),
		field.Time("published_at").
			Optional().
			Comment("发布时间").
			StructTag(`json:"publishedAt,omitempty"`),
		field.Time("archived_at").
			Optional().
			Comment("归档时间").
			StructTag(`json:"archivedAt,omitempty"`),
	}
}

func (AuditedEntitySchema) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id"),
		index.Fields("deleted_at"),
		index.Fields("created_at"),
		index.Fields("updated_at"),
		index.Fields("published_at"),
	}
}

// Fields returns domain-scoped entity fields (id + audit + owner_domain_id).
func (BaseEntitySchema) Fields() []ent.Field {
	fields := AuditedEntitySchema{}.Fields()
	return append(fields,
		field.UUID("owner_domain_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("Owner domain ID for domain-based isolation").
			StructTag(`json:"ownerDomainId,omitempty"`),
	)
}

// Indexes returns domain-scoped entity indexes.
func (BaseEntitySchema) Indexes() []ent.Index {
	indexes := AuditedEntitySchema{}.Indexes()
	return append(indexes, index.Fields("owner_domain_id"))
}

// Fields returns global/platform entity fields.
func (GlobalEntitySchema) Fields() []ent.Field {
	return AuditedEntitySchema{}.Fields()
}

// Indexes returns global/platform entity indexes.
func (GlobalEntitySchema) Indexes() []ent.Index {
	return AuditedEntitySchema{}.Indexes()
}

// Fields returns tenant-scoped fields and requires explicit tenant assignment.
func (TenantEntitySchema) Fields() []ent.Field {
	fields := AuditedEntitySchema{}.Fields()
	return append(fields,
		field.String("tenant_id").
			NotEmpty().
			Comment("租户ID").
			StructTag(`json:"tenantId"`),
	)
}

// Indexes returns tenant-scoped indexes.
func (TenantEntitySchema) Indexes() []ent.Index {
	indexes := AuditedEntitySchema{}.Indexes()
	return append(indexes, index.Fields("tenant_id"))
}

func UpdatedAtHook(next ent.Mutator) ent.Mutator {
	type UpdateTimeSetter interface {
		SetUpdateAt(time.Time)
	}
	return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		if setter, ok := m.(UpdateTimeSetter); ok {
			setter.SetUpdateAt(time.Now())
		}
		return next.Mutate(ctx, m)
	})
}

func DeletedAtHook(next ent.Mutator) ent.Mutator {
	type DeleteTimeSetter interface {
		SetDeleteAt(time.Time)
	}
	return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		if setter, ok := m.(DeleteTimeSetter); ok {
			setter.SetDeleteAt(time.Now())
		}
		return next.Mutate(ctx, m)
	})
}

type Status string

const (
	Draft     Status = "draft"
	Published Status = "published"
	Archived  Status = "archived"
)

type StatusContextKey struct{}

func WithStatus(ctx context.Context, status Status) context.Context {
	return context.WithValue(ctx, StatusContextKey{}, status)
}

func StatusFromContext(ctx context.Context) (Status, bool) {
	status, ok := ctx.Value(StatusContextKey{}).(Status)
	return status, ok
}

type ActiveStatus string

const (
	StatusActive   ActiveStatus = "active"
	StatusInactive ActiveStatus = "inactive"
)

func (s ActiveStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *ActiveStatus) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*s = ActiveStatus(v)
		return nil
	case []byte:
		*s = ActiveStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Status", value)
	}
}

type ActiveStatusSchema struct {
	mixin.Schema
}

func (ActiveStatusSchema) Fields() []ent.Field {
	return []ent.Field{
		field.String("active_status").
			GoType(ActiveStatus("")).
			SchemaType(map[string]string{
				dialect.MySQL:    "enum('active', 'inactive')",
				dialect.Postgres: "varchar(20)",
			}).
			Default(string(StatusActive)).
			NotEmpty(),
	}
}
