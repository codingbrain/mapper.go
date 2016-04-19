# Dynamic data manipulation library for Go

Manipulating data dynamically using `map` in Go is difficult.
The library provides a few utilities to handle common use cases.

### Features

#### Convert between `map` and `struct`

##### Sample 1

```go

...

import "github.com/codingbrain/mapper.go/mapper"

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

# License

MIT
