package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Compatible type classes
const (
	InvalidClass int = iota
	BoolClass
	IntClass
	UintClass
	FloatClass
	ComplexClass
	SliceClass
	ChanClass
	FuncClass
	InterfaceClass
	MapClass
	PtrClass
	StringClass
	StructClass
	UnsafePointerClass
)

// Assignability
const (
	Assignable int = iota
	Convertible
	Incompatible
)

var (
	// ErrorNotStruct indicates the passed in value is not a struct
	ErrorNotStruct = errors.New("not a struct")
	// ErrorNoSetValue indicates CanSet returns false
	ErrorNoSetValue = errors.New("not allowed to set value")
	// ErrorInvalidValue indicates the value is invalid
	ErrorInvalidValue = errors.New("invalid value")
	// ErrorKeyTypeMismatch indicates incompatible map key types
	ErrorKeyTypeMismatch = errors.New("map key type mismatch")

	// StringType defined and used as a const
	StringType = reflect.TypeOf("")
	// InterfaceType defined and used as a const
	InterfaceType = reflect.TypeOf([]interface{}{}).Elem()
)

// FieldInfo contains parsed information from struct field
type FieldInfo struct {
	Exported  bool
	Squash    bool
	OmitEmpty bool
	Wildcard  bool
	Ignore    bool
	MapName   string
}

// TypeClass converts reflect.Kind to compatible class
func TypeClass(kind reflect.Kind) int {
	switch kind {
	case reflect.Invalid:
		return InvalidClass
	case reflect.Bool:
		return BoolClass
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return IntClass
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return UintClass
	case reflect.Float32, reflect.Float64:
		return FloatClass
	case reflect.Complex64, reflect.Complex128:
		return ComplexClass
	case reflect.Array, reflect.Slice:
		return SliceClass
	case reflect.Chan:
		return ChanClass
	case reflect.Func:
		return FuncClass
	case reflect.Interface:
		return InterfaceClass
	case reflect.Map:
		return MapClass
	case reflect.Ptr:
		return PtrClass
	case reflect.String:
		return StringClass
	case reflect.Struct:
		return StructClass
	case reflect.UnsafePointer:
		return UnsafePointerClass
	}
	panic("Unknown kind " + kind.String())
}

// TypeCompatibility determines the assignment/conversion compatibility
func TypeCompatibility(from, to reflect.Type) int {
	if from.AssignableTo(to) {
		return Assignable
	} else if from.ConvertibleTo(to) {
		fromClass := TypeClass(from.Kind())
		toClass := TypeClass(to.Kind())
		if toClass == StringClass && (fromClass == IntClass || fromClass == UintClass) {
			return Incompatible
		}
		if fromClass == FloatClass && (toClass == IntClass || toClass == UintClass) {
			return Incompatible
		}
		return Convertible
	}
	return Incompatible
}

// TypeConverter defines the function to convert values
// IsValid is false if not convertible
type TypeConverter func(reflect.Value) reflect.Value

// TypeConverterFactory creates the converter by types
func TypeConverterFactory(from, to reflect.Type) TypeConverter {
	switch TypeCompatibility(from, to) {
	case Assignable:
		return func(v reflect.Value) reflect.Value { return v }
	case Convertible:
		return func(v reflect.Value) reflect.Value { return v.Convert(to) }
	default:
		if from.Kind() == reflect.Interface {
			return func(v reflect.Value) (r reflect.Value) {
				if v.CanInterface() {
					v = reflect.ValueOf(v.Interface())
					switch TypeCompatibility(v.Type(), to) {
					case Assignable:
						r = v
					case Convertible:
						r = v.Convert(to)
					}
				}
				return
			}
		}
	}
	return nil
}

