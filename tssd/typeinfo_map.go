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

var simpleSave = map[reflect.Kind]srcAddr{
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

func (ti *typeInfo) mapSimpleSave(value reflect.Value, buf *Buffer) error {
	s := simpleSave[ti.rtype.Kind()](value)
	return ti.memAppend(s, buf)
}

func (ti *typeInfo) mapSimpleDump(src []byte) (reflect.Value, []byte, error) {
	d := simpleDump[ti.rtype.Kind()]()
	src, _ = ti.dump(ti, src, d)
	return reflect.NewAt(ti.rtype, d).Elem(), src, nil
}

func (ti *typeInfo) mapStrSave(value reflect.Value, buf *Buffer) error {
	s := value.String()
	return ti.strSave(Ptr(&s), buf)
}

func (ti *typeInfo) mapStrDump(src []byte) (reflect.Value, []byte, error) {
	var s string

	src, _ = ti.strDump(src, Ptr(&s))
	return reflect.ValueOf(s), src, nil
}

func (ti *typeInfo) mapTimeSave(value reflect.Value, buf *Buffer) error {
	s := value.Interface().(time.Time)
	return ti.timeSave(Ptr(&s), buf)
}

func (ti *typeInfo) mapTimeDump(src []byte) (reflect.Value, []byte, error) {
	var s time.Time

	src, _ = ti.timeDump(src, Ptr(&s))
	return reflect.ValueOf(s), src, nil
}

func (ti *typeInfo) mapStructSave(value reflect.Value, buf *Buffer) error {
	buf.AppendByte(byte(ti.Type))
	index, pos := buf.writePos()
	size := buf.Size
	buf.appendSize4(0).appendSize2(len(ti.info))
	for i := range len(ti.info) {
		ti.info[i].mapSave(&ti.info[i], value.Field(i), buf)
	}
	buf.updateSize(index, pos, buf.Size-size-TSSD_SIZET_LENGTH)
	return nil
}

func (ti *typeInfo) mapStructDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < 1 {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	switch int8(src[0]) {
	case ti.Type:
		_, fields, remain, err := checkDumpSize(src)
		if err != nil {
			return v, src, ErrorInSufficientData
		}

		if fields != len(ti.info) {
			return v, src, fmt.Errorf("%w [fields mismatch %d %d]", ErrorInvalidTSSDData, len(ti.info), fields)
		}
		v = reflect.New(ti.rtype).Elem()
		var f reflect.Value
		for i := range fields {
			f, remain, err = ti.info[i].mapDump(&ti.info[i], remain)
			if err != nil {
				return v, src, err
			}
			v.Field(i).Set(f)
		}
		return v, remain, nil

	case -ti.Type:
		//skip this field
		//TODO return a error when need skip the field
		return v, src[1:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) mapSliceValueSave(value reflect.Value, buf *Buffer) error {

	arrayN := value.Len()

	buf.AppendByte(byte(ti.Type))
	index, pos := buf.writePos()
	size := buf.Size
	buf.appendSize4(0).appendSize2(arrayN)
	for i := range arrayN {
		ti.info[0].mapSave(&ti.info[0], value.Index(i), buf)
	}
	buf.updateSize(index, pos, buf.Size-size-TSSD_SIZET_LENGTH)
	return nil
}

func (ti *typeInfo) mapSliceValueDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < 1 {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	switch int8(src[0]) {
	case ti.Type:
		_, arrayN, remain, err := checkDumpSize(src)
		if err != nil {
			return v, src, ErrorInSufficientData
		}

		d := reflect.New(ti.rtype).Elem()
		if ti.rtype.Kind() == reflect.Slice {
			d.Grow(arrayN)
			d.SetLen(arrayN)
		}
		for i := 0; i < arrayN; i++ {
			v, remain, err = ti.info[0].mapDump(&ti.info[0], remain)
			d.Index(i).Set(v)
		}

		return d, remain, nil
	case -ti.Type:
		//skip this field
		return v, src[1:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) mapMergeSliceValueSave(value reflect.Value, buf *Buffer) error {

	arrayN := value.Len()
	buf.Append([]byte{byte(ti.Type), byte(ti.info[0].Type)})
	totalSize := ti.info[0].size * arrayN
	buf.appendSize4(totalSize).appendSize2(arrayN)

	for i := range arrayN {
		s := simpleSave[ti.info[0].rtype.Kind()](value.Index(i))
		buf.Append(Slice(Ptr(s), Size_t(ti.info[0].size)))
	}
	return nil
}

func (ti *typeInfo) mapMergeSliceValueDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < TSSD_TYPE_LENGTH+TSSD_TYPE_LENGTH {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	switch int8(src[0]) {
	case ti.Type: //src[0]:Tarraym, src[1] Telement
		_, arrayN, remain, err := checkDumpSize(src[TSSD_TYPE_LENGTH:])
		if err != nil {
			return v, src, ErrorInSufficientData
		}

		d := reflect.New(ti.rtype).Elem()
		if ti.rtype.Kind() == reflect.Slice {
			d.Grow(arrayN)
			d.SetLen(arrayN)
		}

		for i := 0; i < arrayN; i++ {
			obj := simpleDump[ti.info[0].rtype.Kind()]()
			copy(Slice(Ptr(obj), Size_t(ti.info[0].size)), remain[:ti.info[0].size])
			remain = remain[ti.info[0].size:]
			d.Index(i).Set(reflect.NewAt(ti.info[0].rtype, obj).Elem())
		}

		return d, remain, nil

	case -ti.Type:
		//skip this field
		return v, src[TSSD_TYPE_LENGTH:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) mapMapValueSave(value reflect.Value, buf *Buffer) error {

	arrayN := value.Len()

	buf.AppendByte(byte(ti.Type))
	index, pos := buf.writePos()
	size := buf.Size

	buf.appendSize4(0).appendSize2(arrayN)

	keys := value.MapKeys()
	for _, k := range keys {
		v := value.MapIndex(k)
		buf.AppendByte(byte(Tdictk))
		ti.info[0].mapSave(&ti.info[0], k, buf)
		buf.AppendByte(byte(Tdictv))
		ti.info[1].mapSave(&ti.info[1], v, buf)
	}
	buf.updateSize(index, pos, buf.Size-size-TSSD_SIZET_LENGTH)

	return nil
}

func (ti *typeInfo) mapMapValueDump(src []byte) (v reflect.Value, remain []byte, err error) {

	if len(src) < TSSD_TYPE_LENGTH {
		//TODO, add field name info
		return v, src, ErrorInSufficientData
	}
	//var size int
	switch int8(src[0]) {
	case ti.Type:
		_, mapLen, remain, err := checkDumpSize(src)
		if err != nil {
			return v, src, ErrorInSufficientData
		}

		mvalue := reflect.MakeMapWithSize(ti.rtype, mapLen)
		ktype := ti.rtype.Key()
		vtype := ti.rtype.Elem()

		var kk, vv reflect.Value
		for k := 0; k < mapLen; k++ {
			key := reflect.New(ktype).Elem()
			value := reflect.New(vtype).Elem()

			if remain[0] != byte(Tdictk) {
				return v, src, fmt.Errorf("%w [field type mismatch: expect %d but %d", ErrorInvalidTSSDData, Tdictk, remain[0])
			}

			kk, remain, err = ti.info[0].mapDump(&ti.info[0], remain[TSSD_TYPE_LENGTH:])
			if err != nil {
				return v, src, err
			}
			key.Set(kk.Convert(ktype))

			if remain[0] != byte(Tdictv) {
				return v, src, fmt.Errorf("%w [field type mismatch: expect %d but %d", ErrorInvalidTSSDData, Tdictv, remain[0])
			}

			vv, remain, err = ti.info[1].mapDump(&ti.info[1], remain[TSSD_TYPE_LENGTH:])
			if err != nil {
				return v, src, err
			}
			value.Set(vv.Convert(value.Type()))

			mvalue.SetMapIndex(key, value)
		}
		return mvalue, remain, nil

		//reflect.NewAt(ti.rtype, dest).Elem().Set(mvalue)

	case -ti.Type:
		//skip this field
		return v, src[TSSD_TYPE_LENGTH:], nil
	default:
		return v, src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
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
