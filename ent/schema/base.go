package schema

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
)

type BaseEntitySchema struct {
	mixin.Schema
}

func (BaseEntitySchema) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				v7, err := uuid.NewV7()
				if err != nil {
					panic("failed to create UUID v7: " + err.Error())
				}
				return v7
			}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Immutable().
			Comment("唯一标识"),
		field.String("tenant_id").
			Default("default").
			NotEmpty().
			Comment("租户ID"),
		field.UUID("created_by_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Immutable().
			Optional().
			Comment("创建者ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Optional().
			Comment("创建时间"),
		field.UUID("updated_by_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Optional().
			Comment("更新者ID"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Optional().
			Comment("更新时间"),
		field.UUID("deleted_by_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Optional().
			Comment("删除者ID"),
		field.Time("deleted_at").
			Optional().
			Comment("删除时间"),
		field.Time("published_at").
			Optional().
			Comment("发布时间"),
		field.Time("archived_at").
			Optional().
			Comment("归档时间"),
	}
}

func (BaseEntitySchema) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id"),
		index.Fields("tenant_id"),
		index.Fields("deleted_at"),
		index.Fields("created_at"),
		index.Fields("updated_at"),
		index.Fields("published_at"),
	}
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