// UnwrapInterface returns the actual value of the interface
func UnwrapInterface(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

// UnwrapPtr returns the actual value pointed to
func UnwrapPtr(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// IsEmpty determine if the value is an empty value
func IsEmpty(v reflect.Value) bool {
	for {
		switch TypeClass(v.Kind()) {
		case InvalidClass:
			return true
		case IntClass:
			return v.Int() == 0
		case UintClass:
			return v.Uint() == 0
		case FloatClass:
			return v.Float() == 0
		case ComplexClass:
			return v.Complex() == 0
		case SliceClass, MapClass, StringClass:
			return v.Len() == 0
		case InterfaceClass:
			v = UnwrapInterface(v)
		case PtrClass:
			v = UnwrapPtr(v)
		case UnsafePointerClass:
			return v.Pointer() == 0
		default:
			return false
		}
	}
}

// Mapper assign dynamic values
type Mapper struct {
	FieldTags []string
}

func (m *Mapper) assignValue(d, s reflect.Value) (assigned bool, err error) {
	if !d.IsValid() {
		return false, ErrorInvalidValue
	}
	if !s.IsValid() {
		return
	}

	if d.Kind() == reflect.Ptr {
		return m.assignToPtr(d, s)
	}

	if !d.CanSet() {
		return false, ErrorNoSetValue
	}

	if s.Kind() == reflect.Interface {
		s = UnwrapInterface(s)
		if !s.IsValid() {
			return
		}
	}

	switch TypeClass(d.Kind()) {
	case SliceClass:
		assigned, err = m.assignToSlice(d, s)
	case MapClass:
		assigned, err = m.assignToMap(d, s)
	case StructClass:
		assigned, err = m.assignToStruct(d, s)
	default:
		switch TypeCompatibility(s.Type(), d.Type()) {
		case Assignable:
			d.Set(s)
			assigned = true
		case Convertible:
			d.Set(s.Convert(d.Type()))
			assigned = true
		}
	}
	if assigned || err != nil {
		return
	}
	if s.Kind() == reflect.Ptr {
		return m.assignValue(d, s.Elem())
	}

	return false, fmt.Errorf("unable to assign from type %s to %s",
		s.Kind().String(), d.Kind().String())
}

func (m *Mapper) assignToPtr(d, s reflect.Value) (bool, error) {
	if !d.CanSet() {
		return m.assignValue(d.Elem(), s)
	}
	if s.Type().ConvertibleTo(d.Type()) {
		d.Set(s.Convert(d.Type()))
		return true, nil
	}
	v := reflect.New(d.Type().Elem())
	assigned, err := m.assignValue(v.Elem(), s)
	if err == nil && assigned {
		d.Set(v)
	}
	return assigned, err
}

func (m *Mapper) assignToSlice(d, s reflect.Value) (assigned bool, err error) {
	if TypeClass(s.Kind()) == SliceClass {
		v := reflect.MakeSlice(d.Type(), s.Len(), s.Len())
		for i := 0; i < s.Len(); i++ {
			if a, err := m.assignValue(v.Index(i), s.Index(i)); err != nil {
				return false, err
			} else if a {
				assigned = true
			}
		}
		if assigned {
			d.Set(v)
		}
	}
	return
}

func (m *Mapper) assignToMap(d, s reflect.Value) (assigned bool, err error) {
	switch TypeClass(s.Kind()) {
	case MapClass:
		convFn := TypeConverterFactory(s.Type().Key(), d.Type().Key())
		if convFn == nil {
			return false, ErrorKeyTypeMismatch
		}
		keys := s.MapKeys()
		if len(keys) > 0 {
			elemType := d.Type().Elem()
			v := reflect.MakeMap(reflect.MapOf(d.Type().Key(), elemType))
			for _, key := range keys {
				val := reflect.New(elemType).Elem()
				if _, err = m.assignValue(val, s.MapIndex(key)); err != nil {
					return
				}
				cvKey := convFn(key)
				if cvKey.IsValid() {
					v.SetMapIndex(cvKey, val)
				} else {
					return false, ErrorKeyTypeMismatch
				}
			}
			d.Set(v)
			assigned = true
		}
	case StructClass:
		if d.Type().Elem().Kind() != reflect.Interface {
			return
		}
		convFn := TypeConverterFactory(StringType, d.Type().Key())
		if convFn == nil {
			return false, ErrorKeyTypeMismatch
		}
		v := reflect.MakeMap(d.Type())
		errs := make(map[string]*structAssignErr)
		m.assignStructToMap(v, s, convFn, errs)
		for _, e := range errs {
			if len(e.errs) > 0 && e.succeeded == 0 {
				return false, e.errs[0]
			}
		}
		d.Set(v)
		assigned = true
	}
	return
}

func (m *Mapper) assignToStruct(d, s reflect.Value) (assigned bool, err error) {
	switch TypeClass(s.Kind()) {
	case StructClass:
		if s.Type().AssignableTo(d.Type()) {
			d.Set(s)
			assigned = true
		}
	case MapClass:
		convFn := TypeConverterFactory(s.Type().Key(), StringType)
		if convFn != nil {
			v := reflect.New(d.Type()).Elem()
			errs := make(map[string]*structAssignErr)
			keys := make(map[string]*mapKeyAssign)
			for _, key := range s.MapKeys() {
				cvKey := convFn(key)
				if cvKey.IsValid() {
					keys[cvKey.String()] = &mapKeyAssign{key: key}
				}
			}
			m.assignMapToStruct(v, s, keys, errs)
			for _, e := range errs {
				if len(e.errs) > 0 && e.succeeded == 0 {
					return false, e.errs[0]
				}
			}
			unassignedCnt := 0
			for _, mka := range keys {
				if !mka.assigned {
					unassignedCnt++
				}
			}
			if unassignedCnt > 0 {
				// some unassigned keys left, looking for a wildcard map
				for i := 0; i < v.NumField(); i++ {
					field := v.Type().Field(i)
					info := m.ParseField(field)
					// looking for a wildcard map
					if !info.Wildcard || field.Type.Kind() != reflect.Map {
						continue
					}
					// map key/value convertible
					keyConvFn := TypeConverterFactory(s.Type().Key(), field.Type.Key())
					valConvFn := TypeConverterFactory(s.Type().Elem(), field.Type.Elem())
					if keyConvFn == nil || valConvFn == nil {
						continue
					}
					m := v.Field(i)
					if m.IsNil() {
						m.Set(reflect.MakeMap(field.Type))
					}
					for _, mka := range keys {
						if mka.assigned {
							continue
						}
						cvKey := keyConvFn(mka.key)
						cvVal := valConvFn(s.MapIndex(mka.key))
						if !cvKey.IsValid() || !cvVal.IsValid() {
							continue
						}
						m.SetMapIndex(cvKey, cvVal)
					}
					break
				}
			}
			d.Set(v)
			assigned = true
		}
	default:
		for i := 0; i < d.NumField(); i++ {
			field := d.Type().Field(i)
			info := m.ParseField(field)
			if info.Wildcard {
				t := field.Type
				for t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				convFn := TypeConverterFactory(s.Type(), t)
				if convFn != nil {
					convVal := convFn(s)
					if convVal.IsValid() {
						return m.assignValue(d.Field(i), convFn(s))
					}
				}
			}
		}
	}
	return
}

type structAssignErr struct {
	succeeded int
	errs      []error
}

type mapKeyAssign struct {
	key      reflect.Value
	assigned bool
}

func (m *Mapper) assignStructToMap(d, s reflect.Value, convFn TypeConverter, errs map[string]*structAssignErr) {
	for i := 0; i < s.NumField(); i++ {
		field := s.Type().Field(i)
		info := m.ParseField(field)
		var err error
		var assignedVal reflect.Value
		if field.Type.Kind() == reflect.Struct {
			if field.Anonymous || info.Squash {
				m.assignStructToMap(d, s.Field(i), convFn, errs)
			} else {
				assignedVal = reflect.MakeMap(reflect.MapOf(StringType, InterfaceType))
				m.assignStructToMap(assignedVal, s.Field(i), convFn, errs)
			}
		} else if info.Exported && !info.Ignore && info.MapName != "" {
			v := s.Field(i)
			if !v.IsValid() || (IsEmpty(v) && info.OmitEmpty) {
				continue
			}
			var val interface{}
			pv := reflect.ValueOf(&val)
			_, err = m.assignValue(pv.Elem(), v)
			assignedVal = pv.Elem()
		}
		if assignedVal.IsValid() {
			key := convFn(reflect.ValueOf(info.MapName))
			if key.IsValid() {
				d.SetMapIndex(key, assignedVal)
			} else {
				err = ErrorKeyTypeMismatch
			}
		}
		assignErr := errs[info.MapName]
		if assignErr == nil {
			assignErr = &structAssignErr{}
			errs[info.MapName] = assignErr
		}
		if err != nil {
			assignErr.errs = append(assignErr.errs, err)
		} else {
			assignErr.succeeded++
		}
	}
}

func (m *Mapper) assignMapToStruct(d, s reflect.Value, keys map[string]*mapKeyAssign, errs map[string]*structAssignErr) {
	for i := 0; i < d.Type().NumField(); i++ {
		field := d.Type().Field(i)
		info := m.ParseField(field)
		if (field.Anonymous || info.Squash) && field.Type.Kind() == reflect.Struct {
			m.assignMapToStruct(d.Field(i), s, keys, errs)
		} else if key := info.MapName; info.Exported && !info.Ignore && key != "" {
			if mka, exist := keys[key]; !exist {
				continue
			} else if mapVal := s.MapIndex(mka.key); !mapVal.IsValid() {
				continue
			} else {
				assignErr := errs[key]
				if assignErr == nil {
					assignErr = &structAssignErr{}
					errs[key] = assignErr
				}
				assigned, err := m.assignValue(d.Field(i), s.MapIndex(mka.key))
				if err != nil {
					assignErr.errs = append(assignErr.errs, err)
				} else {
					assignErr.succeeded++
				}
				if assigned {
					mka.assigned = true
				}
			}
		}
	}
}

// ParseField extracts useful information from struct field
func (m *Mapper) ParseField(f reflect.StructField) *FieldInfo {
	info := &FieldInfo{}
	info.Exported = len(f.Name) > 0 && f.Name[0] >= 'A' && f.Name[0] <= 'Z'
	if !f.Anonymous && info.Exported {
		info.MapName = f.Name
		tags := m.FieldTags
		if len(tags) == 0 {
			tags = []string{"json"}
		}
		for _, tag := range tags {
			if val := f.Tag.Get(tag); val != "" {
				vals := strings.Split(val, ",")
				if vals[0] == "-" {
					info.Ignore = true
				} else if vals[0] != "" {
					info.MapName = vals[0]
					if info.MapName == "*" {
						info.Wildcard = true
					}
				}
				for i := 1; i < len(vals); i++ {
					switch vals[i] {
					case "squash":
						info.Squash = true
					case "omitempty":
						info.OmitEmpty = true
					}
				}
				break
			}
		}
	}
	return info
}

// MapValue copies values of reflect.Value
// If the destination is a pointer, the address is assigned
func (m *Mapper) MapValue(v, s reflect.Value) error {
	_, err := m.assignValue(v, s)
	return err
}

// Map assign values between interface{} types
func (m *Mapper) Map(v, s interface{}) error {
	return m.MapValue(reflect.ValueOf(v), reflect.ValueOf(s))
}
