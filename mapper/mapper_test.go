package mapper

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func tracedMapper(t *testing.T) *Mapper {
	return &Mapper{Tracer: func(d, s reflect.Value, loc string) {
		t.Log(loc, d.Kind().String(), s.Kind().String())
	}}
}

type struct1 struct {
	StrPtr   *string `json:"strptr"`
	Str      string
	FloatPtr *float64
	Skip     string `json:"-"`
	internal int
}

type struct2 struct {
	Ref1 struct1
	Ptr1 *struct1
	Map  map[string]*struct1
	Arr1 []*struct1
}

type struct3 struct {
	struct1
	Val int
}

type struct4 struct {
	Str1 string  `json:"str"`
	Str2 *string `json:"str"`
	Int1 int     `json:"str"`
}

type struct5 struct {
	S4 map[string]*struct4
}

func TestMapScalar(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	var int1, int2 int
	int2 = 10
	if a.NoError(m.MapValue(reflect.ValueOf(&int1), reflect.ValueOf(int2))) {
		a.Equal(10, int1)
	}
	int1 = 0
	if a.NoError(m.Map(&int1, int2)) {
		a.Equal(10, int1)
	}
	int1 = 0
	if a.NoError(m.Map(&int1, &int2)) {
		a.Equal(10, int1)
	}
	var s1, s2 string
	s2 = "hello"
	if a.NoError(m.Map(&s1, s2)) {
		a.Equal("hello", s1)
	}
	b1 := false
	if a.NoError(m.Map(&b1, true)) {
		a.True(b1)
	}
	c1 := complex(1, 1)
	c2 := complex(2, 2)
	if a.NoError(m.Map(&c1, c2)) {
		a.EqualValues(complex(2, 2), c1)
	}
}

func TestMapConvert(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	var int1 int
	if a.NoError(m.Map(&int1, int64(10))) {
		a.Equal(10, int1)
	}
	if a.NoError(m.Map(&int1, uint64(20))) {
		a.Equal(20, int1)
	}
	a.Error(m.Map(&int1, 3.4))
	var num1 float32
	if a.NoError(m.Map(&num1, float64(1.1))) {
		a.EqualValues(1.1, num1)
	}
	if a.NoError(m.Map(&num1, int(2))) {
		a.EqualValues(2, num1)
	}
	if a.NoError(m.Map(&num1, uint(3))) {
		a.EqualValues(3, num1)
	}
	var uint1 uint
	if a.NoError(m.Map(&uint1, int64(10))) {
		a.EqualValues(10, uint1)
	}
	if a.NoError(m.Map(&uint1, uint64(20))) {
		a.EqualValues(20, uint1)
	}
}

func TestMapKeyType(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	s1 := map[string]interface{}{"key": 10}
	s2 := map[interface{}]interface{}{"key": "hello"}
	if a.NoError(m.Map(&s1, s2)) {
		a.Equal("hello", s1["key"])
	}
	i1 := map[int]interface{}{1: 100}
	if a.NoError(m.Map(&s2, i1)) {
		a.Equal(100, s2[1])
	}
	a.Error(m.Map(&s1, i1))
}

func TestMapPtr(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	var p *struct1
	s := struct1{Str: "str"}
	if a.NoError(m.Map(&p, &s)) {
		a.NotNil(p)
		a.Equal(&s, p)
	}
	s.StrPtr = &s.Str
	if a.NoError(m.Map(&s, map[string]interface{}{"strptr": "str1"})) {
		a.Equal("str1", s.Str)
		a.Equal(&s.Str, s.StrPtr)
	}

	dict := map[string]interface{}{"strptr": &s.Str, "other": "other"}
	if a.NoError(m.Map(dict, map[string]interface{}{"strptr": "str2"})) {
		_, ok := dict["strptr"].(*string)
		a.False(ok)
		strVal, ok := dict["strptr"].(string)
		a.True(ok)
		a.Equal("str2", strVal)
		a.Equal("str1", s.Str)
		a.EqualValues("other", dict["other"])
	}

	pstr1 := &s.Str
	str2 := "str2"
	pstr2 := &str2
	if a.NoError(m.Map(&pstr1, pstr2)) {
		a.Equal(&str2, pstr1)
		a.Equal(&str2, pstr2)
		a.Equal("str1", s.Str)
	}
}

