# Common library for Go

The library provides commonly used features for Go language.

### Features

#### Convert between `map` and `struct`

##### General sample

```go

...

import "github.com/easeway/langx.go/mapper"

...

type Person struct {
    Name    string `json:"name"`
    Address string `json:"address"`
    Age     int `json:"age"`
    Emails  []string `json:"emails"`
}

var JSONString = `{"name": "Brainer", "address": ...}`
func main() {
    map0 := make(map[string]interface{})
    err := json.Unmarshal([]byte(JSONString), map0)
    m := &mapper.Mapper{}
    var person Person
    m.Map(&person, map0)
    ...
    map1 := make(map[string]interface{})
    m.Map(map1, &person)
    json.Marshal(map1)
    ...
}
```

##### Anonymous structure

If the structure contains anonymous structures,
the fields are treated as the same level.

```go

type Person struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type User struct {
    Person
    Roles string `json:"roles"`
}

var JSONString = `{"name": "Brainer", "address": ..., "roles": ["admin", "dev"]}`
...
```

##### Squash structure

For non-anonymous structure,
flatten the fields, and achieve the same effect as anonymous structure.

```go

type Person struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type User struct {
    Person Person `json:",squash"`
    Roles  string `json:"roles"`
}

var JSONString = `{"name": "Brainer", "address": ..., "roles": ["admin", "dev"]}`
...
```

##### Multi-mapping

If a value in JSON can be of different types, multi-mapping solve the problem.

With the following JSON documents:

```json
{ "additionalProperties": true }
```

```json
{ "additionalProperties": ["alias", "age"] }
```

The structure can be defined as:

```go
type MultiMapping struct {
    AllowAdditionalProperties bool     `json:"additionalProperties,omitempty"`
    AdditionalProperties      []string `json:"additionalProperties,omitempty"`
}
```

Please note, `omitempty` is recommended.
Otherwise `Mapper` gets confused when converting the structure to map.

##### Wildcard mapping

In most cases, a `map` can be converted into a structure.
However, in some cases, the value can be a `map`, or some other values.
_wildcard_ fields are used to accept non-map values.

```go
type Command struct {
    Shell string `json:"*"`
    Macro string `json:"macro"`
}

type Target struct {
    Commands []*Command `json:"commands"`
}
```

The definition above can accept the following documents:

```json
{
    "commands": [
        { "macro": "wait" },
        ...
    ]
}
```

Or

```json
{
    "commands": [
        "mkdir -p /tmp/abc",
        "cp ...",
        {"macro": "wait"},
        ...
    ]
}
```

When the item in `commands` is a simple string,
it doesn't match the expected type `*Command`,
as `Command` has a _wildcard_ field of type string, the value is filled in.

It's very useful when the schema has a few fix properties and also open to
additional properties.
The following structure is usually defined for this case:

```go
type OpenStruct struct {
    Type       string                 `json:"type"`
    Properties map[string]interface{} `json:"*"`
}
```

Currently, structures with _wildcard_ fields can't be converted back to a map.

##### Override the tag name

It's not necessary to require `json` as tag name in struct fields.
Construct a `Mapper` with a list of tag names is possible:

```go
m := &Mapper{FieldTags: []string{"n", "map"}}
```

It will search for tags in the order of `n`, `map` until a tag is found.

##### Trace the mapping

This is mostly for debugging purpose.
Assign a function to `Mapper.Tracer` can track the traversal during conversion.

#### Aggregated Errors

Sometime multiple errors need aggregated and reported as a single error.
The type `AggregatedError` implements this behavior.

```go

import "github.com/easeway/langx.go/errors"

...
// First define an AggregatedError
errs := errors.AggregatedError{}
...
errs.Add(err)
// Or
errs.AddErr(err)
// Or
errs.AddMany(err1, err2, ...)
// Or if the function only returns error
errs.Add(os.Remove(...))
// And finally return
return errs.Aggregate()
```

When using `Add/AddErr/AddMany`, don't worry about `err` is `nil` or not,
`nil` won't be added.
And `Aggregate` only returns `nil` if no error is added.

# License

MIT
