package entities

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

// DomainScopeMixin adds an optional owner_domain_id field for domain-scoped data isolation.
// Entities using this mixin can be filtered by the DomainInterceptor based on the acting domain.
type DomainScopeMixin struct {
	mixin.Schema
}

func (DomainScopeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("owner_domain_id", uuid.UUID{}).
			SchemaType(map[string]string{
				dialect.Postgres: "uuid",
				dialect.MySQL:    "char(36)",
				dialect.SQLite:   "text",
			}).
			Optional().
			Nillable().
			Comment("Domain ID that owns this resource").
			StructTag(`json:"ownerDomainId,omitempty"`),
	}
}

func (DomainScopeMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner_domain_id"),
	}
}
