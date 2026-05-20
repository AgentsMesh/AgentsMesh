// Package gormvalidate wires backend/pkg/slugkit.IdentifierValidator into
// GORM's persistence lifecycle. It registers Before-Create and
// Before-Update callbacks that call ValidateIdentifiers() on any model
// implementing the interface, restoring Layer 2 of the identifier
// contract defense without forcing domain packages to import gorm.
package gormvalidate

import (
	"reflect"
	"sync"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

type Plugin struct{}

func (p *Plugin) Name() string { return "agentsmesh:identifier_validator" }

func (p *Plugin) Initialize(db *gorm.DB) error {
	cb := db.Callback()
	if err := cb.Create().Before("gorm:create").Register("identifier_validator:create", validate); err != nil {
		return err
	}
	return cb.Update().Before("gorm:update").Register("identifier_validator:update", validate)
}

// validatorIface caches reflect.Type → does it (or its pointer) implement
// IdentifierValidator. Avoids per-row Addr().Interface() allocation for
// model types known not to implement the interface (Message, membership
// join tables, etc.).
var (
	validatorIface    = reflect.TypeOf((*slugkit.IdentifierValidator)(nil)).Elem()
	implementsIVCache sync.Map // map[reflect.Type]bool
)

// implementsValidator reports whether the model type (after unwrapping any
// pointer indirection) is an IdentifierValidator via its pointer receiver.
// Result is memoised per reflect.Type.
func implementsValidator(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if cached, ok := implementsIVCache.Load(t); ok {
		return cached.(bool)
	}
	ok := reflect.PointerTo(t).Implements(validatorIface)
	implementsIVCache.Store(t, ok)
	return ok
}

// validate runs ValidateIdentifiers on every struct or slice element in
// db.Statement.ReflectValue that implements slugkit.IdentifierValidator.
// Errors are forwarded via db.AddError so the surrounding Create/Update
// fails with the same shape as a hook returning an error.
func validate(db *gorm.DB) {
	if db.Statement == nil {
		return
	}
	v := db.Statement.ReflectValue
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			return
		}
		if !implementsValidator(v.Type().Elem()) {
			return
		}
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() != reflect.Ptr {
				if !elem.CanAddr() {
					continue
				}
				elem = elem.Addr()
			}
			invoke(db, elem.Interface().(slugkit.IdentifierValidator))
		}
	case reflect.Struct:
		if !v.CanAddr() {
			return
		}
		if !implementsValidator(v.Type()) {
			return
		}
		invoke(db, v.Addr().Interface().(slugkit.IdentifierValidator))
	}
}

func invoke(db *gorm.DB, iv slugkit.IdentifierValidator) {
	if err := iv.ValidateIdentifiers(); err != nil {
		_ = db.AddError(err)
	}
}
