package database

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"gorm.io/gorm"
)

type MockDB struct {
	mu sync.RWMutex

	data map[string][]interface{} // table name -> records

	model       interface{}
	tableName   string
	conditions  []mockCondition
	preloads    []string
	orderBy     interface{}
	limitVal    int
	offsetVal   int
	selectQuery interface{}
	selectArgs  []interface{}
	groupBy     string
	joins       []mockJoin

	CreateErr      error
	FirstErr       error
	FindErr        error
	SaveErr        error
	DeleteErr      error
	UpdatesErr     error
	CountErr       error
	ScanErr        error
	TransactionErr error

	CreatedRecords  []interface{}
	UpdatedRecords  []interface{}
	DeletedRecords  []interface{}
	QueriedTables   []string
	TransactionHits int
}

type mockCondition struct {
	query interface{}
	args  []interface{}
}

type mockJoin struct {
	query string
	args  []interface{}
}

func NewMockDB() *MockDB {
	return &MockDB{
		data: make(map[string][]interface{}),
	}
}

func (m *MockDB) clone() *MockDB {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MockDB{
		data:            m.data,
		model:           m.model,
		tableName:       m.tableName,
		conditions:      append([]mockCondition{}, m.conditions...),
		preloads:        append([]string{}, m.preloads...),
		orderBy:         m.orderBy,
		limitVal:        m.limitVal,
		offsetVal:       m.offsetVal,
		selectQuery:     m.selectQuery,
		selectArgs:      m.selectArgs,
		groupBy:         m.groupBy,
		joins:           append([]mockJoin{}, m.joins...),
		CreateErr:       m.CreateErr,
		FirstErr:        m.FirstErr,
		FindErr:         m.FindErr,
		SaveErr:         m.SaveErr,
		DeleteErr:       m.DeleteErr,
		UpdatesErr:      m.UpdatesErr,
		CountErr:        m.CountErr,
		ScanErr:         m.ScanErr,
		TransactionErr:  m.TransactionErr,
		CreatedRecords:  m.CreatedRecords,
		UpdatedRecords:  m.UpdatedRecords,
		DeletedRecords:  m.DeletedRecords,
		QueriedTables:   m.QueriedTables,
		TransactionHits: m.TransactionHits,
	}
}

func (m *MockDB) Transaction(fc func(tx DB) error) error {
	m.mu.Lock()
	m.TransactionHits++
	m.mu.Unlock()

	if m.TransactionErr != nil {
		return m.TransactionErr
	}
	return fc(m)
}

func (m *MockDB) WithContext(ctx context.Context) DB {
	return m.clone()
}

func (m *MockDB) Create(value interface{}) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreatedRecords = append(m.CreatedRecords, value)

	setID(value, int64(len(m.CreatedRecords)))

	tableName := getTableName(value)
	m.data[tableName] = append(m.data[tableName], value)
	return nil
}

func (m *MockDB) First(dest interface{}, conds ...interface{}) error {
	if m.FirstErr != nil {
		return m.FirstErr
	}
	return gorm.ErrRecordNotFound
}

func (m *MockDB) Find(dest interface{}, conds ...interface{}) error {
	if m.FindErr != nil {
		return m.FindErr
	}
	return nil
}

func (m *MockDB) Save(value interface{}) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	m.UpdatedRecords = append(m.UpdatedRecords, value)
	m.mu.Unlock()
	return nil
}

func (m *MockDB) Delete(value interface{}, conds ...interface{}) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	m.DeletedRecords = append(m.DeletedRecords, value)
	m.mu.Unlock()
	return nil
}

func (m *MockDB) Updates(model interface{}, values interface{}) error {
	if m.UpdatesErr != nil {
		return m.UpdatesErr
	}
	m.mu.Lock()
	m.UpdatedRecords = append(m.UpdatedRecords, map[string]interface{}{
		"model":  model,
		"values": values,
	})
	m.mu.Unlock()
	return nil
}

func (m *MockDB) Model(value interface{}) DB {
	c := m.clone()
	c.model = value
	c.tableName = getTableName(value)
	return c
}

func (m *MockDB) Table(name string) DB {
	c := m.clone()
	c.tableName = name
	m.mu.Lock()
	m.QueriedTables = append(m.QueriedTables, name)
	m.mu.Unlock()
	return c
}

func (m *MockDB) Where(query interface{}, args ...interface{}) DB {
	c := m.clone()
	c.conditions = append(c.conditions, mockCondition{query: query, args: args})
	return c
}

func (m *MockDB) Select(query interface{}, args ...interface{}) DB {
	c := m.clone()
	c.selectQuery = query
	c.selectArgs = args
	return c
}

func (m *MockDB) Joins(query string, args ...interface{}) DB {
	c := m.clone()
	c.joins = append(c.joins, mockJoin{query: query, args: args})
	return c
}

func (m *MockDB) Preload(query string, args ...interface{}) DB {
	c := m.clone()
	c.preloads = append(c.preloads, query)
	return c
}

func (m *MockDB) Order(value interface{}) DB {
	c := m.clone()
	c.orderBy = value
	return c
}

func (m *MockDB) Limit(limit int) DB {
	c := m.clone()
	c.limitVal = limit
	return c
}

func (m *MockDB) Offset(offset int) DB {
	c := m.clone()
	c.offsetVal = offset
	return c
}

func (m *MockDB) Group(name string) DB {
	c := m.clone()
	c.groupBy = name
	return c
}

func (m *MockDB) Count(count *int64) error {
	if m.CountErr != nil {
		return m.CountErr
	}
	*count = 0
	return nil
}

func (m *MockDB) Scan(dest interface{}) error {
	if m.ScanErr != nil {
		return m.ScanErr
	}
	return nil
}

func (m *MockDB) GormDB() *gorm.DB {
	return nil
}

func (m *MockDB) SetData(tableName string, records []interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[tableName] = records
}

// GetCreatedRecords returns all created records (thread-safe).
func (m *MockDB) GetCreatedRecords() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]interface{}, len(m.CreatedRecords))
	copy(result, m.CreatedRecords)
	return result
}

// GetUpdatedRecords returns all updated records (thread-safe).
func (m *MockDB) GetUpdatedRecords() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]interface{}, len(m.UpdatedRecords))
	copy(result, m.UpdatedRecords)
	return result
}

// GetDeletedRecords returns all deleted records (thread-safe).
func (m *MockDB) GetDeletedRecords() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]interface{}, len(m.DeletedRecords))
	copy(result, m.DeletedRecords)
	return result
}

func (m *MockDB) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]interface{})
	m.CreatedRecords = nil
	m.UpdatedRecords = nil
	m.DeletedRecords = nil
	m.QueriedTables = nil
	m.TransactionHits = 0
	m.conditions = nil
	m.preloads = nil
	m.joins = nil
	m.model = nil
	m.tableName = ""
}

func getTableName(value interface{}) string {
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

func setID(value interface{}, id int64) {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	idField := v.FieldByName("ID")
	if idField.IsValid() && idField.CanSet() && idField.Kind() == reflect.Int64 {
		idField.SetInt(id)
	}
}

func (m *MockDB) SetFirstResult(result interface{}, err error) {
	m.FirstErr = err
}

func (m *MockDB) SetFindResult(result interface{}, err error) {
	m.FindErr = err
}

var _ DB = (*MockDB)(nil)

var ErrMockNotImplemented = errors.New("mock: operation not implemented")
