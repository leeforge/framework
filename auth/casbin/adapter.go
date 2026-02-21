package casbin

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/leeforge/framework/ent"
	"github.com/leeforge/framework/ent/casbinpolicy"
	"github.com/leeforge/framework/ent/predicate"
)

const (
	maxPolicyValues = 6
)

// EntAdapter Casbin Ent 适配器
// 将 Casbin 策略存储在 Ent 数据库中
type EntAdapter struct {
	client *ent.Client
}

// NewEntAdapter 创建新的 Ent 适配器
func NewEntAdapter(client *ent.Client) *EntAdapter {
	return &EntAdapter{client: client}
}

// LoadPolicy 从数据库加载策略到模型
func (a *EntAdapter) LoadPolicy(m model.Model) error {
	ctx := context.Background()

	policies, err := a.client.CasbinPolicy.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to load policies: %w", err)
	}

	for _, p := range policies {
		ptype := strings.TrimSpace(p.Ptype)
		if ptype == "" {
			continue
		}
		rule := extractRuleValues(p)
		if len(rule) == 0 {
			continue
		}

		sec := string(ptype[0])
		if len(sec) == 0 {
			continue
		}
		m.AddPolicy(sec, ptype, rule)
	}

	return nil
}

// SavePolicy 保存模型策略到数据库
func (a *EntAdapter) SavePolicy(m model.Model) error {
	ctx := context.Background()

	if _, err := a.client.CasbinPolicy.Delete().Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear policies: %w", err)
	}

	for sec, astMap := range m {
		for ptype, ast := range astMap {
			for _, rule := range ast.Policy {
				if err := a.addPolicy(ctx, sec, ptype, rule); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// AddPolicy 添加策略
func (a *EntAdapter) AddPolicy(sec, ptype string, rule []string) error {
	return a.addPolicy(context.Background(), sec, ptype, rule)
}

// RemovePolicy 删除策略
func (a *EntAdapter) RemovePolicy(sec, ptype string, rule []string) error {
	ctx := context.Background()

	_, err := a.client.CasbinPolicy.
		Delete().
		Where(matchPolicyPredicates(ptype, rule)...).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}
	return nil
}

// RemoveFilteredPolicy 删除过滤的策略
func (a *EntAdapter) RemoveFilteredPolicy(sec, ptype string, fieldIndex int, fieldValues ...string) error {
	ctx := context.Background()
	preds := []predicate.CasbinPolicy{
		casbinpolicy.PtypeEQ(ptype),
	}

	for i, value := range fieldValues {
		if value == "" {
			continue
		}

		switch fieldIndex + i {
		case 0:
			preds = append(preds, casbinpolicy.V0EQ(value))
		case 1:
			preds = append(preds, casbinpolicy.V1EQ(value))
		case 2:
			preds = append(preds, casbinpolicy.V2EQ(value))
		case 3:
			preds = append(preds, casbinpolicy.V3EQ(value))
		case 4:
			preds = append(preds, casbinpolicy.V4EQ(value))
		case 5:
			preds = append(preds, casbinpolicy.V5EQ(value))
		}
	}

	if _, err := a.client.CasbinPolicy.Delete().Where(preds...).Exec(ctx); err != nil {
		return fmt.Errorf("failed to remove filtered policy: %w", err)
	}
	return nil
}

// UpdatePolicy 更新策略
func (a *EntAdapter) UpdatePolicy(sec, ptype string, oldRule, newRule []string) error {
	ctx := context.Background()

	updater := a.client.CasbinPolicy.
		Update().
		Where(matchPolicyPredicates(ptype, oldRule)...)
	setRuleOnUpdate(updater, trimRule(newRule))
	if _, err := updater.Save(ctx); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}

// UpdatePolicies 批量更新策略
func (a *EntAdapter) UpdatePolicies(sec, ptype string, oldRules, newRules [][]string) error {
	ctx := context.Background()

	limit := len(oldRules)
	if len(newRules) < limit {
		limit = len(newRules)
	}
	for i := 0; i < limit; i++ {
		updater := a.client.CasbinPolicy.
			Update().
			Where(matchPolicyPredicates(ptype, oldRules[i])...)
		setRuleOnUpdate(updater, trimRule(newRules[i]))
		if _, err := updater.Save(ctx); err != nil {
			return fmt.Errorf("failed to update policy %d: %w", i, err)
		}
	}
	return nil
}

// ClearPolicy 清空所有策略
func (a *EntAdapter) ClearPolicy() error {
	ctx := context.Background()
	if _, err := a.client.CasbinPolicy.Delete().Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear policies: %w", err)
	}
	return nil
}

// LoadFilteredPolicy 加载过滤的策略 (可选实现)
func (a *EntAdapter) LoadFilteredPolicy(m model.Model, filter interface{}) error {
	return a.LoadPolicy(m)
}

// IsFiltered 检查是否过滤
func (a *EntAdapter) IsFiltered() bool {
	return false
}

// AddPolicies 批量添加策略
func (a *EntAdapter) AddPolicies(sec, ptype string, rules [][]string) error {
	ctx := context.Background()

	for i := range rules {
		if err := a.addPolicy(ctx, sec, ptype, rules[i]); err != nil {
			return fmt.Errorf("failed to add policy %d: %w", i, err)
		}
	}
	return nil
}

// RemovePolicies 批量删除策略
func (a *EntAdapter) RemovePolicies(sec, ptype string, rules [][]string) error {
	ctx := context.Background()

	for i := range rules {
		if _, err := a.client.CasbinPolicy.
			Delete().
			Where(matchPolicyPredicates(ptype, rules[i])...).
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to remove policy %d: %w", i, err)
		}
	}
	return nil
}

