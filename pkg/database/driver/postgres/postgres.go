package postgres

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Option func(opts *postgres.Config)

type Postgres struct {
	instance gorm.Dialector
	Config   postgres.Config
}

func New(opts ...Option) *Postgres {
	pg := &Postgres{}
	for _, f := range opts {
		f(&pg.Config)
	}
	pg.instance = postgres.New(pg.Config)
	return pg
}

func (p *Postgres) Instance() gorm.Dialector {
	return p.instance
}
