package jem

import (
	"reflect"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
)

// flag represents a field flag
type flag uint8

// field flags
const (
	flagID flag = 1 << iota
	flagAuto
	flagAutoFull
	flagAutoPartial
	flagPersist
	flagReadonly
)

// field represents a struct field
type field struct {
	flags flag
	name  string
	tag   string
}

// is checks if the field has the given flag
func (f *field) is(fl flag) bool {
	return f.flags&fl != 0
}

// isAuto checks if the field has an auto flag
func (f *field) isAuto() bool {
	return f.is(flagAuto) || f.is(flagAutoFull) || f.is(flagAutoPartial)
}

// entity represents a struct entity
type entity struct {
	fields    map[string]field
	id        *field
	name      string
	validator *validator.Validate
}

// newEntity creates a new entity from a struct
func newEntity(v any) *entity {
	e := &entity{
		fields:    map[string]field{},
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		panic("entity must be a struct")
	}
	e.name = t.Name() // struct name
	fm := map[string]flag{
		"auto":         flagAuto,
		"auto:full":    flagAutoFull,
		"auto:partial": flagAutoPartial,
		"persist":      flagPersist,
		"readonly":     flagReadonly,
	}
	var flags flag
	for i := 0; i < t.NumField(); i++ {
		fl := t.Field(i)

		jTags := fieldTagValues("json", fl)
		var jt string
		if len(jTags) > 0 {
			jt = jTags[0]
		}
		if jt == "" || jt == "-" {
			continue // ignore fields without json tag or with "-"
		}

		vt := fieldTagValues("validate", fl)
		f := field{
			name: e.name + "." + fl.Name, // fully qualified name
			tag:  jt,
		}

		// set flags
		if len(vt) > 0 {
			if slices.Contains(vt, "id") {
				if e.hasID() {
					panic("struct '" + e.name + "' has multiple fields with 'id' validation rule")
				}
				f.flags |= flagID
				flags |= flagID
				e.id = &f
			}
			for k, v := range fm {
				if slices.Contains(vt, k) {
					f.flags |= v
					flags |= v
				}
			}
		}

		// add field to entity
		e.fields[jt] = f
	}

	jsonTagFn := func(fl reflect.StructField) string {
		v := fl.Tag.Get("json")
		if v == "-" || v == "" {
			return ""
		}
		return v
	}
	e.validator.RegisterTagNameFunc(jsonTagFn)

	rulesFn := func(fl validator.FieldLevel) bool { return true }
	rules := map[string]flag{
		"id":           flagID,
		"auto":         flagAuto,
		"auto:full":    flagAutoFull,
		"auto:partial": flagAutoPartial,
		"persist":      flagPersist,
		"readonly":     flagReadonly,
	}
	// register custom validation rules
	for k, v := range rules {
		if flags&v != 0 {
			err := e.validator.RegisterValidation(k, rulesFn)
			if err != nil {
				panic("failed to register validation for " + k + ": " + err.Error())
			}
		}
	}

	return e
}

// field returns the field with the given tag
func (e *entity) field(tag string) (field, bool) {
	f, ok := e.fields[tag]
	return f, ok
}

// hasID checks if the entity has an ID field
func (e *entity) hasID() bool {
	return e.id != nil
}

// fieldTagValues returns the values of the given struct field tag
func fieldTagValues(tag string, f reflect.StructField) []string {
	v := f.Tag.Get(tag)
	if v == "" {
		return []string{}
	}
	return strings.Split(v, ",")
}