func (a *EntAdapter) addPolicy(ctx context.Context, sec, ptype string, rule []string) error {
	rule = trimRule(rule)
	if len(rule) == 0 {
		return nil
	}

	builder := a.client.CasbinPolicy.Create().SetPtype(ptype)
	setRuleOnCreate(builder, rule)
	if _, err := builder.Save(ctx); err != nil {
		return fmt.Errorf("failed to add policy: %w", err)
	}
	return nil
}

func extractRuleValues(p *ent.CasbinPolicy) []string {
	values := []string{p.V0, p.V1, p.V2, p.V3, p.V4, p.V5}
	return trimRule(values)
}

func trimRule(values []string) []string {
	limit := len(values)
	if limit > maxPolicyValues {
		limit = maxPolicyValues
	}
	trimmed := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		trimmed = append(trimmed, strings.TrimSpace(values[i]))
	}
	for len(trimmed) > 0 && trimmed[len(trimmed)-1] == "" {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}

func setRuleOnCreate(builder *ent.CasbinPolicyCreate, rule []string) {
	if len(rule) > 0 {
		builder.SetV0(rule[0])
	}
	if len(rule) > 1 {
		builder.SetV1(rule[1])
	}
	if len(rule) > 2 {
		builder.SetV2(rule[2])
	}
	if len(rule) > 3 {
		builder.SetV3(rule[3])
	}
	if len(rule) > 4 {
		builder.SetV4(rule[4])
	}
	if len(rule) > 5 {
		builder.SetV5(rule[5])
	}
}

func setRuleOnUpdate(updater *ent.CasbinPolicyUpdate, rule []string) {
	updater.ClearV0()
	updater.ClearV1()
	updater.ClearV2()
	updater.ClearV3()
	updater.ClearV4()
	updater.ClearV5()

	if len(rule) > 0 {
		updater.SetV0(rule[0])
	}
	if len(rule) > 1 {
		updater.SetV1(rule[1])
	}
	if len(rule) > 2 {
		updater.SetV2(rule[2])
	}
	if len(rule) > 3 {
		updater.SetV3(rule[3])
	}
	if len(rule) > 4 {
		updater.SetV4(rule[4])
	}
	if len(rule) > 5 {
		updater.SetV5(rule[5])
	}
}

func matchPolicyPredicates(ptype string, rule []string) []predicate.CasbinPolicy {
	rule = trimRule(rule)
	preds := []predicate.CasbinPolicy{
		casbinpolicy.PtypeEQ(ptype),
	}
	if len(rule) > 0 {
		preds = append(preds, casbinpolicy.V0EQ(rule[0]))
	} else {
		preds = append(preds, casbinpolicy.V0IsNil())
	}
	if len(rule) > 1 {
		preds = append(preds, casbinpolicy.V1EQ(rule[1]))
	} else {
		preds = append(preds, casbinpolicy.V1IsNil())
	}
	if len(rule) > 2 {
		preds = append(preds, casbinpolicy.V2EQ(rule[2]))
	} else {
		preds = append(preds, casbinpolicy.V2IsNil())
	}
	if len(rule) > 3 {
		preds = append(preds, casbinpolicy.V3EQ(rule[3]))
	} else {
		preds = append(preds, casbinpolicy.V3IsNil())
	}
	if len(rule) > 4 {
		preds = append(preds, casbinpolicy.V4EQ(rule[4]))
	} else {
		preds = append(preds, casbinpolicy.V4IsNil())
	}
	if len(rule) > 5 {
		preds = append(preds, casbinpolicy.V5EQ(rule[5]))
	} else {
		preds = append(preds, casbinpolicy.V5IsNil())
	}
	return preds
}