func TestMapInterface(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	s := struct1{Str: "str"}
	var i interface{}
	if a.NoError(m.Map(&i, &s)) {
		s1, ok := i.(*struct1)
		if a.True(ok) {
			a.Equal("str", s1.Str)
		}
	}
	var p *struct1
	if a.NoError(m.Map(&p, i)) {
		a.Equal(&s, p)
	}
	if a.NoError(m.Map(&i, int(10))) {
		intVal, ok := i.(int)
		if a.True(ok) {
			a.Equal(10, intVal)
		}
	}
}

func TestMapChan(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	var p *chan struct{}
	v := make(chan struct{})
	a.NoError(m.Map(&p, &v))
}

func TestMapFunc(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	var fn func() int
	if a.NoError(m.Map(&fn, func() int { return 10 })) {
		a.Equal(10, fn())
	}
}

func TestMapStruct(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	s0 := &struct1{Str: "s1"}
	s1 := s0
	s2 := &struct1{Str: "s2"}
	if a.NoError(m.Map(s1, s2)) {
		s2.Str = "new"
		a.Equal("s2", s0.Str)
		s1.Str = "s0"
		a.Equal("s0", s0.Str)
		a.Equal("new", s2.Str)
	}
}

func TestAssignMap(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	src := map[string]interface{}{
		"Ref1": map[string]interface{}{
			"strptr":   "s1",
			"Str":      "s1",
			"FloatPtr": 1.1,
			"Skip":     "s1",
			"internal": 10,
		},
		"Ptr1": map[string]interface{}{
			"Str":      nil,
			"FloatPtr": nil,
		},
		"Map": map[string]interface{}{
			"k1": map[string]interface{}{
				"StrPtr": nil,
				"Str":    "s2",
			},
		},
		"Arr1": []map[string]interface{}{
			map[string]interface{}{
				"Str": "s2",
			},
		},
	}
	var d struct2
	if a.NoError(m.Map(&d, src)) {
		if a.NotNil(d.Ref1.StrPtr) {
			a.Equal("s1", *d.Ref1.StrPtr)
		}
		a.Equal("s1", d.Ref1.Str)
		if a.NotNil(d.Ref1.FloatPtr) {
			a.Equal(1.1, *d.Ref1.FloatPtr)
		}
		a.Equal("", d.Ref1.Skip)
		a.Equal(0, d.Ref1.internal)
		if a.NotNil(d.Ptr1) {
			a.Equal("", d.Ptr1.Str)
			a.Nil(d.Ptr1.FloatPtr)
		}
		k1 := d.Map["k1"]
		if a.NotNil(k1) {
			a.Nil(k1.StrPtr)
			a.Equal("s2", k1.Str)
		}
		if a.Len(d.Arr1, 1) && a.NotNil(d.Arr1[0]) {
			a.Equal("s2", d.Arr1[0].Str)
		}
	}
}

func TestMapAnonStructField(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	src := map[string]interface{}{
		"Str": "s1",
		"Val": 101,
	}
	var s struct3
	if a.NoError(m.Map(&s, src)) {
		a.Equal("s1", s.Str)
		a.Equal(101, s.Val)
	}
}

