package jem

import (
	"reflect"
	"testing"
)

func TestEntity(t *testing.T) {
	e := newEntity(testUser{})
	if e == nil {
		t.Fatal("failed to create entity")
	}
	if e.name != "testUser" {
		t.Fatalf("expected entity name 'testUser', got '%s'", e.name)
	}
	if e.validator == nil {
		t.Fatal("expected non-nil validator")
	}
	if e.id == nil {
		t.Fatal("expected non-nil ID field")
	}
	if e.id.tag != "id" {
		t.Fatalf("expected id tag 'id', got '%s'", e.id.tag)
	}
	if len(e.fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(e.fields))
	}
}

func TestNewEntity_InvalidType(t *testing.T) {
	defer func() {
		wantErr := "entity must be a struct"
		if r := recover(); r != wantErr {
			t.Fatal("expected panic with", wantErr)
		}
	}()
	newEntity("invalid")
}

func TestNewEntity_IgnoreField(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id" validate:"id"`
		Name string `json:"-"`
		Age  int    `json:""`
	}
	e := newEntity(testStruct{})
	if len(e.fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(e.fields))
	}
}

func TestNewEntity_MultipleID(t *testing.T) {
	type testStruct struct {
		ID  string `json:"id"  validate:"id"`
		ID2 string `json:"id2" validate:"id"`
	}
	defer func() {
		wantErr := "struct 'testStruct' has multiple fields with 'id' validation rule"
		if r := recover(); r != wantErr {
			t.Fatal("expected panic with", wantErr)
		}
	}()
	newEntity(testStruct{})
}

func TestEntity_FieldFlags(t *testing.T) {
	type testStruct struct {
		ID          string `json:"id"          validate:"id"`
		Auto        string `json:"auto"        validate:"auto"`
		AutoFull    string `json:"autoFull"    validate:"auto:full"`
		AutoPartial string `json:"autoPartial" validate:"auto:partial"`
		Persist     string `json:"persist"     validate:"persist"`
		None        string `json:"none"`
	}
	e := newEntity(testStruct{})
	if len(e.fields) != 6 {
		t.Fatalf("expected 6 fields, got %d", len(e.fields))
	}
	for _, f := range e.fields {
		switch f.name {
		case "testStruct.ID":
			if !f.is(flagID) {
				t.Fatalf("expected field '%s' to have ID flag", f.name)
			}
			if f.is(flagAuto) || f.is(flagAutoFull) || f.is(flagAutoPartial) || f.is(flagPersist) {
				t.Fatalf("expected field '%s' to only have ID flag", f.name)
			}
		case "testStruct.Auto":
			if !f.is(flagAuto) {
				t.Fatalf("expected field '%s' to have Auto flag", f.name)
			}
			if f.is(flagID) || f.is(flagAutoFull) || f.is(flagAutoPartial) || f.is(flagPersist) {
				t.Fatalf("expected field '%s' to only have Auto flag", f.name)
			}
		case "testStruct.AutoFull":
			if !f.is(flagAutoFull) {
				t.Fatalf("expected field '%s' to have AutoFull flag", f.name)
			}
			if f.is(flagID) || f.is(flagAuto) || f.is(flagAutoPartial) || f.is(flagPersist) {
				t.Fatalf("expected field '%s' to only have AutoFull flag", f.name)
			}
		case "testStruct.AutoPartial":
			if !f.is(flagAutoPartial) {
				t.Fatalf("expected field '%s' to have AutoPartial flag", f.name)
			}
			if f.is(flagID) || f.is(flagAuto) || f.is(flagAutoFull) || f.is(flagPersist) {
				t.Fatalf("expected field '%s' to only have AutoPartial flag", f.name)
			}
		case "testStruct.Persist":
			if !f.is(flagPersist) {
				t.Fatalf("expected field '%s' to have Persist flag", f.name)
			}
			if f.is(flagID) || f.is(flagAuto) || f.is(flagAutoFull) || f.is(flagAutoPartial) {
				t.Fatalf("expected field '%s' to only have Persist flag", f.name)
			}
		case "testStruct.None":
			if f.is(flagID) || f.is(flagAuto) || f.is(flagAutoFull) || f.is(flagAutoPartial) ||
				f.is(flagPersist) {
				t.Fatalf("expected field '%s' to have no flags", f.name)
			}
		default:
			t.Fatalf("unexpected field '%s'", f.name)
		}
	}
}

