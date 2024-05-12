---
description: Built-in JSON Schema validation rules for request parameters and bodies.
---

# Request Validation

## Request Validation { .hidden }

Go struct tags are used to annotate inputs/output structs with information that gets turned into [JSON Schema](https://json-schema.org/) for documentation and validation. For example:

```go title="code.go"
type Person struct {
    Name string `json:"name" doc:"Person's name" minLength:"1" maxLength:"80"`
    Age  uint   `json:"age,omitempty" doc:"Person's age" maximum:"120"`
}
```

## Field Naming

The standard `json` tag is supported and can be used to rename a field. Any field tagged with `json:"-"` will be ignored in the schema, as if it did not exist.

## Optional / Required

Fields being optional/required is determined automatically but can be overidden as needed using the logic below:

1. Start with all fields required.
2. If a field has `omitempty`, it is optional.
3. If a field has `required:"false"`, it is optional.
4. If a field has `required:"true"`, it is required.

Pointers have no effect on optional/required. The same rules apply regardless of whether the struct is being used for request input or response output. Some examples:

```go
type MyStruct struct {
    // The following are all required.
    Required1 string  `json:"required1"`
    Required2 *string `json:"required2"`
    Required3 string  `json:"required3,omitempty" required:"true"`

    // The following are all optional.
    Optional1 string  `json:"optional1,omitempty"`
    Optional2 *string `json:"optional2,omitempty"`
    Optional3 string  `json:"optional3" required:"false"`
}
```

!!! info "Note"

    Why use `omitempty` for inputs when Go itself only uses the field for marshaling? Imagine a client which is going to send a request to your API - it must still be marshaled into JSON (or a similar format). You can think of your input structs as modeling what an API client would produce as output.

## Nullable

In many languages (including Go), there is little to no distinction between an explicit empty value vs. an undefined one. Marking a field as optional as explained above is enough to support either case. Javascript & Typescript are exceptions to this rule, as they have explicit `null` and `undefined` values.

Huma tries to balance schema simplicity, usability, and broad compatibility with schema correctness and a broad range of language support for end-to-end API tooling. To that end, it supports field nullability to a limited extent, and future changes may modify this default behavior as tools become more compatible with advanced JSON Schema features.

Fields being nullable is determined automatically but can be overidden as needed using the logic below:

1. Start with no fields as nullable
2. If a field is a pointer:
    1. To a `boolean`, `integer`, `number`, `string`: it is nullable unless it has `omitempty`.
    2. To an `array`, `object`: it is **not** nullable, due to complexity and bad support for `anyOf`/`oneOf` in many tools.
3. If a field has `nullable:"false"`, it is not nullable
4. If a field has `nullable:"true"`:
    1. To a `boolean`, `integer`, `number`, `string`: it is nullable
    2. To an `array`, `object`: **panic** saying this is not currently supported
5. If a struct has a field `_` with `nullable: true`, the struct is nullable enabling users to opt-in for `object` without the `anyOf`/`oneOf` complication.

Here are some examples:

```go title="code.go"
// Make an entire struct (not its fields) nullable.
type MyStruct1 struct {
    _ struct{} `nullable:"true"`
    Field1 string `json:"field1"`
    Field2 string `json:"field2"`
}

// Make a specific scalar field nullable. This is *not* supported for
// slices, maps, or structs. Structs *must* use the method above.
type MyStruct2 struct {
    Field1 *string `json:"field1"`
    Field2 string `json:"field2" nullable:"true"`
}
```

Nullable types will generate a type array like `"type": ["string", "null"]` which has broad compatibility and is easy to downgrade to OpenAPI 3.0. Also keep in mind you can always provide a [custom schema](./schema-customization.md) if the built-in features aren't exactly what you need.

## Validation Tags

The following additional tags are supported on model fields:

| Tag                  | Description                                | Example                         |
| -------------------- | ------------------------------------------ | ------------------------------- |
| `doc`                | Describe the field                         | `doc:"Who to greet"`            |
| `format`             | Format hint for the field                  | `format:"date-time"`            |
| `enum`               | A comma-separated list of possible values  | `enum:"one,two,three"`          |
| `default`            | Default value                              | `default:"123"`                 |
| `minimum`            | Minimum (inclusive)                        | `minimum:"1"`                   |
| `exclusiveMinimum`   | Minimum (exclusive)                        | `exclusiveMinimum:"0"`          |
| `maximum`            | Maximum (inclusive)                        | `maximum:"255"`                 |
| `exclusiveMaximum`   | Maximum (exclusive)                        | `exclusiveMaximum:"100"`        |
| `multipleOf`         | Value must be a multiple of this value     | `multipleOf:"2"`                |
| `minLength`          | Minimum string length                      | `minLength:"1"`                 |
| `maxLength`          | Maximum string length                      | `maxLength:"80"`                |
| `pattern`            | Regular expression pattern                 | `pattern:"[a-z]+"`              |
| `patternDescription` | Description of the pattern used for errors | `patternDescription:"alphanum"` |
| `minItems`           | Minimum number of array items              | `minItems:"1"`                  |
| `maxItems`           | Maximum number of array items              | `maxItems:"20"`                 |
| `uniqueItems`        | Array items must be unique                 | `uniqueItems:"true"`            |
| `minProperties`      | Minimum number of object properties        | `minProperties:"1"`             |
| `maxProperties`      | Maximum number of object properties        | `maxProperties:"20"`            |
| `example`            | Example value                              | `example:"123"`                 |
| `readOnly`           | Sent in the response only                  | `readOnly:"true"`               |
| `writeOnly`          | Sent in the request only                   | `writeOnly:"true"`              |
| `deprecated`         | This field is deprecated                   | `deprecated:"true"`             |
| `hidden`             | Hide field/param from documentation        | `hidden:"true"`                 |
| `dependentRequired`  | Required fields when the field is present  | `dependentRequired:"one,two"`   |

Built-in string formats include:

| Format                            | Description                     | Example                                |
| --------------------------------- | ------------------------------- | -------------------------------------- |
| `date-time`                       | Date and time in RFC3339 format | `2021-12-31T23:59:59Z`                 |
| `date-time-http`                  | Date and time in HTTP format    | `Fri, 31 Dec 2021 23:59:59 GMT`        |
| `date`                            | Date in RFC3339 format          | `2021-12-31`                           |
| `time`                            | Time in RFC3339 format          | `23:59:59`                             |
| `email` / `idn-email`             | Email address                   | `kari@example.com`                     |
| `hostname`                        | Hostname                        | `example.com`                          |
| `ipv4`                            | IPv4 address                    | `127.0.0.1`                            |
| `ipv6`                            | IPv6 address                    | `::1`                                  |
| `uri` / `iri`                     | URI                             | `https://example.com`                  |
| `uri-reference` / `iri-reference` | URI reference                   | `/path/to/resource`                    |
| `uri-template`                    | URI template                    | `/path/{id}`                           |
| `json-pointer`                    | JSON Pointer                    | `/path/to/field`                       |
| `relative-json-pointer`           | Relative JSON Pointer           | `0/1`                                  |
| `regex`                           | Regular expression              | `[a-z]+`                               |
| `uuid`                            | UUID                            | `550e8400-e29b-41d4-a716-446655440000` |

## Strict vs. Loose Field Validation

By default, Huma is strict about which fields are allowed in an object, making use of the `additionalProperties: false` JSON Schema setting. This means if a client sends a field that is not defined in the schema, the request will be rejected with an error. This can help to prevent typos and other issues and is recommended for most APIs.

If you need to allow additional fields, for example when using a third-party service which will call your system and you only care about a few fields, you can use the `additionalProperties:"true"` field tag on the struct by assigning it to a dummy `_` field.

```go title="code.go"
type PartialInput struct {
 _      struct{} `json:"-" additionalProperties:"true"`
 Field1 string   `json:"field1"`
 Field2 bool     `json:"field2"`
}
```

!!! info "Note"

    The use of `struct{}` is optional but efficient. It is used to avoid allocating memory for the dummy field as an empty object requires no space.

## Advanced Validation

When using custom JSON Schemas, i.e. not generated from Go structs, it's possible to utilize a few more validation rules. The following schema fields are respected by the built-in validator:

- `not` for negation
- `oneOf` for exclusive inputs
- `anyOf` for matching one-or-more
- `allOf` for schema unions

See [`huma.Schema`](https://pkg.go.dev/github.com/ross96D/huma#Schema) for more information. Note that it may be easier to use a custom [resolver](./request-resolvers.md) to implement some of these rules.

## Dive Deeper

- Tutorial
  - [Your First API](../tutorial/your-first-api.md) includes string length validation
- Reference
  - [`huma.Register`](https://pkg.go.dev/github.com/ross96D/huma#Register) registers new operations
  - [`huma.Operation`](https://pkg.go.dev/github.com/ross96D/huma#Operation) the operation
- External Links
  - [JSON Schema Validation](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-validation-00)
  - [OpenAPI 3.1 Schema Object](https://spec.openapis.org/oas/v3.1.0#schema-object)
  - [OpenAPI 3.1 Operation Object](https://spec.openapis.org/oas/v3.1.0#operation-object)
