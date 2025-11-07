# JEM — JSON Entity Mapper

JEM (JSON Entity Mapper) is a Go library for mapping JSON data to structs and maps with validation, supporting both full and partial inputs. It provides a simple and flexible way to define entity schemas, enforce validation rules, and handle auto-generated fields. JEM is designed for developers who need to work with JSON APIs and want to ensure data integrity and consistency.

## Features

- map JSON input into go structs and maps
- define entity schemas directly from struct definitions
- handle nested structs and arrays seamlessly
- support both create (full) and update (partial) operations
- enforce validation rules via struct tags
- auto-generated and persisted fields
- detect unknown fields with strict JSON decoding
- integrate easily with databases and HTTP handlers

## Installation

```bash
go get github.com/shayanderson/jem
```

## Quick Start

Define your entity:

```go
type User struct {
    ID    string `json:"id" validate:"id,required"`
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0"`
}
```

Parse and map a single JSON object:

```go
f := jem.New[User, string]() // second type parameter is type of ID field

data := []byte(`{"name":"Alice","email":"alice@example.com","age":25}`)

res, err := f.Make(data)
if err != nil {
    panic(err)
}

fmt.Printf("res.Value: %+v\n", res.Value)
fmt.Printf("res.Map: %+v\n", res.Map)
// output:
// &{ID: Name:Alice Email:alice@example.com Age:25}
// map[age:25 email:alice@example.com name:Alice]
```

Parse and map multiple JSON objects:

```go
f := jem.New[User, string]()

arr := []byte(`[{"name":"Alice","email":"alice@example.com","age":25},
                {"name":"Bob","email":"bob@example.com","age":30}]`)

res, err := f.MakeMany(arr)
if err != nil {
    panic(err)
}

for k, v := range res {
    fmt.Printf("%d: v.Value %+v\n", k, v.Value)
    fmt.Printf("%d: v.Map %+v\n", k, v.Map)
}
// output:
// 0: v.Value &{ID: Name:Alice Email:alice@example.com Age:25}
// 0: v.Map map[age:25 email:alice@example.com name:Alice]
// 1: v.Value &{ID: Name:Bob Email:bob@example.com Age:30}
// 1: v.Map map[age:30 email:bob@example.com name:Bob]
```

Parse and map partial JSON objects:

```go
f := jem.New[User, string]()

r, err := f.MakePartial([]byte(`{"id":"u-101","name":"Bob"}`))
if err != nil {
    panic(err)
}
fmt.Println("r.ID:", r.ID)
fmt.Printf("r.Value: %+v\n", r.Value)
fmt.Printf("r.Map: %+v\n", r.Map)
// output:
// r.ID: u-101
// r.Value: &{ID:u-101 Name:Bob Email: Age:0}
// r.Map: map[name:Bob]
```

Parse and map multiple partial JSON objects:

```go
f := jem.New[User, string]()

arr := []byte(`[{"id":"u-101","name":"Bob"},
                {"id":"u-102","age":27}]`)

res, err := f.MakePartialMany(arr)
if err != nil {
    panic(err)
}

for k, v := range res {
    fmt.Printf("%d: v.ID: %s\n", k, v.ID)
    fmt.Printf("%d: v.Value %+v\n", k, v.Value)
    fmt.Printf("%d: v.Map %+v\n", k, v.Map)
}
// output:
// 0: v.ID: u-101
// 0: v.Value &{ID:u-101 Name:Bob Email: Age:0}
// 0: v.Map map[name:Bob]
// 1: v.ID: u-102
// 1: v.Value &{ID:u-102 Name: Email: Age:27}
// 1: v.Map map[age:27]
```

You can also use readers as input:

```go
f := jem.New[User, string]()

// example http handler
http.HandleFunc("POST /user", func(w http.ResponseWriter, r *http.Request) {
    defer r.Body.Close()

    doc, err := f.Read(r.Body)
    if err != nil {
        if errors.Is(err, jem.ErrRead) { // reading error
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        http.Error(w, "validation failed: "+err.Error(), http.StatusUnprocessableEntity)
        return
    }

    // use doc.ID, doc.Value and doc.Map
})
```

Reader methods are: `Read`, `ReadMany`, `ReadPartial`, `ReadPartialMany`.
By defaults `jem.LimitReadSize` is set to 10 MB to avoid reading very large inputs. You can set it to `0` to disable the limit or change it to a different size (in bytes).

## Validation

JEM integrates with [`go-playground/validator`](https://github.com/go-playground/validator). You can use all its built-in rules (required, email, gte, etc.).

JEM supports the following special validation rules. These rules apply only to **top-level** fields.

- `validate:"id"` — marks a unique identifier field
  - must not be included in full JSON input objects
    - can be used with `persist` to allow in full JSON input objects, like `validate:"id,persist"`
  - must be included in partial JSON input objects
  - removed from partial output maps
- `validate:"auto"`, `validate:"auto:full"`, `validate:"auto:partial"` — marks fields that must be [auto-generated](#auto-generated-fields)
  - must not be included in JSON input objects
  - must be included in auto-map when creating output
  - `auto` — must be included in auto-map when creating output
  - `auto:full` — must be included in auto-map when creating full output
  - `auto:partial` — must be included in auto-map when creating partial output
- `validate:"persist"` — marks fields that must always be included
  - should be used with rules like `required` if non-empty values are expected
- `validate:"readonly"` — marks fields that are read-only
  - must not be included in JSON input objects

Partial object validation is only allowed for **top-level** fields, meaning nested fields are not partially validated.

### Custom Validation Rules

You can define custom validation rules using the `RegisterValidation` method on the factory's validator:

```go
f := New[User, string]()
f.Validator().RegisterValidation("rule", func(fl validator.FieldLevel) bool {
    return fl.Field().String() == "test"
})
```

View the [`go-playground/validator`](https://github.com/go-playground/validator) documentation for more information on how to use custom validation rules.

### Error Handling

Validation errors are normalized and always returned as a single validation error value, even if multiple fields are invalid, like `field 'User.name' validation failed for rule 'required'`.

## Auto-Generated Fields

You can define fields that should be automatically set at runtime (e.g., IDs, timestamps). These fields are not allowed in JSON input.

Pass an AutoMap when creating entities:

```go
type User struct {
    ID        string `json:"id" validate:"id,required"`
    Name      string `json:"name" validate:"required"`
    CreatedAt int64  `json:"createdAt" validate:"auto:full,required"`
    UpdatedAt int64  `json:"updatedAt" validate:"auto,required"`
}

// ...
res, err := f.Make([]byte(`{"name":"Bob"}`), jem.AutoMap{
    "createdAt": func() any {
        return time.Now().Unix()
    },
    "updatedAt": func() any {
        return time.Now().Unix()
    },
})
// ...
fmt.Printf("r.Value: %+v\n", r.Value)
fmt.Printf("r.Map: %+v\n", r.Map)
// output:
// r.Value: &{ID: Name:Bob CreatedAt:1750000000 UpdatedAt:1750000000}
// r.Map: map[createdAt:1750000000 name:Bob updatedAt:1750000000]
```
