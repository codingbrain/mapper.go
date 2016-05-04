package mapper

import (
	"fmt"
	"reflect"
	"strconv"
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
	// StringType defined and used as a const
	StringType = reflect.TypeOf("")
	// InterfaceType defined and used as a const
	InterfaceType = reflect.TypeOf([]interface{}{}).Elem()
)

func errNotStruct(loc string) error {
	return fmt.Errorf("not a struct [%s]", loc)
}

func errNoSetValue(loc string) error {
	return fmt.Errorf("not allowed to set value [%s]", loc)
}

func errInvalidValue(loc string) error {
	return fmt.Errorf("invalid value [%s]", loc)
}

func errKeyTypeMismatch(loc string) error {
	return fmt.Errorf("map key type mismatch [%s]", loc)
}

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

// UnwrapAny returns the actual value from interface/ptr
func UnwrapAny(v reflect.Value) reflect.Value {
	for {
		if v.Kind() == reflect.Interface {
			v = UnwrapInterface(v)
		} else if v.Kind() == reflect.Ptr {
			v = UnwrapPtr(v)
		} else {
			return v
		}
	}
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

// IsContainer determine if the value is map or struct
func IsContainer(v reflect.Value) bool {
	switch TypeClass(v.Kind()) {
	case MapClass, StructClass:
		return true
	}
	return false
}

// MapTracer receives the traversal in mapping
type MapTracer func(d, s reflect.Value, loc string)

// Mapper assign dynamic values
type Mapper struct {
	FieldTags []string
	Tracer    MapTracer
}

func locExp(loc, comp string) string {
	return loc + "." + comp
}

func locPtr(loc string) string {
	return loc + "*"
}

func locInterface(loc string) string {
	return loc + "@"
}

func (m *Mapper) traceMap(d, s reflect.Value, loc string) {
	if m.Tracer != nil {
		m.Tracer(d, s, loc)
	}
}

func (m *Mapper) assignValue(d, s reflect.Value, loc string) (assigned bool, err error) {
	m.traceMap(d, s, loc)

	if !d.IsValid() {
		return false, errInvalidValue(loc)
	}
	if !s.IsValid() {
		return
	}

	if d.Kind() == reflect.Ptr {
		return m.assignToPtr(d, s, loc)
	}
	if d.Kind() == reflect.Interface {
		return m.assignToInterface(d, s, loc)
	}

	if s.Kind() == reflect.Interface {
		s = UnwrapInterface(s)
		if !s.IsValid() {
			return
		}
	}

	switch TypeClass(d.Kind()) {
	case SliceClass:
		assigned, err = m.assignToSlice(d, s, loc)
	case MapClass:
		assigned, err = m.assignToMap(d, s, loc)
	case StructClass:
		assigned, err = m.assignToStruct(d, s, loc)
	default:
		assigned, err = m.assignToOther(d, s, loc)
	}
	if assigned || err != nil {
		return
	}
	if s.Kind() == reflect.Ptr {
		return m.assignValue(d, s.Elem(), loc)
	}

	return false, fmt.Errorf("unable to assign from type %s to %s [%s]",
		s.Kind().String(), d.Kind().String(), loc)
}

func (m *Mapper) assignToPtr(d, s reflect.Value, loc string) (bool, error) {
	if d.CanSet() && s.Type().ConvertibleTo(d.Type()) {
		d.Set(s.Convert(d.Type()))
		return true, nil
	}
	if !d.IsNil() {
		return m.assignValue(d.Elem(), s, locPtr(loc))
	}
	v := reflect.New(d.Type().Elem())
	assigned, err := m.assignValue(v.Elem(), s, locPtr(loc))
	if err == nil && assigned {
		d.Set(v)
	}
	return assigned, err
}

func (m *Mapper) tryMergeContainers(d, s reflect.Value, loc string) (assigned bool, err error) {
	unwD := UnwrapAny(d)
	unwS := UnwrapAny(s)
	if IsContainer(unwD) && IsContainer(unwS) {
		return m.assignValue(unwD, unwS, locExp(loc, "+"))
	}
	return
}

func (m *Mapper) assignToInterface(d, s reflect.Value, loc string) (assigned bool, err error) {
	if d.IsValid() {
		assigned, err = m.tryMergeContainers(d, s, loc)
		if err != nil || assigned {
			return
		}

		if !d.CanSet() {
			return m.assignValue(d.Elem(), s, locInterface(loc))
		}
	}
	return m.assignToOther(d, s, loc)
}

func (m *Mapper) assignToSlice(d, s reflect.Value, loc string) (assigned bool, err error) {
	if TypeClass(s.Kind()) == SliceClass {
		if !d.CanSet() {
			return false, errNoSetValue(loc)
		}
		v := reflect.MakeSlice(d.Type(), s.Len(), s.Len())
		for i := 0; i < s.Len(); i++ {
			if a, err := m.assignValue(v.Index(i), s.Index(i), locExp(loc, strconv.Itoa(i))); err != nil {
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

func makeMap(d reflect.Value, loc string) error {
	if d.IsNil() {
		if !d.CanSet() {
			return errNoSetValue(loc)
		}
		d.Set(reflect.MakeMap(d.Type()))
	}
	return nil
}

func (m *Mapper) assignToMap(d, s reflect.Value, loc string) (assigned bool, err error) {
	switch TypeClass(s.Kind()) {
	case MapClass:
		convFn := TypeConverterFactory(s.Type().Key(), d.Type().Key())
		if convFn == nil {
			return false, errKeyTypeMismatch(loc)
		}

		if err = makeMap(d, loc); err != nil {
			return false, err
		}
		keys := s.MapKeys()
		if len(keys) > 0 {
			elemType := d.Type().Elem()
			for _, key := range keys {
				cvKey := convFn(key)
				if !cvKey.IsValid() {
					return false, errKeyTypeMismatch(locExp(loc, key.String()))
				}
				val := d.MapIndex(cvKey)
				sval := s.MapIndex(key)
				valLoc := locExp(loc, key.String())
				valAssigned, e := m.tryMergeContainers(val, sval, valLoc)
				if e != nil {
					return false, e
				}
				if !valAssigned {
					val = reflect.New(elemType).Elem()
					if _, err = m.assignValue(val, sval, valLoc); err != nil {
						return
					}
					d.SetMapIndex(cvKey, val)
				}
			}
			assigned = true
		}
	case StructClass:
		if d.Type().Elem().Kind() != reflect.Interface {
			return
		}
		convFn := TypeConverterFactory(StringType, d.Type().Key())
		if convFn == nil {
			return false, errKeyTypeMismatch(loc)
		}
		if err := makeMap(d, loc); err != nil {
			return false, err
		}
		errs := make(map[string]*structAssignErr)
		m.assignStructToMap(d, s, loc, convFn, errs)
		for _, e := range errs {
			if len(e.errs) > 0 && e.succeeded == 0 {
				return false, e.errs[0]
			}
		}
		assigned = true
	}
	return
}

func (m *Mapper) assignToStruct(d, s reflect.Value, loc string) (assigned bool, err error) {
	if !d.CanSet() {
		return false, errNoSetValue(loc)
	}
	switch TypeClass(s.Kind()) {
	case StructClass:
		if s.Type().AssignableTo(d.Type()) {
			d.Set(s)
			assigned = true
		}
	case MapClass:
		convFn := TypeConverterFactory(s.Type().Key(), StringType)
		if convFn != nil {
			errs := make(map[string]*structAssignErr)
			keys := make(map[string]*mapKeyAssign)
			for _, key := range s.MapKeys() {
				cvKey := convFn(key)
				if cvKey.IsValid() {
					keys[cvKey.String()] = &mapKeyAssign{key: key}
				}
			}
			m.assignMapToStruct(d, s, loc, keys, errs)
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
				for i := 0; i < d.NumField(); i++ {
					field := d.Type().Field(i)
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
					m := d.Field(i)
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
						return m.assignValue(d.Field(i), convFn(s), locExp(loc, field.Name))
					}
				}
			}
		}
	}
	return
}

func (m *Mapper) assignToOther(d, s reflect.Value, loc string) (assigned bool, err error) {
	switch TypeCompatibility(s.Type(), d.Type()) {
	case Assignable:
		if !d.CanSet() {
			return false, errNoSetValue(loc)
		}
		d.Set(s)
		assigned = true
	case Convertible:
		if !d.CanSet() {
			return false, errNoSetValue(loc)
		}
		d.Set(s.Convert(d.Type()))
		assigned = true
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

func (m *Mapper) assignStructToMap(d, s reflect.Value, loc string, convFn TypeConverter, errs map[string]*structAssignErr) {
	for i := 0; i < s.NumField(); i++ {
		field := s.Type().Field(i)
		info := m.ParseField(field)
		var err error
		var assignedVal reflect.Value
		if field.Type.Kind() == reflect.Struct {
			if field.Anonymous || info.Squash {
				m.assignStructToMap(d, s.Field(i), locExp(loc, field.Name), convFn, errs)
			} else {
				assignedVal = reflect.MakeMap(reflect.MapOf(StringType, InterfaceType))
				m.assignStructToMap(assignedVal, s.Field(i), locExp(loc, field.Name), convFn, errs)
			}
		} else if info.Exported && !info.Ignore && info.MapName != "" {
			v := s.Field(i)
			if !v.IsValid() || (IsEmpty(v) && info.OmitEmpty) {
				continue
			}
			var val interface{}
			pv := reflect.ValueOf(&val)
			_, err = m.assignValue(pv.Elem(), v, locExp(loc, field.Name))
			assignedVal = pv.Elem()
		}
		if assignedVal.IsValid() {
			key := convFn(reflect.ValueOf(info.MapName))
			if key.IsValid() {
				d.SetMapIndex(key, assignedVal)
			} else {
				err = errKeyTypeMismatch(locExp(loc, field.Name))
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

func (m *Mapper) assignMapToStruct(d, s reflect.Value, loc string, keys map[string]*mapKeyAssign, errs map[string]*structAssignErr) {
	for i := 0; i < d.Type().NumField(); i++ {
		field := d.Type().Field(i)
		info := m.ParseField(field)
		if (field.Anonymous || info.Squash) && field.Type.Kind() == reflect.Struct {
			m.assignMapToStruct(d.Field(i), s, locExp(loc, field.Name), keys, errs)
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
				assigned, err := m.assignValue(d.Field(i), s.MapIndex(mka.key), locExp(loc, field.Name))
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
	_, err := m.assignValue(v, s, "")
	return err
}

// Map assign values between interface{} types
func (m *Mapper) Map(v, s interface{}) error {
	return m.MapValue(reflect.ValueOf(v), reflect.ValueOf(s))
}

// Map wraps Mapper.Map with a default Mapper instance
func Map(v, s interface{}) error {
	m := &Mapper{}
	return m.Map(v, s)
}