func TestMapMultiStructFields(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	src := map[string]interface{}{"str": "s1"}
	var s struct4
	if a.NoError(m.Map(&s, src)) {
		a.Equal("s1", s.Str1)
		if a.NotNil(s.Str2) {
			a.Equal("s1", *s.Str2)
		}
		a.Equal(0, s.Int1)
	}
	a.Error(m.Map(&s, map[string]interface{}{"str": 1.6}))

	var s5 struct5
	src = map[string]interface{}{
		"S4": map[string]interface{}{
			"a1": map[string]interface{}{"str": 101},
			"a2": map[string]interface{}{"str": "s2"},
		},
	}
	if a.NoError(m.Map(&s5, src)) {
		a1 := s5.S4["a1"]
		if a.NotNil(a1) {
			a.Equal("", a1.Str1)
			a.Nil(a1.Str2)
			a.Equal(101, a1.Int1)
		}
		a2 := s5.S4["a2"]
		if a.NotNil(a2) {
			a.Equal("s2", a2.Str1)
			a.Equal("s2", *a2.Str2)
			a.Equal(0, a2.Int1)
		}
	}
}

type wildcardStruct struct {
	Str string `json:"*"`
	Int int    `json:"*"`
}

type wildcardPtrStruct struct {
	Str *string `json:"*"`
	Int **int   `json:"*"`
}

type wildcardMapStruct struct {
	Str      string                 `json:"str"`
	Squashed struct4                `json:",squash"`
	Ext      map[string]interface{} `json:"*"`
}

func TestMapWildcardStructField(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)
	s1 := &wildcardStruct{}
	if a.NoError(m.Map(s1, "str")) {
		a.Equal("str", s1.Str)
	}
	s2 := &wildcardStruct{}
	if a.NoError(m.Map(s2, int64(10))) {
		a.Equal(10, s2.Int)
	}

	s3 := &wildcardPtrStruct{}
	if a.NoError(m.Map(s3, "str")) {
		if a.NotNil(s3.Str) {
			a.Equal("str", *s3.Str)
		}
	}
	s4 := &wildcardPtrStruct{}
	if a.NoError(m.Map(s4, int64(10))) {
		if a.NotNil(s4.Int) && a.NotNil(*s4.Int) {
			a.Equal(10, **s4.Int)
		}
	}

	s5 := &wildcardMapStruct{}
	if a.NoError(m.Map(s5, map[string]interface{}{"str": "str", "ext": 10})) {
		a.Equal("str", s5.Str)
		a.Equal("str", s5.Squashed.Str1)
		if a.NotNil(s5.Squashed.Str2) {
			a.Equal("str", *s5.Squashed.Str2)
		}
		if a.NotNil(s5.Ext) {
			a.Contains(s5.Ext, "ext")
			a.EqualValues(10, s5.Ext["ext"])
		}
	}
}

type ToMapNested struct {
	Dict map[string]interface{} `json:"dict"`
}

type toMapNested1 struct {
	Str1 string  `json:"str1"`
	Str2 *string `json:"str2,omitempty"`
}

type ToMap struct {
	Str    string  `json:"str,omitempty"`
	PtrStr *string `json:"pstr,omitempty"`
	IntStr *int    `json:"str,omitempty"`

	ToMapNested
	Squashed toMapNested1 `json:",squash"`
	SubStru  toMapNested1 `json:"sub"`
}

func TestStructToMap(t *testing.T) {
	a := assert.New(t)
	m := tracedMapper(t)

	s1 := &ToMap{Str: "str"}
	s1.Dict = map[string]interface{}{"a": 0.1}
	s1.Squashed.Str1 = "str"
	str2 := "str2"
	s1.SubStru.Str2 = &str2

	d := make(map[string]interface{})
	if a.NoError(m.Map(d, s1)) {
		if a.Contains(d, "str") {
			a.Equal("str", d["str"])
		}
		a.NotContains(d, "pstr")
		if a.Contains(d, "dict") {
			a.Equal(map[string]interface{}{"a": 0.1}, d["dict"])
		}
		if a.Contains(d, "str1") {
			a.Equal("str", d["str1"])
		}
		a.NotContains(d, "str2")
		if a.Contains(d, "sub") {
			sub, ok := d["sub"].(map[string]interface{})
			if a.True(ok) && a.Contains(sub, "str2") {
				ptrStr, ok := sub["str2"].(*string)
				if a.True(ok) && a.NotNil(ptrStr) {
					a.Equal("str2", *ptrStr)
				}
			}
		}
	}
}
