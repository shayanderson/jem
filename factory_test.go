package jem

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
)

type testUser struct {
	ID   string `json:"id"   validate:"id,required,len=5"`
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age"  validate:"required"`
}

func TestNewFactory(t *testing.T) {
	type testStruct struct {
		ID string `json:"id"`
	}
	f := New[testStruct]()
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
	if f.entity == nil {
		t.Fatal("expected non-nil entity")
	}
	if f.entity.name != "testStruct" {
		t.Fatalf("expected entity name 'testStruct', got '%s'", f.entity.name)
	}
}

func TestFactory_Make(t *testing.T) {
	f := New[testUser]()
	r, err := f.Make([]byte(`{"name":"Alice","age":30}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "" {
		t.Fatalf("expected ID to be empty, got '%s'", r.Value.ID)
	}
	if r.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", r.Value.Name)
	}
	if r.Value.Age != 30 {
		t.Fatalf("expected age 30, got %d", r.Value.Age)
	}
	id, ok := r.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := r.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", r.Map["name"])
	}
	if name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", name)
	}
	age, ok := r.Map["age"].(float64)
	if !ok {
		t.Fatalf("expected age to be a float64, got %T", r.Map["age"])
	}
	if age != 30 {
		t.Fatalf("expected age 30, got %f", age)
	}
}

func TestFactory_Make_Nested(t *testing.T) {
	type testStruct3 struct {
		Value  int      `json:"value"  validate:"required"`
		Values []string `json:"values" validate:"required,len=2,dive"`
	}
	type testStruct2 struct {
		Name string      `json:"name" validate:"required"`
		TS3  testStruct3 `json:"ts3"  validate:"required"`
	}
	type testStruct struct {
		ID   string      `json:"id"   validate:"id,required,len=5"`
		Name string      `json:"name" validate:"required"`
		TS2  testStruct2 `json:"ts2"  validate:"required"`
	}
	f := New[testStruct]()
	r, err := f.Make([]byte(`{
		"name": "test1",
		"ts2": {
			"name": "test2",
			"ts3": {
				"value": 42,
				"values": ["foo", "bar"]
			}
		}
	}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "" {
		t.Fatalf("expected ID to be empty, got '%s'", r.Value.ID)
	}
	if r.Value.Name != "test1" {
		t.Fatalf("expected name 'test1', got '%s'", r.Value.Name)
	}
	if r.Value.TS2.Name != "test2" {
		t.Fatalf("expected name 'test2', got '%s'", r.Value.TS2.Name)
	}
	if r.Value.TS2.TS3.Value != 42 {
		t.Fatalf("expected value 42, got %d", r.Value.TS2.TS3.Value)
	}
	id, ok := r.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := r.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", r.Map["name"])
	}
	if name != "test1" {
		t.Fatalf("expected name 'test1', got '%s'", name)
	}
	ts2, ok := r.Map["ts2"].(map[string]any)
	if !ok {
		t.Fatalf("expected ts2 to be a map, got %T", r.Map["ts2"])
	}
	ts2Name, ok := ts2["name"].(string)
	if !ok {
		t.Fatalf("expected ts2.name to be a string, got %T", ts2["name"])
	}
	if ts2Name != "test2" {
		t.Fatalf("expected ts2.name 'test2', got '%s'", ts2Name)
	}
	ts3, ok := ts2["ts3"].(map[string]any)
	if !ok {
		t.Fatalf("expected ts3 to be a map, got %T", ts2["ts3"])
	}
	ts3Value, ok := ts3["value"].(float64)
	if !ok {
		t.Fatalf("expected ts3.value to be a float64, got %T", ts3["value"])
	}
	if ts3Value != 42 {
		t.Fatalf("expected ts3.value 42, got %f", ts3Value)
	}
	ts3Values, ok := ts3["values"].([]any)
	if !ok {
		t.Fatalf("expected ts3.values to be a []any, got %T", ts3["values"])
	}
	if len(ts3Values) != 2 {
		t.Fatalf("expected ts3.values to have 2 elements, got %d", len(ts3Values))
	}
	if ts3Values[0] != "foo" {
		t.Fatalf("expected ts3.values[0] 'foo', got '%s'", ts3Values[0])
	}
	if ts3Values[1] != "bar" {
		t.Fatalf("expected ts3.values[1] 'bar', got '%s'", ts3Values[1])
	}

	r, err = f.Make([]byte(`{"name": "test1"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.ts2' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err = f.Make([]byte(`{"name": "test1", "ts2": {"name":"test2"}}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'testStruct.ts2.ts3' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err = f.Make([]byte(`{"name": "test1", "ts2": {"name":"test2", "ts3": {"value": 42}}}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'testStruct.ts2.ts3.values' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err = f.Make([]byte(`{
		"name": "test1", "ts2": {"name":"test2", "ts3": {"value": 42, "values": ["one"]}}
	}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'testStruct.ts2.ts3.values' validation failed for rule 'len'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_Make_Any(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id,required"`
		Name   string `json:"name"   validate:"required"`
		Config any    `json:"config"`
	}
	f := New[testStruct]()
	_, err := f.Make([]byte(`{"name":"test1"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = f.Make([]byte(`{"name":"test1","config":{"key":"value"}}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFactory_Make_InvalidJSON(t *testing.T) {
	f := New[testUser]()
	r, err := f.Make([]byte(`{"name":"Alice","age":30,}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "json parse to map failed: invalid character '}' looking for beginning of object" +
		" key string"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakeMany(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMany([]byte(`[
		{"name":"Alice","age":30},
		{"name":"Bob","age":25}
	]`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(r) != 2 {
		t.Fatalf("expected 2 results, got %d", len(r))
	}

	a := r[0]
	if a.Value == nil {
		t.Fatal("expected non-nil Doc for Alice")
	}
	if a.Value.ID != "" {
		t.Fatalf("expected ID to be empty for Alice, got '%s'", a.Value.ID)
	}
	if a.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", a.Value.Name)
	}
	if a.Value.Age != 30 {
		t.Fatalf("expected age 30 for Alice, got %d", a.Value.Age)
	}
	aID, ok := a.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map for Alice, got '%s'", aID)
	}
	aName, ok := a.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string for Alice, got %T", a.Map["name"])
	}
	if aName != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", aName)
	}
	aAge, ok := a.Map["age"].(float64)
	if !ok {
		t.Fatalf("expected age to be a float64 for Alice, got %T", a.Map["age"])
	}
	if aAge != 30 {
		t.Fatalf("expected age 30 for Alice, got %f", aAge)
	}

	b := r[1]
	if b.Value == nil {
		t.Fatal("expected non-nil Doc for Bob")
	}
	if b.Value.ID != "" {
		t.Fatalf("expected ID to be empty for Bob, got '%s'", b.Value.ID)
	}
	if b.Value.Name != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", b.Value.Name)
	}
	if b.Value.Age != 25 {
		t.Fatalf("expected age 25 for Bob, got %d", b.Value.Age)
	}
	bID, ok := b.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map for Bob, got '%s'", bID)
	}
	bName, ok := b.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string for Bob, got %T", b.Map["name"])
	}
	if bName != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", bName)
	}
	bAge, ok := b.Map["age"].(float64)
	if !ok {
		t.Fatalf("expected age to be a float64 for Bob, got %T", b.Map["age"])
	}
	if bAge != 25 {
		t.Fatalf("expected age 25 for Bob, got %f", bAge)
	}
}

func TestFactory_MakeMany_InvalidJSON(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMany([]byte(`[
		{"name":"Alice","age":30},
		{"name":"Bob","age":25,}
	]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "json parse to array failed: invalid character '}' looking for beginning of object" +
		" key string"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r != nil {
		t.Fatal("expected nil result, got non-nil")
	}
}

func TestFactory_MakeMany_Empty(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMany([]byte(`[]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input is empty"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r != nil {
		t.Fatal("expected nil result, got non-nil")
	}
}

func TestFactory_MakeMany_WithIndexError(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMany([]byte(`[
		{"name":"Alice","age":30},
		{"name":"Bob"}
	]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "failed at index 1: field 'testUser.age' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r != nil {
		t.Fatal("expected nil result, got non-nil")
	}
}

func TestFactory_MakeMap(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
		"age":  30,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "" {
		t.Fatalf("expected ID to be empty, got '%s'", r.Value.ID)
	}
	if r.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", r.Value.Name)
	}
	if r.Value.Age != 30 {
		t.Fatalf("expected age 30, got %d", r.Value.Age)
	}
	id, ok := r.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := r.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", r.Map["name"])
	}
	if name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", name)
	}
	age, ok := r.Map["age"].(int)
	if !ok {
		t.Fatalf("expected age to be an int, got %T", r.Map["age"])
	}
	if age != 30 {
		t.Fatalf("expected age 30, got %d", age)
	}
}

func TestFactory_MakeMap_ZeroFields(t *testing.T) {
	type testStruct struct{}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "struct 'testStruct' has zero fields with json tags"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
	if r.Map != nil {
		t.Fatal("expected nil Map, got non-nil")
	}
}

func TestFactory_MakeMap_Empty(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMap(map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input is empty"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakeMap_UnknownField(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
		"age":  30,
		"foo":  "bar",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "unknown field 'foo'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakeMap_UnknownFieldNested(t *testing.T) {
	type testNested struct {
		Field1 string `json:"field1" validate:"required"`
	}
	type testStruct struct {
		ID     string     `json:"id"     validate:"id"`
		Nested testNested `json:"nested" validate:"required"`
	}
	f := New[testStruct]()
	_, err := f.MakeMap(map[string]any{
		"nested": map[string]any{
			"field1": "value1",
			"foo":    "bar",
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "json parse failed: json: unknown field \"foo\""
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakeMap_Validate(t *testing.T) {
	f := New[testUser]()
	_, err := f.MakeMap(map[string]any{
		"name": "Alice",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testUser.age' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakeMap_ValidateCustomInvalid(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id,required"`
		Custom string `json:"custom" validate:"custom,required"`
	}
	defer func() {
		wantErr := "Undefined validation function 'custom' on field 'Custom'"
		if r := recover(); r != wantErr {
			t.Fatalf("expected panic %q, got: %v", wantErr, r)
		}
	}()
	f := New[testStruct]()
	_, _ = f.MakeMap(map[string]any{
		"custom": "custom value",
	})
}

func TestFactory_MakeMap_ValidateCustom(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id,required"`
		Custom string `json:"custom" validate:"custom,required"`
	}
	f := New[testStruct]()
	f.Validator().RegisterValidation("custom", func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "test"
	})
	_, err := f.MakeMap(map[string]any{
		"custom": "test2",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.custom' validation failed for rule 'custom'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err := f.MakeMap(map[string]any{
		"custom": "test",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.Custom != "test" {
		t.Fatalf("expected Custom to be 'test', got '%s'", r.Value.Custom)
	}
	v, ok := r.Map["custom"].(string)
	if !ok {
		t.Fatalf("expected custom to be a string, got %T", r.Map["custom"])
	}
	if v != "test" {
		t.Fatalf("expected custom 'test', got '%s'", v)
	}
}

func TestFactory_MakeMap_WithID(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"age":  30,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'id' with validation rule 'id' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakeMap_WithIDPersist(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id,persist,required,len=5"`
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"  validate:"required"`
	}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"age":  30,
	})
	if err != nil {
		t.Fatal("expected error, got nil")
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "u-123" {
		t.Fatalf("expected ID to be 'u-123', got '%s'", r.Value.ID)
	}
	if r.Map == nil {
		t.Fatal("expected non-nil Map")
	}
	id, ok := r.Map["id"].(string)
	if !ok {
		t.Fatalf("expected id to be a string, got %T", r.Map["id"])
	}
	if id != "u-123" {
		t.Fatalf("expected id 'u-123', got '%s'", id)
	}
	_, err = f.MakeMap(map[string]any{
		"name": "Alice",
		"age":  30,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.id' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakeMap_WithAuto(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto"`
	}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
		"auto": "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Auto' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakeMap(map[string]any{
		"name": "Alice",
	}, AutoMap{
		"auto": func() any {
			return "test"
		},
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.Auto != "test" {
		t.Fatalf("expected Auto to be 'test', got '%s'", r.Value.Auto)
	}
	v, ok := r.Map["auto"].(string)
	if !ok {
		t.Fatalf("expected auto to be a string, got %T", r.Map["auto"])
	}
	if v != "test" {
		t.Fatalf("expected auto 'test', got '%s'", v)
	}
}

func TestFactory_MakeMap_WithAutoFull(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto:full,required"`
	}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
		"auto": "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Auto' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakeMap(map[string]any{
		"name": "Alice",
	}, AutoMap{
		"auto": func() any {
			return "test"
		},
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.Auto != "test" {
		t.Fatalf("expected Auto to be 'test', got '%s'", r.Value.Auto)
	}
	v, ok := r.Map["auto"].(string)
	if !ok {
		t.Fatalf("expected auto to be a string, got %T", r.Map["auto"])
	}
	if v != "test" {
		t.Fatalf("expected auto 'test', got '%s'", v)
	}
}

func TestFactory_MakeMap_WithAutoPartial(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto:partial"`
	}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
		"auto": "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Auto' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakeMap(map[string]any{
		"name": "Alice",
	}, AutoMap{
		"auto": func() any {
			return "test"
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'auto' is not an 'auto' or 'auto:full' field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakeMap_WithAutoInvalid(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto"`
	}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
	}, AutoMap{
		"foo": func() any {
			return "bar"
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'foo' is not an 'auto' or 'auto:full' field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakeMap_WithPersist(t *testing.T) {
	type testStruct struct {
		ID      string `json:"id"      validate:"id"`
		Name    string `json:"name"    validate:"required"`
		Persist string `json:"persist" validate:"persist"`
	}
	f := New[testStruct]()
	r, err := f.MakeMap(map[string]any{
		"name": "Alice",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	r, err = f.MakeMap(map[string]any{
		"name":    "Alice",
		"persist": "test",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.Persist != "test" {
		t.Fatalf("expected Persist to be 'test', got '%s'", r.Value.Persist)
	}
	v, ok := r.Map["persist"].(string)
	if !ok {
		t.Fatalf("expected persist to be a string, got %T", r.Map["persist"])
	}
	if v != "test" {
		t.Fatalf("expected persist 'test', got '%s'", v)
	}

	type testStruct2 struct {
		ID      string `json:"id"      validate:"id"`
		Name    string `json:"name"    validate:"required"`
		Persist string `json:"persist" validate:"persist,required"`
	}
	f2 := New[testStruct2]()
	_, err = f2.MakeMap(map[string]any{
		"name": "Alice",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct2.persist' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakeMap_WithReadonly(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id"`
		Name   string `json:"name"   validate:"required"`
		ROTest string `json:"roTest" validate:"readonly"`
	}
	f := New[testStruct]()
	_, err := f.MakeMap(map[string]any{
		"name": "Alice",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	_, err = f.MakeMap(map[string]any{
		"name":   "Alice",
		"roTest": "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.ROTest' is readonly and not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakeMapMany(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMapMany([]map[string]any{
		{
			"name": "Alice",
			"age":  30,
		},
		{
			"name": "Bob",
			"age":  25,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(r) != 2 {
		t.Fatalf("expected 2 results, got %d", len(r))
	}

	a := r[0]
	if a.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if a.Value.ID != "" {
		t.Fatalf("expected ID to be empty, got '%s'", a.Value.ID)
	}
	if a.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", a.Value.Name)
	}
	if a.Value.Age != 30 {
		t.Fatalf("expected age 30, got %d", a.Value.Age)
	}
	id, ok := a.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := a.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", a.Map["name"])
	}
	if name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", name)
	}
	age, ok := a.Map["age"].(int)
	if !ok {
		t.Fatalf("expected age to be an int, got %T", a.Map["age"])
	}
	if age != 30 {
		t.Fatalf("expected age 30, got %d", age)
	}

	b := r[1]
	if b.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if b.Value.ID != "" {
		t.Fatalf("expected ID to be empty, got '%s'", b.Value.ID)
	}
	if b.Value.Name != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", b.Value.Name)
	}
	if b.Value.Age != 25 {
		t.Fatalf("expected age 25 for Bob, got %d", b.Value.Age)
	}
	bID, ok := b.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map for Bob, got '%s'", bID)
	}
	bName, ok := b.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string for Bob, got %T", b.Map["name"])
	}
	if bName != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", bName)
	}
	bAge, ok := b.Map["age"].(int)
	if !ok {
		t.Fatalf("expected age to be an int for Bob, got %T", b.Map["age"])
	}
	if bAge != 25 {
		t.Fatalf("expected age 25 for Bob, got %d", bAge)
	}
}

func TestFactory_MakeMapMany_Empty(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMapMany([]map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input is empty"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if len(r) != 0 {
		t.Fatalf("expected 0 results, got %d", len(r))
	}
}

func TestFactory_MakeMapMany_WithIndexError(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakeMapMany([]map[string]any{
		{
			"name": "Alice",
			"age":  30,
		},
		{
			"name": "",
			"age":  25,
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "failed at index 1: field 'testUser.name' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if len(r) != 0 {
		t.Fatalf("expected 0 results, got %d", len(r))
	}
}

func TestFactory_MakePartial(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartial([]byte(`{"id":"u-123","name":"Alice","age":30}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "u-123" {
		t.Fatalf("expected ID 'u-123', got '%s'", r.Value.ID)
	}
	if r.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", r.Value.Name)
	}
	if r.Value.Age != 30 {
		t.Fatalf("expected age 30, got %d", r.Value.Age)
	}
	id, ok := r.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := r.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", r.Map["name"])
	}
	if name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", name)
	}
	age, ok := r.Map["age"].(float64)
	if !ok {
		t.Fatalf("expected age to be a float64, got %T", r.Map["age"])
	}
	if age != 30 {
		t.Fatalf("expected age 30, got %f", age)
	}
}

func TestFactory_MakePartial_Nested(t *testing.T) {
	type testStruct3 struct {
		Value  int      `json:"value"  validate:"required"`
		Values []string `json:"values" validate:"required,len=2,dive"`
	}
	type testStruct2 struct {
		Name string      `json:"name" validate:"required"`
		TS3  testStruct3 `json:"ts3"  validate:"required"`
	}
	type testStruct struct {
		ID   string      `json:"id"   validate:"id,required,len=5"`
		Name string      `json:"name" validate:"required"`
		TS2  testStruct2 `json:"ts2"  validate:"required"`
	}
	f := New[testStruct]()
	r, err := f.MakePartial([]byte(`{
		"id": "test1",
		"name": "test1",
		"ts2": {
			"name": "test2",
			"ts3": {
				"value": 42,
				"values": ["foo", "bar"]
			}
		}
	}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "test1" {
		t.Fatalf("expected ID to be 'test1', got '%s'", r.Value.ID)
	}
	if r.Value.Name != "test1" {
		t.Fatalf("expected name 'test1', got '%s'", r.Value.Name)
	}
	if r.Value.TS2.Name != "test2" {
		t.Fatalf("expected name 'test2', got '%s'", r.Value.TS2.Name)
	}
	if r.Value.TS2.TS3.Value != 42 {
		t.Fatalf("expected value 42, got %d", r.Value.TS2.TS3.Value)
	}
	id, ok := r.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := r.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", r.Map["name"])
	}
	if name != "test1" {
		t.Fatalf("expected name 'test1', got '%s'", name)
	}
	ts2, ok := r.Map["ts2"].(map[string]any)
	if !ok {
		t.Fatalf("expected ts2 to be a map, got %T", r.Map["ts2"])
	}
	ts2Name, ok := ts2["name"].(string)
	if !ok {
		t.Fatalf("expected ts2.name to be a string, got %T", ts2["name"])
	}
	if ts2Name != "test2" {
		t.Fatalf("expected ts2.name 'test2', got '%s'", ts2Name)
	}
	ts3, ok := ts2["ts3"].(map[string]any)
	if !ok {
		t.Fatalf("expected ts3 to be a map, got %T", ts2["ts3"])
	}
	ts3Value, ok := ts3["value"].(float64)
	if !ok {
		t.Fatalf("expected ts3.value to be a float64, got %T", ts3["value"])
	}
	if ts3Value != 42 {
		t.Fatalf("expected ts3.value 42, got %f", ts3Value)
	}
	ts3Values, ok := ts3["values"].([]any)
	if !ok {
		t.Fatalf("expected ts3.values to be a []any, got %T", ts3["values"])
	}
	if len(ts3Values) != 2 {
		t.Fatalf("expected ts3.values to have 2 elements, got %d", len(ts3Values))
	}
	if ts3Values[0] != "foo" {
		t.Fatalf("expected ts3.values[0] 'foo', got '%s'", ts3Values[0])
	}
	if ts3Values[1] != "bar" {
		t.Fatalf("expected ts3.values[1] 'bar', got '%s'", ts3Values[1])
	}

	r, err = f.MakePartial([]byte(`{"id": "test1", "ts2": {"name":"test2"}}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.ts2.ts3' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err = f.MakePartial([]byte(`{
		"id": "test1", "ts2": {"name":"test2", "ts3": {"value": 42}}
	}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'testStruct.ts2.ts3.values' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err = f.MakePartial([]byte(`{
		"id": "test1", "ts2": {"name":"test2", "ts3": {"value": 42, "values": ["one"]}}
	}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'testStruct.ts2.ts3.values' validation failed for rule 'len'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartial_Any(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id,required"`
		Name   string `json:"name"   validate:"required"`
		Config any    `json:"config"`
	}
	f := New[testStruct]()
	_, err := f.MakePartial([]byte(`{"id":"test1","name":"test2"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = f.MakePartial([]byte(`{"id":"test1","config":{"key":"value"}}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFactory_MakePartial_InvalidJSON(t *testing.T) {
	f := New[testUser]()
	_, err := f.MakePartial([]byte(`{"id":"u-123","name":"Alice","age":30,}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "json parse to map failed: invalid character '}' looking for beginning of object" +
		" key string"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMany(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMany([]byte(`[
		{"id":"u-123","name":"Alice","age":30},
		{"id":"u-456","name":"Bob","age":25}
	]`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(r) != 2 {
		t.Fatalf("expected 2 results, got %d", len(r))
	}

	a := r[0]
	if a.Value == nil {
		t.Fatal("expected non-nil Doc for Alice")
	}
	if a.Value.ID != "u-123" {
		t.Fatalf("expected ID 'u-123', got '%s'", a.Value.ID)
	}
	if a.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", a.Value.Name)
	}
	if a.Value.Age != 30 {
		t.Fatalf("expected age 30, got %d", a.Value.Age)
	}
	aID, ok := a.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map for Alice, got '%s'", aID)
	}
	aName, ok := a.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string for Alice, got %T", a.Map["name"])
	}
	if aName != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", aName)
	}
	aAge, ok := a.Map["age"].(float64)
	if !ok {
		t.Fatalf("expected age to be a float64 for Alice, got %T", a.Map["age"])
	}
	if aAge != 30 {
		t.Fatalf("expected age 30 for Alice, got %f", aAge)
	}

	b := r[1]
	if b.Value == nil {
		t.Fatal("expected non-nil Doc for Bob")
	}
	if b.Value.ID != "u-456" {
		t.Fatalf("expected ID 'u-456', got '%s'", b.Value.ID)
	}
	if b.Value.Name != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", b.Value.Name)
	}
	if b.Value.Age != 25 {
		t.Fatalf("expected age 25 for Bob, got %d", b.Value.Age)
	}
	bID, ok := b.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map for Bob, got '%s'", bID)
	}
	bName, ok := b.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string for Bob, got %T", b.Map["name"])
	}
	if bName != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", bName)
	}
	bAge, ok := b.Map["age"].(float64)
	if !ok {
		t.Fatalf("expected age to be a float64 for Bob, got %T", b.Map["age"])
	}
	if bAge != 25 {
		t.Fatalf("expected age 25 for Bob, got %f", bAge)
	}
}

func TestFactory_MakePartialMany_InvalidJSON(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMany([]byte(`[
		{"id":"u-123","name":"Alice","age":30},
		{"id":"u-456","name":"Bob","age":25,}
	]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "json parse to array failed: invalid character '}' looking for beginning of object" +
		" key string"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r != nil {
		t.Fatal("expected nil result, got non-nil")
	}
}

func TestFactory_MakePartialMany_Empty(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMany([]byte(`[]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input is empty"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r != nil {
		t.Fatal("expected nil result, got non-nil")
	}
}

func TestFactory_MakePartialMany_WithIndexError(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMany([]byte(`[
		{"id":"u-123","name":"Alice","age":30},
		{"id":"u-456","name":"","age":25}
	]`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "failed at index 1: field 'testUser.name' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r != nil {
		t.Fatal("expected nil result, got non-nil")
	}
}

func TestFactory_MakePartialMap(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"age":  30,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "u-123" {
		t.Fatalf("expected ID 'u-123', got '%s'", r.Value.ID)
	}
	if r.Value.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", r.Value.Name)
	}
	if r.Value.Age != 30 {
		t.Fatalf("expected age 30, got %d", r.Value.Age)
	}
	id, ok := r.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", id)
	}
	name, ok := r.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", r.Map["name"])
	}
	if name != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", name)
	}
	age, ok := r.Map["age"].(int)
	if !ok {
		t.Fatalf("expected age to be an int, got %T", r.Map["age"])
	}
	if age != 30 {
		t.Fatalf("expected age 30, got %d", age)
	}

	f = New[testUser]()
	_, err = f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFactory_MakePartialMap_ZeroFields(t *testing.T) {
	type testStruct struct{}
	f := New[testStruct]()
	r, err := f.MakePartialMap(map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "struct 'testStruct' has zero fields with json tags"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
	if r.Map != nil {
		t.Fatal("expected nil Map, got non-nil")
	}
}

func TestFactory_MakePartialMap_Empty(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMap(map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input is empty"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakePartialMap_MissingID(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMap(map[string]any{
		"name": "Alice",
		"age":  30,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input must have id field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakePartialMap_WithIDPersist(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id,persist,required,len=5"`
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"  validate:"required"`
	}
	f := New[testStruct]()
	_, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"age":  30,
	})
	if err != nil {
		t.Fatal("expected error, got nil")
	}
	_, err = f.MakePartialMap(map[string]any{
		"name": "Alice",
		"age":  30,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input must have id field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMap_OnlyID(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMap(map[string]any{
		"id": "u-123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input must have id field and at least one other field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakePartialMap_UnknownField(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMap(map[string]any{
		"id":  "u-123",
		"foo": "bar",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "unknown field 'foo'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakePartialMap_UnknownFieldNested(t *testing.T) {
	type testNested struct {
		Field1 string `json:"field1" validate:"required"`
	}
	type testStruct struct {
		ID     string     `json:"id"     validate:"id"`
		Nested testNested `json:"nested" validate:"required"`
	}
	f := New[testStruct]()
	_, err := f.MakePartialMap(map[string]any{
		"id": "u-123",
		"nested": map[string]any{
			"field1": "value1",
			"foo":    "bar",
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "json parse failed: json: unknown field \"foo\""
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMap_WithAuto(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto"`
	}
	f := New[testStruct]()
	r, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"auto": "value",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Auto' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice2",
	}, AutoMap{
		"auto": func() any {
			return "test"
		},
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "u-123" {
		t.Fatalf("expected ID 'u-123', got '%s'", r.Value.ID)
	}
	if r.Value.Auto != "test" {
		t.Fatalf("expected Auto 'test', got '%s'", r.Value.Auto)
	}
	v, ok := r.Map["auto"].(string)
	if !ok {
		t.Fatalf("expected auto to be a string, got %T", r.Map["auto"])
	}
	if v != "test" {
		t.Fatalf("expected auto 'test', got '%s'", v)
	}

	r, err = f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice2",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
}

func TestFactory_MakePartialMap_WithAutoFull(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto:full"`
	}
	f := New[testStruct]()
	r, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"auto": "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Auto' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
	}, AutoMap{
		"auto": func() any {
			return "test"
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'auto' is not an 'auto' or 'auto:partial' field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMap_WithAutoPartial(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto:partial"`
	}
	f := New[testStruct]()
	r, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
		"auto": "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Auto' is not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
	}, AutoMap{
		"auto": func() any {
			return "test"
		},
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.Auto != "test" {
		t.Fatalf("expected Auto 'test', got '%s'", r.Value.Auto)
	}
	v, ok := r.Map["auto"].(string)
	if !ok {
		t.Fatalf("expected auto to be a string, got %T", r.Map["auto"])
	}
	if v != "test" {
		t.Fatalf("expected auto 'test', got '%s'", v)
	}
}

func TestFactory_MakePartialMap_WithAutoInvalid(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required"`
		Auto string `json:"auto" validate:"auto"`
	}
	f := New[testStruct]()
	r, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
	}, AutoMap{
		"foo": func() any {
			return "bar"
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'foo' is not an 'auto' or 'auto:partial' field"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}
}

func TestFactory_MakePartialMap_WithPersist(t *testing.T) {
	type testStruct struct {
		ID          string `json:"id"          validate:"id"`
		Name        string `json:"name"        validate:"required"`
		PersistTest string `json:"persistTest" validate:"persist"`
	}
	f := New[testStruct]()
	r, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'persistTest' with validation rule 'persist' is missing"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if r.Value != nil {
		t.Fatal("expected nil Doc, got non-nil")
	}

	r, err = f.MakePartialMap(map[string]any{
		"id":          "u-123",
		"name":        "Alice2",
		"persistTest": "test",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.ID != "u-123" {
		t.Fatalf("expected ID 'u-123', got '%s'", r.Value.ID)
	}
	if r.Value.PersistTest != "test" {
		t.Fatalf("expected PersistTest 'test', got '%s'", r.Value.PersistTest)
	}
	v, ok := r.Map["persistTest"].(string)
	if !ok {
		t.Fatalf("expected persistTest to be a string, got %T", r.Map["persistTest"])
	}
	if v != "test" {
		t.Fatalf("expected persistTest 'test', got '%s'", v)
	}

	type testStruct2 struct {
		ID          string `json:"id"          validate:"id"`
		Name        string `json:"name"        validate:"required"`
		PersistTest string `json:"persistTest" validate:"persist,required"`
	}
	f2 := New[testStruct2]()
	_, err = f2.MakePartialMap(map[string]any{
		"id":          "u-123",
		"name":        "Alice2",
		"persistTest": "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr = "field 'testStruct2.persistTest' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMap_WithReadonly(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id"`
		Name   string `json:"name"   validate:"required"`
		ROTest string `json:"roTest" validate:"readonly"`
	}
	f := New[testStruct]()
	_, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "Alice",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	_, err = f.MakePartialMap(map[string]any{
		"id":     "u-123",
		"name":   "Alice2",
		"roTest": "value",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.ROTest' is readonly and not allowed in input"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMap_Validate(t *testing.T) {
	f := New[testUser]()
	_, err := f.MakePartialMap(map[string]any{
		"id":   "u-123",
		"name": "",
		"age":  30,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testUser.name' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
}

func TestFactory_MakePartialMap_ValidateCustomInvalid(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id,required"`
		Custom string `json:"custom" validate:"custom,required"`
	}
	defer func() {
		wantErr := "Undefined validation function 'custom' on field 'Custom'"
		if r := recover(); r != wantErr {
			t.Fatalf("expected panic %q, got: %v", wantErr, r)
		}
	}()
	f := New[testStruct]()
	_, _ = f.MakePartialMap(map[string]any{
		"id":     "u-123",
		"custom": "custom value",
	})
}

func TestFactory_MakePartialMap_ValidateCustom(t *testing.T) {
	type testStruct struct {
		ID     string `json:"id"     validate:"id,required"`
		Custom string `json:"custom" validate:"custom,required"`
	}
	f := New[testStruct]()
	f.Validator().RegisterValidation("custom", func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "test"
	})
	_, err := f.MakePartialMap(map[string]any{
		"id":     "u-123",
		"custom": "test2",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.custom' validation failed for rule 'custom'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}

	r, err := f.MakePartialMap(map[string]any{
		"id":     "u-123",
		"custom": "test",
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if r.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if r.Value.Custom != "test" {
		t.Fatalf("expected Custom to be 'test', got '%s'", r.Value.Custom)
	}
	v, ok := r.Map["custom"].(string)
	if !ok {
		t.Fatalf("expected custom to be a string, got %T", r.Map["custom"])
	}
	if v != "test" {
		t.Fatalf("expected custom 'test', got '%s'", v)
	}
}

func TestFactory_MakePartialMapMany(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMapMany([]map[string]any{
		{
			"id":   "u-123",
			"name": "Alice",
			"age":  30,
		},
		{
			"id":   "u-124",
			"name": "Bob",
			"age":  25,
		},
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if len(r) != 2 {
		t.Fatalf("expected 2 results, got %d", len(r))
	}

	a := r[0]
	if a.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if a.Value.ID != "u-123" {
		t.Fatalf("expected ID 'u-123', got '%s'", a.Value.ID)
	}
	if a.Value.Name != "Alice" {
		t.Fatalf("expected Name 'Alice', got '%s'", a.Value.Name)
	}
	if a.Value.Age != 30 {
		t.Fatalf("expected Age 30, got %d", a.Value.Age)
	}
	aID, ok := a.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", aID)
	}
	aName, ok := a.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", a.Map["name"])
	}
	if aName != "Alice" {
		t.Fatalf("expected name 'Alice', got '%s'", aName)
	}
	aAge, ok := a.Map["age"].(int)
	if !ok {
		t.Fatalf("expected age to be an int, got %T", a.Map["age"])
	}
	if aAge != 30 {
		t.Fatalf("expected age 30, got %d", aAge)
	}

	b := r[1]
	if b.Value == nil {
		t.Fatal("expected non-nil Doc")
	}
	if b.Value.ID != "u-124" {
		t.Fatalf("expected ID 'u-124', got '%s'", b.Value.ID)
	}
	if b.Value.Name != "Bob" {
		t.Fatalf("expected Name 'Bob', got '%s'", b.Value.Name)
	}
	if b.Value.Age != 25 {
		t.Fatalf("expected Age 25, got %d", b.Value.Age)
	}
	bID, ok := b.Map["id"].(string)
	if ok {
		t.Fatalf("expected id to be missing in map, got '%s'", bID)
	}
	bName, ok := b.Map["name"].(string)
	if !ok {
		t.Fatalf("expected name to be a string, got %T", b.Map["name"])
	}
	if bName != "Bob" {
		t.Fatalf("expected name 'Bob', got '%s'", bName)
	}
	bAge, ok := b.Map["age"].(int)
	if !ok {
		t.Fatalf("expected age to be an int, got %T", b.Map["age"])
	}
	if bAge != 25 {
		t.Fatalf("expected age 25, got %d", bAge)
	}
}

func TestFactory_MakePartialMapMany_Empty(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMapMany([]map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "input is empty"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if len(r) != 0 {
		t.Fatalf("expected 0 results, got %d", len(r))
	}
}

func TestFactory_MakePartialMapMany_WithIndexError(t *testing.T) {
	f := New[testUser]()
	r, err := f.MakePartialMapMany([]map[string]any{
		{
			"id":   "u-123",
			"name": "Alice",
			"age":  30,
		},
		{
			"id":   "u-124",
			"name": "",
			"age":  25,
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "failed at index 1: field 'testUser.name' validation failed for rule 'required'"
	if err.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, err)
	}
	if len(r) != 0 {
		t.Fatalf("expected 0 results, got %d", len(r))
	}
}

func Test_validationErrorHandler(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"required"`
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"  validate:"required,min=0"`
	}
	v := validator.New()
	user := testStruct{
		ID:   "u-123",
		Name: "",
		Age:  30,
	}
	err := v.Struct(user)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantErr := "field 'testStruct.Name' validation failed for rule 'required'"
	ve := validationErrorHandler(err)
	if ve.Error() != wantErr {
		t.Fatalf("expected error message %q, got: %q", wantErr, ve)
	}

	err = errors.New("test error")
	ve = validationErrorHandler(err)
	if ve.Error() != "test error" {
		t.Fatalf("expected error message %q, got: %q", "test error", ve)
	}
}

func BenchmarkBaselineJSON(b *testing.B) {
	type testStructs struct {
		ID   string `json:"id"   validate:"required,len=5"`
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"  validate:"required"`
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	fn := func(data []byte) (any, error) {
		var v testStructs
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("json unmarshal failed: %w", err)
		}
		if err := validate.Struct(v); err != nil {
			return nil, fmt.Errorf("struct validation failed: %w", err)
		}
		return v, nil
	}

	b.ReportAllocs()
	for b.Loop() {
		_, err := fn([]byte(`{"id":"test1","name":"test1","age":30}`))
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkBaselineJSONMany(b *testing.B) {
	type testStruct struct {
		ID   string `json:"id"   validate:"required,len=5"`
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"  validate:"required"`
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	fn := func(data []byte) (any, error) {
		var v []testStruct
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("json unmarshal failed: %w", err)
		}
		for _, item := range v {
			if err := validate.Struct(item); err != nil {
				return nil, fmt.Errorf("struct validation failed: %w", err)
			}
		}
		return v, nil
	}

	b.ReportAllocs()
	for b.Loop() {
		_, err := fn([]byte(`[
			{"id":"test1","name":"test1","age":30},
			{"id":"test2","name":"test2","age":25}
		]`))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_Make(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.Make([]byte(`{"name":"test1","age":30}`))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakeMany(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakeMany([]byte(`[
			{"name":"test1","age":30},
			{"name":"test2","age":25}
		]`))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakeMap(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakeMap(map[string]any{"name": "test1", "age": 30})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakeMapMany(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakeMapMany([]map[string]any{
			{"name": "test1", "age": 30},
			{"name": "test2", "age": 25},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakePartial(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakePartial([]byte(`{"id":"test1","name":"test1","age":30}`))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakePartialMany(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakePartialMany([]byte(`[
			{"id":"test1","name":"test1","age":30},
			{"id":"test2","name":"test2","age":30}
		]`))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakePartialMap(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakePartialMap(map[string]any{"id": "test1", "name": "test1", "age": 30})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFactory_MakePartialMapMany(b *testing.B) {
	f := New[testUser]()
	b.ReportAllocs()
	for b.Loop() {
		_, err := f.MakePartialMapMany([]map[string]any{
			{"id": "test1", "name": "test1", "age": 30},
			{"id": "test2", "name": "test2", "age": 30},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
