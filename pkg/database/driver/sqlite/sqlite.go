package sqlite

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Option func(opts *Config)

type Config struct {
	DSN string
}

type SQLite struct {
	instance gorm.Dialector
	Config   Config
}

func New(opts ...Option) *SQLite {
	s := &SQLite{}
	for _, f := range opts {
		f(&s.Config)
	}
	s.instance = sqlite.Open(s.Config.DSN)
	return s
}

func (s *SQLite) Instance() gorm.Dialector {
	return s.instance
}

func WithDSN(dsn string) Option {
	return func(opts *Config) {
		opts.DSN = dsn
	}
}
