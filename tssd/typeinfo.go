package tssd

import (
	//"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

type Ptr = unsafe.Pointer
type Size_t = uintptr
type saveFunc = func(*typeInfo, Ptr, *Buffer) error
type dumpFunc = func(*typeInfo, *Buffer, Ptr) error
type mapSave = func(*typeInfo, reflect.Value, *Buffer) error   //map save func
type mapDump = func(*typeInfo, *Buffer) (reflect.Value, error) //map dump func

type typeInfo struct {
	rtype         reflect.Type
	Type          int8       //tssd type
	info          []typeInfo //nest typeInfo
	save          saveFunc
	dump          dumpFunc
	size          int
	offset        Size_t
	name          string
	stype         []byte //all the type stream, includeing fields
	isFixedLength bool
	mapSave
	mapDump
	root *typeInfo //typeInfo root
}

func (info *typeInfo) types() []byte {
	return info.stype
}

func (ti *typeInfo) memAppend(src Ptr, buf *Buffer) error {
	buf.AppendByte(byte(ti.Type)).Append(Slice(src, Size_t(ti.size)))
	return nil
}

func (ti *typeInfo) memDump(buf *Buffer, dest Ptr) error {
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	switch int8(b) {
	case ti.Type:
		if buf.Size < ti.size {
			//TODO, add field name info
			return ErrorInSufficientData
		}
		buf.Read(Slice(dest, Size_t(ti.size)))
	case -ti.Type:
		//skip this field
		return nil
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return nil
}

func (ti *typeInfo) timeSave(src Ptr, buf *Buffer) error {
	p := (*time.Time)(src)
	buf.Append([]byte{byte(ti.Type), byte(Tstring)}).appendString(p.Format(time.RFC3339Nano))
	return nil
}

// dump string directly
func stringDump(buf *Buffer, dest *string) error {
	sizet, err := buf.checkDumpSizet()
	if err != nil {
		return err
	}
	bs, _ := buf.Read(make([]byte, sizet))
	*dest = string(bs)
	return nil
}

func (ti *typeInfo) timeDump(buf *Buffer, dest Ptr) error {
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	switch int8(b) {
	case ti.Type:
		b, err = buf.ReadByte()
		if err != nil {
			return err
		}
		if b != byte(Tstring) {
			return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, Tstring)
		}
		var rfc3339Str string
		if err = stringDump(buf, &rfc3339Str); err != nil {
			return err
		}
		t, err := time.Parse(time.RFC3339Nano, rfc3339Str)
		if err != nil {
			return err
		}
		p := (*time.Time)(dest)
		*p = t
	case -ti.Type:
		//skip this field
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return nil
}

func (ti *typeInfo) strSave(src Ptr, buf *Buffer) error {
	p := (*string)(src)
	buf.AppendByte(byte(ti.Type)).appendString(*p)
	return nil
}

func (ti *typeInfo) strDump(buf *Buffer, dest Ptr) error {
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	switch int8(b) {
	case ti.Type:
		if err = stringDump(buf, (*string)(dest)); err != nil {
			return err
		}
	case -ti.Type:
		//skip this field
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return nil
}

func (ti *typeInfo) objSave(src Ptr, buf *Buffer) error {

	buf.AppendByte(byte(ti.Type))

	index, pos := buf.writePos()
	size := buf.Size
	buf.appendSize4(0).appendSize2(len(ti.info))
	for i := range len(ti.info) {
		ti.info[i].save(&ti.info[i], Ptr(Size_t(src)+ti.info[i].offset), buf)
	}
	buf.updateSize(index, pos, buf.Size-size-TSSD_SIZET_LENGTH)
	return nil
}

func (ti *typeInfo) objDump(buf *Buffer, dest Ptr) error {
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	switch int8(b) {
	case ti.Type:
		_, fields, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		if fields != len(ti.info) {
			return fmt.Errorf("%w [fields mismatch %d %d]", ErrorInvalidTSSDData, len(ti.info), fields)
		}

		for i := range fields {
			if err = ti.info[i].dump(&ti.info[i], buf, Ptr(Size_t(dest)+ti.info[i].offset)); err != nil {
				return err
			}
		}
	case -ti.Type:
		//skip this field
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return nil
}

func (ti *typeInfo) sliceSave(src Ptr, buf *Buffer) error {
	arrayN := ti.size
	addr := Size_t(src)
	if ti.rtype.Kind() == reflect.Slice {
		p := *(*[]byte)(src)
		if arrayN = len(p); arrayN > 0 {
			addr = Size_t(Ptr(&p[0]))
		}
	}

	buf.AppendByte(byte(ti.Type))

	index, pos := buf.writePos()
	size := buf.Size

	buf.appendSize4(0).appendSize2(arrayN)
	for i := range arrayN {
		ti.info[0].save(&ti.info[0], Ptr(addr+Size_t(ti.info[0].size*i)), buf)
	}
	buf.updateSize(index, pos, buf.Size-size-TSSD_SIZET_LENGTH)
	return nil
}

func (ti *typeInfo) sliceDump(buf *Buffer, dest Ptr) error {
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	switch int8(b) {
	case ti.Type:
		_, arrayN, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		addr := dest
		if ti.rtype.Kind() == reflect.Slice {
			p := (*[]byte)(dest)
			if cap(*p) < arrayN { //for slice, need pre-alloc
				*p = make([]byte, arrayN*ti.info[0].size)
			}

			*p = (*p)[0:arrayN] //set size
			if arrayN > 0 {
				addr = Ptr(&((*p)[0]))
			}
		}

		for i := range arrayN {
			if err = ti.info[0].dump(&ti.info[0], buf, Ptr(Size_t(addr)+Size_t(ti.info[0].size*i))); err != nil {
				return err
			}
		}
	case -ti.Type:
		//skip this field
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return nil
}

// [Tarraym][Ttype][size][arrayN][data]
func (ti *typeInfo) mergeSliceSave(src Ptr, buf *Buffer) error {
	arrayN := ti.size
	addr := Size_t(src)
	if ti.rtype.Kind() == reflect.Slice {
		p := *(*[]byte)(src)
		if arrayN = len(p); arrayN > 0 {
			addr = Size_t(Ptr(&p[0]))
		}
	}

	buf.Append([]byte{byte(ti.Type), byte(ti.info[0].Type)})
	totalSize := ti.info[0].size * arrayN
	buf.appendSize4(TSSD_SIZEA_LENGTH + totalSize).appendSize2(arrayN)

	buf.Append(Slice(Ptr(addr), Size_t(totalSize)))
	return nil
}

func (ti *typeInfo) mergeSliceDump(buf *Buffer, dest Ptr) error {
	b, err := buf.Read(make([]byte, 2))
	if err != nil {
		return err
	}
	switch int8(b[0]) {
	case ti.Type: //[0]: Tarraym, [1]: elementType
		if int8(b[1]) != ti.info[0].Type {
			return fmt.Errorf("%w [element type mismatch %d %d]", ErrorInvalidTSSDData, b[1], ti.info[0].Type)
		}
		_, arrayN, err := buf.checkDumpSize()
		if err != nil {
			return err
		}
		addr := dest
		if ti.rtype.Kind() == reflect.Slice {
			p := (*[]byte)(dest)
			if cap(*p) < arrayN { //for slice, need pre-alloc
				*p = make([]byte, arrayN*ti.info[0].size)
			}

			*p = (*p)[0:arrayN] //set size
			if arrayN > 0 {
				addr = Ptr(&((*p)[0]))
			}
		}

		//TODO, for big-endian, we need copy one by one
		buf.Read(Slice(addr, Size_t(arrayN*ti.info[0].size)))
	case -ti.Type:
		//skip this field
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b[0], ti.Type)
	}
	return nil
}

func (ti *typeInfo) dictSave(src Ptr, buf *Buffer) error {

	value := reflect.NewAt(ti.rtype, src).Elem()
	keys := value.MapKeys()

	buf.AppendByte(byte(ti.Type))

	index, pos := buf.writePos()
	size := buf.Size
	buf.appendSize4(0).appendSize2(len(keys))

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

func (ti *typeInfo) dictDump(buf *Buffer, dest Ptr) error {
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	switch int8(b) {
	case ti.Type:
		_, mapLen, err := buf.checkDumpSize()
		if err != nil {
			return err
		}

		mvalue := reflect.MakeMapWithSize(ti.rtype, mapLen)
		ktype := ti.rtype.Key()
		vtype := ti.rtype.Elem()

		var kk, vv reflect.Value
		for k := 0; k < mapLen; k++ {
			key := reflect.New(ktype).Elem()
			value := reflect.New(vtype).Elem()
			b, err = buf.ReadByte()
			if err != nil {
				return err
			}
			if b != byte(Tdictk) {
				return fmt.Errorf("%w [map field type mismatch: %d %d", ErrorInvalidTSSDData, b, Tdictk)
			}

			kk, err = ti.info[0].mapDump(&ti.info[0], buf)
			if err != nil {
				return err
			}
			key.Set(kk.Convert(ktype))

			b, err = buf.ReadByte()
			if err != nil {
				return err
			}
			if b != byte(Tdictv) {
				return fmt.Errorf("%w [map field type mismatch: %d %d", ErrorInvalidTSSDData, b, Tdictv)
			}

			vv, err = ti.info[1].mapDump(&ti.info[1], buf)
			if err != nil {
				return err
			}
			value.Set(vv.Convert(value.Type()))

			mvalue.SetMapIndex(key, value)
		}
		reflect.NewAt(ti.rtype, dest).Elem().Set(mvalue)
	case -ti.Type:
		//skip this field
	default:
		return fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, b, ti.Type)
	}
	return nil
}

func (ti *typeInfo) marshal(src any) (*Buffer, error) {
	buf := &Buffer{}
	err := ti.marshalTo(src, buf)
	return buf, err
}

func (ti *typeInfo) marshalTo(src any, buf *Buffer) error {
	value := reflect.ValueOf(src)
	obj := value.Pointer()

	return ti.save(ti, Ptr(obj), buf)
}

func (ti *typeInfo) unmarshal(buf *Buffer, dest any) error {
	return ti.unmarshalTo(buf, dest)
}

func (ti *typeInfo) unmarshalTo(buf *Buffer, dest any) error {
	value := reflect.ValueOf(dest)
	obj := value.Pointer()
	return ti.dump(ti, buf, Ptr(obj))
}

func toTSSDType(kind reflect.Kind) (typee int8) {
	switch kind {
	case reflect.Bool:
		typee = Tbool
	case reflect.Int8:
		typee = Tint8
	case reflect.Uint8:
		typee = Tuint8
	case reflect.Int16:
		typee = Tint16
	case reflect.Uint16:
		typee = Tuint16
	case reflect.Int32:
		typee = Tint32
	case reflect.Uint32:
		typee = Tuint32
	case reflect.Int64, reflect.Int:
		typee = Tint64
	case reflect.Uint64, reflect.Uint:
		typee = Tuint64
	case reflect.Float32:
		typee = Tfloat32
	case reflect.Float64:
		typee = Tfloat64
	default:
		fmt.Println("error: invalid type:", kind)
		//TODO
	}
	return typee
}

func (ti *typeInfo) updateType(typ int8, pos int) {
	ti.Type = typ
	ti.root.stype[pos] = byte(typ)
}

// set type
// append to types
// return the prevous size, we can update it later if needed
func (ti *typeInfo) setType(typ int8) (pos int) {
	ti.Type = typ
	pos = len(ti.root.stype)
	ti.root.stype = append(ti.root.stype, byte(ti.Type))
	if typ == Ttime {
		ti.root.stype = append(ti.root.stype, byte(Tstring)) //for Ttime, add subtype
	}
	return pos
}

func (ti *typeInfo) doParse(intf interface{}) *typeInfo {

	field := reflect.TypeOf(intf)
	value := reflect.ValueOf(intf)

	//ti.typee = value.Kind()
	ti.rtype = field
	ti.size = int(field.Size())
	if strings.HasPrefix(field.String(), TSSD_FLAT_KIND) {
		return nil
	}

	switch value.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64:
		ti.save = (*typeInfo).memAppend
		ti.dump = (*typeInfo).memDump
		ti.setType(toTSSDType(field.Kind()))
		ti.isFixedLength = true
		ti.mapSave, ti.mapDump = (*typeInfo).mapSimpleSave, (*typeInfo).mapSimpleDump

	case reflect.String:
		ti.save = (*typeInfo).strSave
		ti.dump = (*typeInfo).strDump
		ti.setType(Tstring)
		ti.mapSave, ti.mapDump = (*typeInfo).mapStrSave, (*typeInfo).mapStrDump

	case reflect.Struct:
		//process time
		if field.String() == TSSD_TIME_KIND {
			ti.setType(Ttime)
			ti.save = (*typeInfo).timeSave
			ti.dump = (*typeInfo).timeDump
			ti.mapSave, ti.mapDump = (*typeInfo).mapTimeSave, (*typeInfo).mapTimeDump
			return ti
		}

		ti.save = (*typeInfo).objSave
		ti.dump = (*typeInfo).objDump
		ti.mapSave, ti.mapDump = (*typeInfo).mapStructSave, (*typeInfo).mapStructDump

		fields := reflect.TypeOf(intf)
		num := fields.NumField()

		ti.setType(Tobject)

		//we append struct's fields to validate, but exclude Flat self
		pos := len(ti.root.stype)
		ti.root.stype = appendSize2(ti.root.stype, num)

		ti.info = make([]typeInfo, num)
		var j = 0
		for i := 0; i < num; i++ {
			ti.info[j].root = ti.root
			if (&ti.info[j]).doParse(value.Field(i).Interface()) == nil {
				continue
			}

			ti.info[j].offset = fields.Field(i).Offset
			ti.info[j].name = fields.Field(i).Name
			j++
		}
		ti.info = ti.info[:j]
		appendSize2(ti.root.stype[:pos], j) // update fields num to skip Flat self

	case reflect.Array, reflect.Slice: //for array, the memorry is continus:  &array==&array[0]
		pos := ti.setType(Tarray)
		ti.save = (*typeInfo).sliceSave
		ti.dump = (*typeInfo).sliceDump
		ti.size = value.Len()
		ti.mapSave, ti.mapDump = (*typeInfo).mapSliceValueSave, (*typeInfo).mapSliceValueDump

		//TODO: we need disable []any
		ti.info = make([]typeInfo, 1)
		ti.info[0].root = ti.root
		v := value.Type().Elem()
		if (&ti.info[0]).doParse(reflect.New(v).Elem().Interface()) == nil {
			ti.info = ti.info[:0]
			return ti
		}

		if ti.info[0].isFixedLength {
			ti.updateType(Tarraym, pos)
			ti.save = (*typeInfo).mergeSliceSave
			ti.dump = (*typeInfo).mergeSliceDump
			ti.mapSave, ti.mapDump = (*typeInfo).mapMergeSliceValueSave, (*typeInfo).mapMergeSliceValueDump
		}

	case reflect.Map:
		ti.setType(Tdict)
		ti.save = (*typeInfo).dictSave
		ti.dump = (*typeInfo).dictDump
		ti.mapSave, ti.mapDump = (*typeInfo).mapMapValueSave, (*typeInfo).mapMapValueDump
		ti.info = make([]typeInfo, 2)
		k := value.Type().Key()
		v := value.Type().Elem()
		ti.info[0].root, ti.info[1].root = ti.root, ti.root
		index := 0
		if (&ti.info[index]).doParse(reflect.New(k).Elem().Interface()) != nil {
			index++
		}
		if (&ti.info[index]).doParse(reflect.New(v).Elem().Interface()) != nil {
			index++
		}
		ti.info = ti.info[:index]

	default:
		fmt.Printf("Not support data type: %d %s\n", value.Kind(), ti.rtype.String())
		//continue
		//return nil
	}
	return ti

}

func parse(intf interface{}) (ti *typeInfo) {

	ti = &typeInfo{stype: make([]byte, 0, 1024)}
	ti.root = ti

	return ti.doParse(intf)
}
