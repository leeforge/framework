package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CasbinPolicy holds the schema definition for the CasbinPolicy entity.
// 用于存储 Casbin RBAC/ABAC 策略规则
type CasbinPolicy struct {
	ent.Schema
}

// Fields of the CasbinPolicy.
func (CasbinPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.String("ptype").
			NotEmpty().
			MaxLen(10).
			Comment("Policy type: p, g, p2, etc."),
		field.String("v0").
			Optional().
			Comment("Policy value 0"),
		field.String("v1").
			Optional().
			Comment("Policy value 1"),
		field.String("v2").
			Optional().
			Comment("Policy value 2"),
		field.String("v3").
			Optional().
			Comment("Policy value 3"),
		field.String("v4").
			Optional().
			Comment("Policy value 4"),
		field.String("v5").
			Optional().
			Comment("Policy value 5"),
	}
}

// Indexes of the CasbinPolicy.
func (CasbinPolicy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ptype"),
		index.Fields("ptype", "v0", "v1", "v2", "v3", "v4", "v5"),
	}
}
