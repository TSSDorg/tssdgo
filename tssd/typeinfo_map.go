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

func (ti *typeInfo) mapSimpleDump(buf *Buffer) (reflect.Value, error) {
	d := simpleDump[ti.rtype.Kind()]()
	err := ti.dump(ti, buf, d)
	return reflect.NewAt(ti.rtype, d).Elem(), err
}

func (ti *typeInfo) mapStrSave(value reflect.Value, buf *Buffer) error {
	s := value.String()
	return ti.strSave(Ptr(&s), buf)
}

func (ti *typeInfo) mapStrDump(buf *Buffer) (reflect.Value, error) {
	var s string
	err := ti.strDump(buf, Ptr(&s))
	return reflect.ValueOf(s), err
}

func (ti *typeInfo) mapTimeSave(value reflect.Value, buf *Buffer) error {
	s := value.Interface().(time.Time)
	return ti.timeSave(Ptr(&s), buf)
}

func (ti *typeInfo) mapTimeDump(buf *Buffer) (reflect.Value, error) {
	var s time.Time
	err := ti.timeDump(buf, Ptr(&s))
	return reflect.ValueOf(s), err
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

func (ti *typeInfo) mapStructDump(buf *Buffer) (v reflect.Value, err error) {
	b, err := buf.ReadByte()
	if err != nil {
		return v, err
	}
	switch int8(b) {
	case ti.Type:
		_, fields, err := buf.checkDumpSize()
		if err != nil {
			return v, err
		}
		if fields != len(ti.info) {
			return v, fmt.Errorf("%w [fields mismatch %d %d]", ErrorInvalidTSSDData, fields, len(ti.info))
		}
		v = reflect.New(ti.rtype).Elem()
		var f reflect.Value
		for i := range fields {
			f, err = ti.info[i].mapDump(&ti.info[i], buf)
			if err != nil {
				return v, err
			}
			v.Field(i).Set(f)
		}
	case -ti.Type:
		//skip this field
	default:
		return v, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return v, nil
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

func (ti *typeInfo) mapSliceValueDump(buf *Buffer) (v reflect.Value, err error) {
	b, err := buf.ReadByte()
	if err != nil {
		return v, err
	}
	switch int8(b) {
	case ti.Type:
		_, arrayN, err := buf.checkDumpSize()
		if err != nil {
			return v, err
		}

		v = reflect.New(ti.rtype).Elem()
		if ti.rtype.Kind() == reflect.Slice {
			v.Grow(arrayN)
			v.SetLen(arrayN)
		}
		for i := 0; i < arrayN; i++ {
			ve, err := ti.info[0].mapDump(&ti.info[0], buf)
			if err != nil {
				return v, err
			}
			v.Index(i).Set(ve)
		}
	case -ti.Type:
		//skip this field
	default:
		return v, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return v, nil
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

func (ti *typeInfo) mapMergeSliceValueDump(buf *Buffer) (v reflect.Value, err error) {
	b, err := buf.Read(make([]byte, 2))
	if err != nil {
		return v, err
	}
	switch int8(b[0]) {
	case ti.Type: //src[0]:Tarraym, src[1] Telement
		if int8(b[1]) != ti.info[0].Type {
			return v, fmt.Errorf("%w [element type mismatch %d %d]", ErrorInvalidTSSDData, b[1], ti.info[0].Type)
		}
		_, arrayN, err := buf.checkDumpSize()
		if err != nil {
			return v, err
		}

		v = reflect.New(ti.rtype).Elem()
		if ti.rtype.Kind() == reflect.Slice {
			v.Grow(arrayN)
			v.SetLen(arrayN)
		}

		for i := 0; i < arrayN; i++ {
			obj := simpleDump[ti.info[0].rtype.Kind()]()
			buf.Read(Slice(Ptr(obj), Size_t(ti.info[0].size)))
			v.Index(i).Set(reflect.NewAt(ti.info[0].rtype, obj).Elem())
		}
	case -ti.Type:
		//skip this field
	default:
		return v, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b[0], ti.Type)
	}
	return v, nil
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

func (ti *typeInfo) mapMapValueDump(buf *Buffer) (v reflect.Value, err error) {
	b, err := buf.ReadByte()
	if err != nil {
		return v, err
	}
	//var size int
	switch int8(b) {
	case ti.Type:
		_, mapLen, err := buf.checkDumpSize()
		if err != nil {
			return v, err
		}

		v = reflect.MakeMapWithSize(ti.rtype, mapLen)
		ktype := ti.rtype.Key()
		vtype := ti.rtype.Elem()

		var kk, vv reflect.Value
		for k := 0; k < mapLen; k++ {
			key := reflect.New(ktype).Elem()
			value := reflect.New(vtype).Elem()
			b, err = buf.ReadByte()
			if err != nil {
				return v, err
			}
			if b != byte(Tdictk) {
				return v, fmt.Errorf("%w [map field type mismatch: %d %d", ErrorInvalidTSSDData, b, Tdictk)
			}

			kk, err = ti.info[0].mapDump(&ti.info[0], buf)
			if err != nil {
				return v, err
			}
			key.Set(kk.Convert(ktype))
			b, err = buf.ReadByte()
			if err != nil {
				return v, err
			}
			if b != byte(Tdictv) {
				return v, fmt.Errorf("%w [map field type mismatch: %d %d", ErrorInvalidTSSDData, b, Tdictv)
			}

			vv, err = ti.info[1].mapDump(&ti.info[1], buf)
			if err != nil {
				return v, err
			}
			value.Set(vv.Convert(value.Type()))

			v.SetMapIndex(key, value)
		}
	case -ti.Type:
		//skip this field
		return v, nil
	default:
		return v, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return v, nil
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
