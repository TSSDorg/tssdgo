package tssd

import (
	"fmt"
	"reflect"
	"time"
)

type srcAddr = func(reflect.Value) Ptr

func intSrcAddr(value reflect.Value) Ptr {
	s := value.Int()
	return Ptr(&s)
}
func uintSrcAddr(value reflect.Value) Ptr {
	s := value.Uint()
	return Ptr(&s)
}

var simpleSave map[reflect.Kind]srcAddr = map[reflect.Kind]srcAddr{
	reflect.Bool: func(value reflect.Value) Ptr {
		s := value.Bool()
		return Ptr(&s)
	},
	reflect.Int:    intSrcAddr,
	reflect.Uint:   uintSrcAddr,
	reflect.Int8:   intSrcAddr,
	reflect.Uint8:  uintSrcAddr,
	reflect.Int16:  intSrcAddr,
	reflect.Uint16: uintSrcAddr,
	reflect.Int32:  intSrcAddr,
	reflect.Uint32: uintSrcAddr,
	reflect.Int64:  intSrcAddr,
	reflect.Uint64: uintSrcAddr,
	reflect.Float32: func(value reflect.Value) Ptr {
		s := float32(value.Float())
		return Ptr(&s)
	},
	reflect.Float64: func(value reflect.Value) Ptr {
		s := value.Float()
		return Ptr(&s)
	},
}

type destAddr = func() Ptr

func intDestAddr() Ptr {
	s := new(int)
	return Ptr(s)
}

func uintDestAddr() Ptr {
	s := new(uint)
	return Ptr(s)
}

var simpleDump map[reflect.Kind]destAddr = map[reflect.Kind]destAddr{
	reflect.Bool: func() Ptr {
		s := new(bool)
		return Ptr(s)
	},
	reflect.Int:    intDestAddr,
	reflect.Uint:   uintDestAddr,
	reflect.Int8:   intDestAddr,
	reflect.Uint8:  uintDestAddr,
	reflect.Int16:  intDestAddr,
	reflect.Uint16: uintDestAddr,
	reflect.Int32:  intDestAddr,
	reflect.Uint32: uintDestAddr,
	reflect.Int64:  intDestAddr,
	reflect.Uint64: uintDestAddr,
	reflect.Float32: func() Ptr {
		s := new(float32)
		return Ptr(s)
	},
	reflect.Float64: func() Ptr {
		s := new(float64)
		return Ptr(s)
	},
}

func (ti *typeInfo) mapSimpleSave(value reflect.Value, dest []byte) ([]byte, error) {
	s := simpleSave[ti.rtype.Kind()](value)
	return ti.memAppend(s, dest)
}

func (ti *typeInfo) mapSimpleDump(src []byte) (reflect.Value, []byte, error) {
	d := simpleDump[ti.rtype.Kind()]()
	src, _ = ti.dump(ti, src, d)
	return reflect.NewAt(ti.rtype, d).Elem(), src, nil
}

func (ti *typeInfo) mapStrSave(value reflect.Value, dest []byte) ([]byte, error) {
	s := value.String()
	return ti.strSave(Ptr(&s), dest)
}

func (ti *typeInfo) mapStrDump(src []byte) (reflect.Value, []byte, error) {
	var s string

	src, _ = ti.strDump(src, Ptr(&s))
	return reflect.ValueOf(s), src, nil
}

func (ti *typeInfo) mapTimeSave(value reflect.Value, dest []byte) ([]byte, error) {
	s := value.Interface().(time.Time)
	return ti.timeSave(Ptr(&s), dest)
}

func (ti *typeInfo) mapTimeDump(src []byte) (reflect.Value, []byte, error) {
	var s time.Time

	src, _ = ti.timeDump(src, Ptr(&s))
	return reflect.ValueOf(s), src, nil
}

func (ti *typeInfo) mapStructSave(value reflect.Value, dest []byte) ([]byte, error) {
	dest = append(dest, byte(ti.Type)) //T
	sizePos := len(dest)
	dest = appendSize(dest, 0)            //reserved total Size (S)
	dest = appendSize(dest, len(ti.info)) //S
	//fmt.Println("objSave dest:", dest)
	for i := range len(ti.info) {
		dest, _ = ti.info[i].mapSave(&ti.info[i], value.Field(i), dest)
	}
	appendSize(dest[:sizePos], len(dest)-sizePos-2) //object size exclude size self(2bytes)
	return dest, nil
}

