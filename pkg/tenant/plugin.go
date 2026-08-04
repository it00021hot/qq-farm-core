package tenant

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const pluginName = "go-skeleton:tenant"

// Plugin GORM 租户行级隔离插件
type Plugin struct{}

func (Plugin) Name() string { return pluginName }

func (p Plugin) Initialize(db *gorm.DB) error {
	cbs := []struct {
		name string
		fn   func(*gorm.DB)
		hook string
	}{
		{pluginName + ":before_create", beforeCreate, "gorm:create"},
		{pluginName + ":before_query", beforeQuery, "gorm:query"},
		{pluginName + ":before_update", beforeUpdate, "gorm:update"},
		{pluginName + ":before_delete", beforeDelete, "gorm:delete"},
	}
	for _, c := range cbs {
		var err error
		switch {
		case c.hook == "gorm:create":
			err = db.Callback().Create().Before(c.hook).Register(c.name, c.fn)
		case c.hook == "gorm:query":
			err = db.Callback().Query().Before(c.hook).Register(c.name, c.fn)
		case c.hook == "gorm:update":
			err = db.Callback().Update().Before(c.hook).Register(c.name, c.fn)
		case c.hook == "gorm:delete":
			err = db.Callback().Delete().Before(c.hook).Register(c.name, c.fn)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func beforeCreate(db *gorm.DB) {
	if shouldSkip(db) || db.Statement.Schema == nil || !hasTenantColumn(db.Statement.Schema) {
		return
	}
	tid, ok := IDFrom(db.Statement.Context)
	if !ok {
		return
	}
	if m, ok := db.Statement.Dest.(Model); ok {
		m.SetTenantID(tid)
		return
	}
	db.Statement.SetColumn("tenant_id", tid, true)
}

func beforeQuery(db *gorm.DB)  { applyTenantWhere(db) }
func beforeUpdate(db *gorm.DB) { applyTenantWhere(db) }
func beforeDelete(db *gorm.DB) { applyTenantWhere(db) }

func applyTenantWhere(db *gorm.DB) {
	if shouldSkip(db) || db.Statement.Schema == nil || !hasTenantColumn(db.Statement.Schema) {
		return
	}
	tid, ok := IDFrom(db.Statement.Context)
	if !ok {
		return
	}
	db.Statement.AddClause(clause.Where{
		Exprs: []clause.Expression{
			clause.Eq{Column: clause.Column{Table: db.Statement.Table, Name: "tenant_id"}, Value: tid},
		},
	})
}

func shouldSkip(db *gorm.DB) bool {
	if db == nil || db.Statement == nil {
		return true
	}
	return IsSkip(db.Statement.Context)
}

func hasTenantColumn(s *schema.Schema) bool {
	if s == nil {
		return false
	}
	_, ok := s.FieldsByDBName["tenant_id"]
	return ok
}

// Register 注册插件
func Register(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Use(Plugin{})
}

// Scope 返回带租户 context 的 DB（业务入口统一使用）
func Scope(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
}

// Global 跳过租户过滤的 DB
func Global(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.WithContext(WithSkip(ctx))
}
