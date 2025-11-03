package jem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// AutoMap is a map of auto-generated field names to their values
type AutoMap = map[string]func() any

// Doc holds the deserialized and validated JSON document and its raw map representation
// on full, the `Value` and `Map` fields contain the full object
// on partial, the `Value` and `Map` fields contain a partial object
type Doc[T any] struct {
	// Value is the deserialized and validated JSON document
	Value *T
	// Map is the raw map representation of the JSON document
	Map map[string]any
}

// Factory is responsible for creating and validating entities
type Factory[T any] struct {
	entity *entity
}

// New creates a new factory for the given entity type
func New[T any]() *Factory[T] {
	var v T
	return &Factory[T]{entity: newEntity(v)}
}

// Make decodes and maps the JSON document to the entity and validates it
func (f *Factory[T]) Make(data []byte, auto ...AutoMap) (Doc[T], error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Doc[T]{}, fmt.Errorf("json parse to map failed: %w", err)
	}
	return f.MakeMap(m, auto...)
}

// MakeMany decodes and maps the JSON array to multiple entities and validates them
func (f *Factory[T]) MakeMany(data []byte, auto ...AutoMap) ([]Doc[T], error) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("json parse to array failed: %w", err)
	}
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}

	r := make([]Doc[T], len(arr))
	for i, m := range arr {
		res, err := f.MakeMap(m, auto...)
		if err != nil {
			return nil, fmt.Errorf("failed at index %d: %w", i, err)
		}
		r[i] = res
	}
	return r, nil
}

// MakeMap maps the JSON document to the entity and validates it
func (f *Factory[T]) MakeMap(m map[string]any, auto ...AutoMap) (Doc[T], error) {
	var r Doc[T]
	if len(f.entity.fields) == 0 {
		return r, fmt.Errorf("struct '%s' has zero fields with json tags", f.entity.name)
	}

	// check for empty input
	if len(m) == 0 {
		return r, errors.New("input is empty")
	}
	r.Map = m

	for k := range r.Map {
		fl, ok := f.entity.field(k)
		// verify field exists in entity
		if !ok {
			return r, fmt.Errorf("unknown field '%s'", k)
		}
		// verify id field does not exist in map, unless persist
		if fl.is(flagID) && !fl.is(flagPersist) {
			return r, fmt.Errorf("field '%s' with validation rule 'id' is not allowed in input", k)
		}
		// verify auto field does not exist in map
		if fl.isAuto() {
			return r, fmt.Errorf("field '%s' is not allowed in input", fl.name)
		}
		// verify readonly field does not exist in map
		if fl.is(flagReadonly) {
			return r, fmt.Errorf("field '%s' is readonly and not allowed in input", fl.name)
		}
	}

	if len(auto) > 0 {
		// apply auto-generated fields
		for k, fn := range auto[0] {
			if fl, ok := f.entity.field(k); !ok || !(fl.is(flagAuto) || fl.is(flagAutoFull)) {
				return r, fmt.Errorf("field '%s' is not an 'auto' or 'auto:full' field", k)
			}
			r.Map[k] = fn()
		}
	}

	// marshal for validation
	v, err := json.Marshal(&r.Map)
	if err != nil {
		return r, fmt.Errorf("json marshal failed: %w", err)
	}

	// use decoder to detect unknown nested fields
	decoder := json.NewDecoder(bytes.NewReader(v))
	decoder.DisallowUnknownFields()
	// decode
	if err := decoder.Decode(&r.Value); err != nil {
		return r, fmt.Errorf("json parse failed: %w", err)
	}
	// validate
	if f.entity.hasID() && !f.entity.id.is(flagPersist) {
		// do no validate id field, unless persist
		err = f.entity.validator.StructFiltered(r.Value, func(ns []byte) bool {
			fl := string(ns)
			return fl == f.entity.id.name
		})
	} else {
		err = f.entity.validator.Struct(r.Value)
	}
	if err != nil {
		return r, validationErrorHandler(err)
	}

	return r, nil
}

// MakeMapMany maps the JSON documents to multiple entities and validates them
func (f *Factory[T]) MakeMapMany(arr []map[string]any, auto ...AutoMap) ([]Doc[T], error) {
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}
	var r []Doc[T]
	for i, m := range arr {
		res, err := f.MakeMap(m, auto...)
		if err != nil {
			return nil, fmt.Errorf("failed at index %d: %w", i, err)
		}
		r = append(r, res)
	}
	return r, nil
}

// MakePartial decodes and maps the JSON document to the entity and validates it
func (f *Factory[T]) MakePartial(data []byte, auto ...AutoMap) (Doc[T], error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Doc[T]{}, fmt.Errorf("json parse to map failed: %w", err)
	}
	return f.MakePartialMap(m, auto...)
}

