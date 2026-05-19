package database

import (
	"context"

	"gorm.io/gorm"
)

type DB interface {
	Transaction(fc func(tx DB) error) error
	WithContext(ctx context.Context) DB

	Create(value interface{}) error
	First(dest interface{}, conds ...interface{}) error
	Find(dest interface{}, conds ...interface{}) error
	Save(value interface{}) error
	Delete(value interface{}, conds ...interface{}) error
	Updates(model interface{}, values interface{}) error

	Model(value interface{}) DB
	Table(name string) DB
	Where(query interface{}, args ...interface{}) DB
	Select(query interface{}, args ...interface{}) DB
	Joins(query string, args ...interface{}) DB
	Preload(query string, args ...interface{}) DB
	Order(value interface{}) DB
	Limit(limit int) DB
	Offset(offset int) DB
	Group(name string) DB
	Count(count *int64) error
	Scan(dest interface{}) error

	GormDB() *gorm.DB
}

type GormWrapper struct {
	db *gorm.DB
}

func NewGormWrapper(db *gorm.DB) *GormWrapper {
	return &GormWrapper{db: db}
}

func (w *GormWrapper) Transaction(fc func(tx DB) error) error {
	return w.db.Transaction(func(tx *gorm.DB) error {
		return fc(&GormWrapper{db: tx})
	})
}

func (w *GormWrapper) WithContext(ctx context.Context) DB {
	return &GormWrapper{db: w.db.WithContext(ctx)}
}

func (w *GormWrapper) Create(value interface{}) error {
	return w.db.Create(value).Error
}

func (w *GormWrapper) First(dest interface{}, conds ...interface{}) error {
	return w.db.First(dest, conds...).Error
}

func (w *GormWrapper) Find(dest interface{}, conds ...interface{}) error {
	return w.db.Find(dest, conds...).Error
}

func (w *GormWrapper) Save(value interface{}) error {
	return w.db.Save(value).Error
}

func (w *GormWrapper) Delete(value interface{}, conds ...interface{}) error {
	return w.db.Delete(value, conds...).Error
}

func (w *GormWrapper) Updates(model interface{}, values interface{}) error {
	return w.db.Model(model).Updates(values).Error
}

func (w *GormWrapper) Model(value interface{}) DB {
	return &GormWrapper{db: w.db.Model(value)}
}

func (w *GormWrapper) Table(name string) DB {
	return &GormWrapper{db: w.db.Table(name)}
}

func (w *GormWrapper) Where(query interface{}, args ...interface{}) DB {
	return &GormWrapper{db: w.db.Where(query, args...)}
}

func (w *GormWrapper) Select(query interface{}, args ...interface{}) DB {
	return &GormWrapper{db: w.db.Select(query, args...)}
}

func (w *GormWrapper) Joins(query string, args ...interface{}) DB {
	return &GormWrapper{db: w.db.Joins(query, args...)}
}

func (w *GormWrapper) Preload(query string, args ...interface{}) DB {
	return &GormWrapper{db: w.db.Preload(query, args...)}
}

func (w *GormWrapper) Order(value interface{}) DB {
	return &GormWrapper{db: w.db.Order(value)}
}

func (w *GormWrapper) Limit(limit int) DB {
	return &GormWrapper{db: w.db.Limit(limit)}
}

func (w *GormWrapper) Offset(offset int) DB {
	return &GormWrapper{db: w.db.Offset(offset)}
}

func (w *GormWrapper) Group(name string) DB {
	return &GormWrapper{db: w.db.Group(name)}
}

func (w *GormWrapper) Count(count *int64) error {
	return w.db.Count(count).Error
}

func (w *GormWrapper) Scan(dest interface{}) error {
	return w.db.Scan(dest).Error
}

func (w *GormWrapper) GormDB() *gorm.DB {
	return w.db
}

var _ DB = (*GormWrapper)(nil)
