package jem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/go-playground/validator/v10"
)

// ErrRead is returned when reading from the reader fails
var ErrRead = errors.New("failed to read input")

// AutoMap is a map of auto-generated field names to their values
type AutoMap = map[string]func() any

// Doc holds the deserialized and validated JSON document and its raw map representation
// on full, the `Value` and `Map` fields contain the full object
// on partial, the `Value` and `Map` fields contain a partial object
type Doc[T any, ID comparable] struct {
	// ID is the value of the ID field
	ID ID
	// Map is the raw map representation of the JSON document
	Map map[string]any
	// Value is the deserialized and validated JSON document
	Value *T
}

// Factory is responsible for creating and validating entities
type Factory[T any, ID comparable] struct {
	entity  *entity
	parseID func(any) (ID, bool)
}

// New creates a new factory for the given entity type
func New[T any, ID comparable]() *Factory[T, ID] {
	var v T
	return &Factory[T, ID]{entity: newEntity(v)}
}

// Make decodes and maps the JSON document to the entity and validates it
func (f *Factory[T, ID]) Make(data []byte, auto ...AutoMap) (Doc[T, ID], error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Doc[T, ID]{}, fmt.Errorf("json parse to map failed: %w", err)
	}
	return f.MakeMap(m, auto...)
}

// MakeMany decodes and maps the JSON array to multiple entities and validates them
func (f *Factory[T, ID]) MakeMany(data []byte, auto ...AutoMap) ([]Doc[T, ID], error) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("json parse to array failed: %w", err)
	}
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}

	r := make([]Doc[T, ID], len(arr))
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
func (f *Factory[T, ID]) MakeMap(m map[string]any, auto ...AutoMap) (Doc[T, ID], error) {
	var r Doc[T, ID]
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
		// verify auto field does not exist in map
		if fl.isAuto() {
			return r, fmt.Errorf("field '%s' is not allowed in input", fl.name)
		}
		// verify readonly field does not exist in map
		if fl.is(flagReadonly) {
			return r, fmt.Errorf("field '%s' is readonly and not allowed in input", fl.name)
		}
		// verify id field
		if fl.is(flagID) {
			// verify id field does not exist in map, unless persist
			if !fl.is(flagPersist) {
				return r, fmt.Errorf(
					"field '%s' with validation rule 'id' is not allowed in input", k,
				)
			}
			// set ID value
			id, ok := f.makeID(r.Map[k])
			if !ok {
				return r, fmt.Errorf("id field '%s' has invalid type", k)
			}
			r.ID = id
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
func (f *Factory[T, ID]) MakeMapMany(arr []map[string]any, auto ...AutoMap) ([]Doc[T, ID], error) {
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}
	var r []Doc[T, ID]
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
func (f *Factory[T, ID]) MakePartial(data []byte, auto ...AutoMap) (Doc[T, ID], error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Doc[T, ID]{}, fmt.Errorf("json parse to map failed: %w", err)
	}
	return f.MakePartialMap(m, auto...)
}

// MakePartialMany decodes and maps the JSON array to multiple entities and validates them
func (f *Factory[T, ID]) MakePartialMany(data []byte, auto ...AutoMap) ([]Doc[T, ID], error) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("json parse to array failed: %w", err)
	}
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}

	r := make([]Doc[T, ID], len(arr))
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
func (f *Factory[T, ID]) MakePartialMap(m map[string]any, auto ...AutoMap) (Doc[T, ID], error) {
	var r Doc[T, ID]
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
		// set ID value
		id, ok := f.makeID(r.Map[f.entity.id.tag])
		if !ok {
			return r, fmt.Errorf("id field '%s' has invalid type", f.entity.id.tag)
		}
		r.ID = id
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
func (f *Factory[T, ID]) MakePartialMapMany(
	arr []map[string]any,
	auto ...AutoMap,
) ([]Doc[T, ID], error) {
	// check for empty input
	if len(arr) == 0 {
		return nil, errors.New("input is empty")
	}
	var r []Doc[T, ID]
	for i, m := range arr {
		res, err := f.MakePartialMap(m, auto...)
		if err != nil {
			return nil, fmt.Errorf("failed at index %d: %w", i, err)
		}
		r = append(r, res)
	}
	return r, nil
}

// Read reads from the reader, decodes and maps the JSON document to the entity and validates it
func (f *Factory[T, ID]) Read(reader io.Reader, auto ...AutoMap) (Doc[T, ID], error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return Doc[T, ID]{}, fmt.Errorf("%w: %v", ErrRead, err)
	}
	return f.Make(b, auto...)
}

// ReadMany reads from the reader, decodes and maps the JSON array to multiple entities
// and validates them
func (f *Factory[T, ID]) ReadMany(reader io.Reader, auto ...AutoMap) ([]Doc[T, ID], error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRead, err)
	}
	return f.MakeMany(b, auto...)
}

// ReadPartial reads from the reader, decodes and maps the JSON document to the entity
// and validates it
func (f *Factory[T, ID]) ReadPartial(reader io.Reader, auto ...AutoMap) (Doc[T, ID], error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return Doc[T, ID]{}, fmt.Errorf("%w: %v", ErrRead, err)
	}
	return f.MakePartial(b, auto...)
}

// ReadPartialMany reads from the reader, decodes and maps the JSON array to multiple entities
// and validates them
func (f *Factory[T, ID]) ReadPartialMany(reader io.Reader, auto ...AutoMap) ([]Doc[T, ID], error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRead, err)
	}
	return f.MakePartialMany(b, auto...)
}

// Validator returns the validator for the factory
func (f *Factory[T, ID]) Validator() *validator.Validate {
	return f.entity.validator
}

// WithIDParser sets the ID parser function for the factory
func (f *Factory[T, ID]) WithIDParser(fn func(any) (ID, bool)) *Factory[T, ID] {
	f.parseID = fn
	return f
}

// makeID converts the given value to the ID type using the parser function if provided
// otherwise, it attempts to cast the value to the ID type directly
// returns an error if the conversion fails
func (f *Factory[T, ID]) makeID(v any) (ID, bool) {
	if f.parseID != nil {
		return f.parseID(v)
	}

	id, ok := v.(ID)
	if !ok {
		var zero ID
		return zero, false
	}

	return id, true
}

// StringIDParser returns a parser function that converts a string to the ID type
func StringIDParser[ID ~string]() func(any) (ID, bool) {
	return func(v any) (ID, bool) {
		var zero ID

		s, ok := v.(string)
		if !ok {
			return zero, false
		}

		return ID(s), true
	}
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