// MakePartialMany decodes and maps the JSON array to multiple entities and validates them
func (f *Factory[T]) MakePartialMany(data []byte, auto ...AutoMap) ([]Doc[T], error) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("json parse to array failed: %w", err)
	}
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}

	r := make([]Doc[T], len(arr))
	for i, m := range arr {
		res, err := f.MakePartialMap(m, auto...)
		if err != nil {
			return nil, fmt.Errorf("failed at index %d: %w", i, err)
		}
		r[i] = res
	}
	return r, nil
}

// MakePartialMap maps the JSON document to the entity and validates it
func (f *Factory[T]) MakePartialMap(m map[string]any, auto ...AutoMap) (Doc[T], error) {
	var r Doc[T]
	if len(f.entity.fields) == 0 {
		return r, fmt.Errorf("struct '%s' has zero fields with json tags", f.entity.name)
	}

	// check for empty input
	if len(m) == 0 {
		return r, errors.New("input is empty")
	}
	r.Map = m

	if f.entity.hasID() {
		// verify id field exists in map
		if _, ok := r.Map[f.entity.id.tag]; !ok {
			return r, errors.New("input must have id field")
		}
		// verify at least one other field exists
		if len(r.Map) < 2 {
			return r, errors.New("input must have id field and at least one other field")
		}
	}

	for k := range r.Map {
		fl, ok := f.entity.field(k)
		// verify field exists in entity
		if !ok {
			return r, fmt.Errorf("unknown field '%s'", k)
		}
		// verify auto field does not exist in map
		if fl.isAuto() {
			return r, fmt.Errorf("field '%s' is not allowed in input", fl.name)
		}
		// verify readonly field does not exist in map
		if fl.is(flagReadonly) {
			return r, fmt.Errorf("field '%s' is readonly and not allowed in input", fl.name)
		}
	}

	if len(auto) > 0 {
		// apply auto-generated fields
		for k, fn := range auto[0] {
			if fl, ok := f.entity.field(k); !ok || !(fl.is(flagAuto) || fl.is(flagAutoPartial)) {
				return r, fmt.Errorf("field '%s' is not an 'auto' or 'auto:partial' field", k)
			}
			r.Map[k] = fn()
		}
	}

	// filter top-level fields for validation
	filter := map[string]struct{}{}
	for k, v := range f.entity.fields {
		if v.is(flagAuto) || v.is(flagAutoPartial) {
			if _, ok := r.Map[k]; !ok {
				// add auto-generated field with default value for validation
				r.Map[k] = nil
			}
		}
		if _, ok := r.Map[k]; !ok {
			if v.is(flagPersist) {
				// persist fields must be in input
				return r, fmt.Errorf("field '%s' with validation rule 'persist' is missing", k)
			}
			// filter top-level field
			filter[v.name] = struct{}{}
		}
	}

	// marshal for validation
	v, err := json.Marshal(&r.Map)
	if err != nil {
		return r, fmt.Errorf("json marshal failed: %w", err)
	}

	// use decoder to detect unknown nested fields
	decoder := json.NewDecoder(bytes.NewReader(v))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&r.Value); err != nil {
		return r, fmt.Errorf("json parse failed: %w", err)
	}

	// validate
	if len(filter) > 0 {
		// validate partial fields
		err = f.entity.validator.StructFiltered(r.Value, func(ns []byte) bool {
			_, ok := filter[string(ns)]
			return ok
		})
	} else {
		err = f.entity.validator.Struct(r.Value)
	}
	if err != nil {
		return r, validationErrorHandler(err)
	}

	// remove id field from map
	if f.entity.hasID() {
		delete(r.Map, f.entity.id.tag)
	}

	return r, nil
}

// MakePartialMapMany maps the JSON documents to multiple entities and validates them
func (f *Factory[T]) MakePartialMapMany(
	arr []map[string]any,
	auto ...AutoMap,
) ([]Doc[T], error) {
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}
	var r []Doc[T]
	for i, m := range arr {
		res, err := f.MakePartialMap(m, auto...)
		if err != nil {
			return nil, fmt.Errorf("failed at index %d: %w", i, err)
		}
		r = append(r, res)
	}
	return r, nil
}

// Validator returns the validator for the factory
func (f *Factory[T]) Validator() *validator.Validate {
	return f.entity.validator
}

// validationErrorHandler handles validation errors and returns a formatted error
func validationErrorHandler(err error) error {
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, f := range ve {
			return fmt.Errorf(
				"field '%s' validation failed for rule '%s'",
				f.Namespace(),
				f.Tag(),
			)
		}
	}
	return err
}
