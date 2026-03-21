package elgon

import (
	"github.com/dmesha3/elgon/db"
	"github.com/dmesha3/elgon/orm"
)

// SetSQL sets the application's raw SQL adapter used by SQL and ORM accessors.
func (a *App) SetSQL(adapter db.Adapter) {
	a.dataMu.Lock()
	defer a.dataMu.Unlock()
	a.sqlDB = adapter
	a.ormClient = nil
}

// SetORMDialect sets placeholder behavior for app.ORM() repositories.
func (a *App) SetORMDialect(dialect string) {
	a.dataMu.Lock()
	defer a.dataMu.Unlock()
	a.ormDialect = dialect
	a.ormClient = nil
}

// SQL returns the configured raw SQL adapter.
func (a *App) SQL() db.Adapter {
	a.dataMu.RLock()
	defer a.dataMu.RUnlock()
	return a.sqlDB
}

// ORM returns a typed ORM client backed by the configured SQL adapter.
func (a *App) ORM() *orm.Client {
	a.dataMu.RLock()
	sqlDB := a.sqlDB
	client := a.ormClient
	a.dataMu.RUnlock()

	if sqlDB == nil {
		return nil
	}
	if client != nil {
		return client
	}

	a.dataMu.Lock()
	defer a.dataMu.Unlock()
	if a.sqlDB == nil {
		return nil
	}
	if a.ormClient == nil {
		a.ormClient = orm.NewWithConfig(a.sqlDB, orm.Config{Dialect: a.ormDialect})
	}
	return a.ormClient
}