func TestEntity_field(t *testing.T) {
	e := newEntity(testUser{})
	id, ok := e.field("id")
	if !ok {
		t.Fatal("expected entity to have field 'id'")
	}
	if id.name != "testUser.ID" {
		t.Fatalf("expected field name 'testUser.ID', got '%s'", id.name)
	}
	if id.tag != "id" {
		t.Fatalf("expected field tag 'id', got '%s'", id.tag)
	}
	name, ok := e.field("name")
	if !ok {
		t.Fatal("expected entity to have field 'name'")
	}
	if name.name != "testUser.Name" {
		t.Fatalf("expected field name 'testUser.Name', got '%s'", name.name)
	}
	if name.tag != "name" {
		t.Fatalf("expected field tag 'name', got '%s'", name.tag)
	}
	age, ok := e.field("age")
	if !ok {
		t.Fatal("expected entity to have field 'age'")
	}
	if age.name != "testUser.Age" {
		t.Fatalf("expected field name 'testUser.Age', got '%s'", age.name)
	}
	if age.tag != "age" {
		t.Fatalf("expected field tag 'age', got '%s'", age.tag)
	}
}
func TestEntity_field_Nested(t *testing.T) {
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
	e := newEntity(testStruct{})
	id, ok := e.field("id")
	if !ok {
		t.Fatal("expected entity to have field 'id'")
	}
	if id.name != "testStruct.ID" {
		t.Fatalf("expected field name 'testStruct.ID', got '%s'", id.name)
	}
	if id.tag != "id" {
		t.Fatalf("expected field tag 'id', got '%s'", id.tag)
	}
	name, ok := e.field("name")
	if !ok {
		t.Fatal("expected entity to have field 'name'")
	}
	if name.name != "testStruct.Name" {
		t.Fatalf("expected field name 'testStruct.Name', got '%s'", name.name)
	}
	if name.tag != "name" {
		t.Fatalf("expected field tag 'name', got '%s'", name.tag)
	}
	ts2, ok := e.field("ts2")
	if !ok {
		t.Fatal("expected entity to have field 'ts2'")
	}
	if ts2.name != "testStruct.TS2" {
		t.Fatalf("expected field name 'testStruct.TS2', got '%s'", ts2.name)
	}
	if ts2.tag != "ts2" {
		t.Fatalf("expected field tag 'ts2', got '%s'", ts2.tag)
	}
	_, ok = e.field("ts2.name")
	if ok {
		t.Fatal("expected entity to not have field 'ts2.name'")
	}
}

func TestEntity_hasID(t *testing.T) {
	e := newEntity(testUser{})
	if !e.hasID() {
		t.Fatal("expected entity to have id")
	}
}

func Test_fieldTagValues(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"   validate:"id"`
		Name string `json:"name" validate:"required,gt=2"`
		Age  int    `json:"age"  validate:"required,gt=0,lt=130"`
		None string
	}
	v := testStruct{
		ID:   "123",
		Name: "Alice",
		Age:  30,
		None: "none",
	}
	j0 := fieldTagValues("json", reflect.TypeOf(v).Field(0))
	v0 := fieldTagValues("validate", reflect.TypeOf(v).Field(0))
	if len(j0) != 1 || j0[0] != "id" {
		t.Fatalf("expected json tag 'id', got: %+v", j0)
	}
	if len(v0) != 1 || v0[0] != "id" {
		t.Fatalf("expected validate tag 'id', got: %+v", v0)
	}

	j1 := fieldTagValues("json", reflect.TypeOf(v).Field(1))
	v1 := fieldTagValues("validate", reflect.TypeOf(v).Field(1))
	if len(j1) != 1 || j1[0] != "name" {
		t.Fatalf("expected json tag 'name', got: %+v", j1)
	}
	if len(v1) != 2 || v1[0] != "required" || v1[1] != "gt=2" {
		t.Fatalf("expected validate tag 'required,gt=2', got: %+v", v1)
	}

	j2 := fieldTagValues("json", reflect.TypeOf(v).Field(2))
	v2 := fieldTagValues("validate", reflect.TypeOf(v).Field(2))
	if len(j2) != 1 || j2[0] != "age" {
		t.Fatalf("expected json tag 'age', got: %+v", j2)
	}
	if len(v2) != 3 || v2[0] != "required" || v2[1] != "gt=0" || v2[2] != "lt=130" {
		t.Fatalf("expected validate tag 'required,gt=0,lt=130', got: %+v", v2)
	}

	j3 := fieldTagValues("json", reflect.TypeOf(v).Field(3))
	v3 := fieldTagValues("validate", reflect.TypeOf(v).Field(3))
	if len(j3) != 0 {
		t.Fatal("expected no json tag")
	}
	if len(v3) != 0 {
		t.Fatal("expected no validate tag")
	}
}