func (ti *typeInfo) mapStructDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < 1 {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	var size, fields int
	switch int8(src[0]) {
	case ti.Type:
		if len(src) < 3 {
			//TODO, add field name info
			return v, src, ErrorInSufficientData
		}
		size = int(dumpSize(src[1:]))
		if len(src) < 3+size {
			//TODO, add field name info
			return v, src, ErrorInSufficientData
		}

		fields = int(dumpSize(src[3:]))
		if fields != len(ti.info) {
			return v, src, fmt.Errorf("%w [fields mismatch %d %d]", ErrorInvalidTSSDData, len(ti.info), fields)
		}
		r := src[5:]
		v = reflect.New(ti.rtype).Elem()

		var f reflect.Value

		for i := range fields {
			f, r, err = ti.info[i].mapDump(&ti.info[i], r)
			if err != nil {
				return v, src, err
			}
			v.Field(i).Set(f)
		}

	case -ti.Type:
		//skip this field
		//TODO return a error when need skip the field
		return v, src[1:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
	return v, src[3+size:], nil
}

func (ti *typeInfo) mapSliceValueSave(value reflect.Value, dest []byte) ([]byte, error) {

	arrayN := value.Len()

	dest = append(dest, byte(ti.Type)) //T
	sizePos := len(dest)
	dest = appendSize(dest, 0)      //reserved total Size (S)
	dest = appendSize(dest, arrayN) //S
	for i := range arrayN {
		dest, _ = ti.info[0].mapSave(&ti.info[0], value.Index(i), dest)
	}
	appendSize(dest[:sizePos], len(dest)-sizePos-2) //object size exclude size self(2bytes)
	return dest, nil
}

func (ti *typeInfo) mapSliceValueDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < 1 {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	var size int
	switch int8(src[0]) {
	case ti.Type:
		if len(src) < 3 {
			//TODO, add field name info
			return v, src, ErrorInSufficientData
		}
		size = int(dumpSize(src[1:]))
		if len(src) < 3+size {
			//TODO, add field name info
			return v, src, ErrorInSufficientData
		}

		arrayN := int(dumpSize(src[3:]))
		d := reflect.New(ti.rtype).Elem()
		if ti.rtype.Kind() == reflect.Slice {

			d.Grow(arrayN)
			d.SetLen(arrayN)
		}

		r := src[5:]
		for i := 0; i < arrayN; i++ {
			v, r, err = ti.info[0].mapDump(&ti.info[0], r)
			d.Index(i).Set(v)
		}

		return d, r, nil

	case -ti.Type:
		//skip this field
		return v, src[1:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
	//return v, src[3+size:], nil
}

func (ti *typeInfo) mapMergeSliceValueSave(value reflect.Value, dest []byte) ([]byte, error) {

	arrayN := value.Len()

	dest = append(dest, byte(ti.Type))         //TmergeArray
	dest = append(dest, byte(ti.info[0].Type)) //Telement

	totalSize := ti.info[0].size * arrayN
	dest = appendSize(dest, totalSize) //total Size
	dest = appendSize(dest, arrayN)    //S

	for i := range arrayN {
		s := simpleSave[ti.info[0].rtype.Kind()](value.Index(i))
		dest = append(dest, Slice(Ptr(s), Size_t(ti.info[0].size))...)
	}
	return dest, nil
}

func (ti *typeInfo) mapMergeSliceValueDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < 1 {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	var size int
	switch int8(src[0]) {
	case ti.Type: //src[0]:TmergeArray, src[1] Telement
		if len(src) < 4 {
			//TODO, add field name info
			return v, src, ErrorInSufficientData
		}
		size = int(dumpSize(src[2:]))
		if len(src) < 4+size {
			//TODO, add field name info
			return v, src, ErrorInSufficientData
		}

		arrayN := int(dumpSize(src[4:]))
		d := reflect.New(ti.rtype).Elem()
		if ti.rtype.Kind() == reflect.Slice {
			d.Grow(arrayN)
			d.SetLen(arrayN)
		}

		r := src[6:]
		for i := 0; i < arrayN; i++ {
			obj := simpleDump[ti.info[0].rtype.Kind()]()
			copy(Slice(Ptr(obj), Size_t(ti.info[0].size)), r[:ti.info[0].size])
			r = r[ti.info[0].size:]
			d.Index(i).Set(reflect.NewAt(ti.info[0].rtype, obj).Elem())
		}

		return d, r, nil

	case -ti.Type:
		//skip this field
		return v, src[1:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
	//return v, src[3+size:], nil
}

func MakeMap(intf interface{}) reflect.Value {
	v := reflect.ValueOf(intf)
	mtyp := v.Type().Elem()

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.IsNil() {
		// Allocate map
		v.Set(reflect.MakeMap(mtyp))
	}

	return v
}
