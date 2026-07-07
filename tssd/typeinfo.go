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
type saveFunc = func(*typeInfo, Ptr, []byte) ([]byte, error)
type dumpFunc = func(*typeInfo, []byte, Ptr) ([]byte, error)
type mapSave = func(*typeInfo, reflect.Value, []byte) ([]byte, error) //map save func
type mapDump = func(*typeInfo, []byte) (reflect.Value, []byte, error) //map dump func

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

// put a binary content into a slice
// yeah! convert everything to byte slice
func Slice(k Ptr, size Size_t) []byte {
	p := (*[1<<31 - 1]byte)(k) //yeah, it's magic number, which is maxnum can be accept by golang compiler
	return (*p)[0:size]
}

func appendSize2(dest []byte, le int) []byte {
	l := uint16(le)
	return append(dest, Slice(Ptr(&l), unsafe.Sizeof(l))...)
}

func dumpSize2(src []byte) (size int) {
	copy(Slice(Ptr(&size), TSSD_SIZEA_LENGTH), src)
	return size
}

func appendSize4(dest []byte, le int) []byte {
	l := uint32(le)
	return append(dest, Slice(Ptr(&l), unsafe.Sizeof(l))...)
}

func dumpSize4(src []byte) (size int) {
	copy(Slice(Ptr(&size), TSSD_SIZET_LENGTH), src)
	return size
}

// check and dump sizet
func checkDumpSizet(src []byte) (sizet int, remain []byte, err error) {
	if len(src) < TSSD_TYPE_LENGTH+TSSD_SIZET_LENGTH {
		//TODO, add field name info
		return 0, src, ErrorInSufficientData
	}
	sizet = dumpSize4(src[TSSD_TYPE_LENGTH:])
	if len(src) < TSSD_TYPE_LENGTH+TSSD_SIZET_LENGTH+sizet {
		//TODO, add field name info
		return 0, src, ErrorInSufficientData
	}

	return sizet, src[TSSD_TYPE_LENGTH+TSSD_SIZET_LENGTH:], nil
}

// check and dump sizet, sizea
func checkDumpSize(src []byte) (sizet int, sizea int, remain []byte, err error) {
	if sizet, remain, err = checkDumpSizet(src); err != nil {
		return
	}
	//we have check total size in checkDumpSizet, so dump sizea directly
	sizea = dumpSize2(remain)

	return sizet, sizea, remain[TSSD_SIZEA_LENGTH:], nil
}

func appendString(dest []byte, s string) []byte {
	return append(appendSize4(dest, len(s)), s...)
}

func (info *typeInfo) types() []byte {
	return info.stype
}

func (ti *typeInfo) memAppend(src Ptr, dest []byte) ([]byte, error) {
	dest = append(dest, byte(ti.Type))
	//dest = dest[0:len(dest)+ti.size]
	return append(dest, Slice(src, Size_t(ti.size))...), nil
	//copy(dest[len(dest):], Slice(src, Size_t(ti.size)))
	//return dest, nil
}

func (ti *typeInfo) memDump(src []byte, dest Ptr) ([]byte, error) {
	if len(src) < 1 {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}

	switch int8(src[0]) {
	case ti.Type:
		if len(src) < 1+ti.size {
			//TODO, add field name info
			return src, ErrorInSufficientData
		}
		copy(Slice(dest, Size_t(ti.size)), src[1:1+ti.size])
	case -ti.Type:
		//skip this field
		return src[TSSD_TYPE_LENGTH:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
	return src[TSSD_TYPE_LENGTH+ti.size:], nil
}

func (ti *typeInfo) timeSave(src Ptr, dest []byte) ([]byte, error) {
	p := (*time.Time)(src)

	dest = append(dest, byte(ti.Type))
	dest = append(dest, byte(Tstring))
	return appendString(dest, p.Format(time.RFC3339Nano)), nil
}

func (ti *typeInfo) timeDump(src []byte, dest Ptr) ([]byte, error) {
	if len(src) < TSSD_TYPE_LENGTH + TSSD_TYPE_LENGTH {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}

	switch int8(src[0]) {
	case ti.Type:
		if src[TSSD_TYPE_LENGTH] != byte(Tstring) {
			return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
		}
		sizet, remain, err := checkDumpSizet(src[TSSD_TYPE_LENGTH:])
		if err != nil {
			return src, err
		}
		p := (*time.Time)(dest)
		rfc3339Str := string(remain[:sizet])
		t, err := time.Parse(time.RFC3339Nano, rfc3339Str)
		if err != nil {
			return src, err
		}
		*p = t
		return remain[sizet:], nil
	case -ti.Type:
		//skip this field
		return src[TSSD_TYPE_LENGTH:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) strSave(src Ptr, dest []byte) ([]byte, error) {
	p := (*string)(src)

	dest = append(dest, byte(ti.Type))
	return appendString(dest, *p), nil
}

func (ti *typeInfo) strDump(src []byte, dest Ptr) ([]byte, error) {
	if len(src) < TSSD_TYPE_LENGTH {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}
	switch int8(src[0]) {
	case ti.Type:
		sizet, remain, err := checkDumpSizet(src)
		if err != nil {
			return src, ErrorInSufficientData
		}
		p := (*string)(dest)
		*p = string(remain[:sizet])
		return remain[sizet:], nil
	case -ti.Type:
		//skip this field
		return src[TSSD_TYPE_LENGTH:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) objSave(src Ptr, dest []byte) ([]byte, error) {
	dest = append(dest, byte(ti.Type)) //T
	sizePos := len(dest)
	dest = appendSize4(dest, 0)            //reserved total Size (S)
	dest = appendSize2(dest, len(ti.info)) //S
	for i := range len(ti.info) {
		dest, _ = ti.info[i].save(&ti.info[i], Ptr(Size_t(src)+ti.info[i].offset), dest)
	}
	appendSize4(dest[:sizePos], len(dest)-sizePos-TSSD_SIZET_LENGTH)

	return dest, nil
}

func (ti *typeInfo) objDump(src []byte, dest Ptr) (remain []byte, err error) {
	if len(src) < TSSD_TYPE_LENGTH {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}
	switch int8(src[0]) {
	case ti.Type:
		_, fields, remain, err := checkDumpSize(src)
		if err != nil {
			return src, ErrorInSufficientData
		}

		if fields != len(ti.info) {
			return src, fmt.Errorf("%w [fields mismatch %d %d]", ErrorInvalidTSSDData, len(ti.info), fields)
		}

		for i := range fields {
			if remain, err = ti.info[i].dump(&ti.info[i], remain, Ptr(Size_t(dest)+ti.info[i].offset)); err != nil {
				return src, err
			}
		}
		return remain, nil

	case -ti.Type:
		//skip this field
		return src[TSSD_TYPE_LENGTH:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) sliceSave(src Ptr, dest []byte) ([]byte, error) {
	arrayN := ti.size
	addr := Size_t(src)
	if ti.rtype.Kind() == reflect.Slice {
		p := *(*[]byte)(src)
		if arrayN = len(p); arrayN > 0 {
			addr = Size_t(Ptr(&p[0]))
		}
	}

	dest = append(dest, byte(ti.Type)) //T
	sizePos := len(dest)
	dest = appendSize4(dest, 0)      //reserved total Size (S)
	dest = appendSize2(dest, arrayN) //S

	for i := range arrayN {
		dest, _ = ti.info[0].save(&ti.info[0], Ptr(addr+Size_t(ti.info[0].size*i)), dest)
	}
	appendSize4(dest[:sizePos], len(dest)-sizePos-TSSD_SIZET_LENGTH)
	return dest, nil
}

func (ti *typeInfo) sliceDump(src []byte, dest Ptr) (remain []byte, err error) {
	if len(src) < TSSD_TYPE_LENGTH {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}

	switch int8(src[0]) {
	case ti.Type:
		_, arrayN, remain, err := checkDumpSize(src)
		if err != nil {
			return src, ErrorInSufficientData
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
			if remain, err = ti.info[0].dump(&ti.info[0], remain, Ptr(Size_t(addr)+Size_t(ti.info[0].size*i))); err != nil {
				return src, err
			}
		}
		return remain, nil

	case -ti.Type:
		//skip this field
		return src[1:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

// [Tarraym][Ttype][size][arrayN][data]
func (ti *typeInfo) mergeSliceSave(src Ptr, dest []byte) ([]byte, error) {
	arrayN := ti.size
	addr := Size_t(src)
	if ti.rtype.Kind() == reflect.Slice {
		p := *(*[]byte)(src)
		if arrayN = len(p); arrayN > 0 {
			addr = Size_t(Ptr(&p[0]))
		}
	}

	dest = append(dest, byte(ti.Type))         //T
	dest = append(dest, byte(ti.info[0].Type)) // we add a element data type after T

	totalSize := ti.info[0].size * arrayN
	dest = appendSize4(dest, 2+totalSize) //arrayN  + total Size
	dest = appendSize2(dest, arrayN)      //S

	//TODO, for big-endian, we need copy one by one
	return append(dest, Slice(Ptr(addr), Size_t(totalSize))...), nil
}

func (ti *typeInfo) mergeSliceDump(src []byte, dest Ptr) (remain []byte, err error) {
	if len(src) < 1 {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}
	var sizet, arrayN int
	switch int8(src[0]) {
	case ti.Type: //[0]: Tarraym, [1]: elementType
		sizet, arrayN, remain, err = checkDumpSize(src[1:])
		if err != nil {
			return src, ErrorInSufficientData
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
		copy(Slice(addr, Size_t(arrayN*ti.info[0].size)), remain)

	case -ti.Type:
		//skip this field
		return src[1:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
	return src[TSSD_TYPE_LENGTH+TSSD_TYPE_LENGTH+TSSD_SIZET_LENGTH+sizet:], nil
}

func (ti *typeInfo) dictSave(src Ptr, dest []byte) ([]byte, error) {

	value := reflect.NewAt(ti.rtype, src).Elem()
	keys := value.MapKeys()

	dest = append(dest, byte(ti.Type)) //T
	sizePos := len(dest)
	dest = appendSize4(dest, 0)         //reserved total Size (S)
	dest = appendSize2(dest, len(keys)) //S

	for _, k := range keys {
		v := value.MapIndex(k)
		dest = append(dest, byte(Tdictk))
		dest, _ = ti.info[0].mapSave(&ti.info[0], k, dest)
		dest = append(dest, byte(Tdictv))
		dest, _ = ti.info[1].mapSave(&ti.info[1], v, dest)
	}
	appendSize4(dest[:sizePos], len(dest)-sizePos-TSSD_SIZET_LENGTH)

	return dest, nil
}

func (ti *typeInfo) dictDump(src []byte, dest Ptr) (remain []byte, err error) {

	if len(src) < 1 {
		//TODO, add field name info
		return src, ErrorInSufficientData
	}

	switch int8(src[0]) {
	case ti.Type:
		_, mapLen, remain, err := checkDumpSize(src)
		if err != nil {
			return src, ErrorInSufficientData
		}

		mvalue := reflect.MakeMapWithSize(ti.rtype, mapLen)
		ktype := ti.rtype.Key()
		vtype := ti.rtype.Elem()

		var kk, vv reflect.Value
		for k := 0; k < mapLen; k++ {
			key := reflect.New(ktype).Elem()
			value := reflect.New(vtype).Elem()
			if remain[0] != byte(Tdictk) {
				return src, fmt.Errorf("%w [field type mismatch: expect %d but %d", ErrorInvalidTSSDData, Tdictk, remain[0])
			}

			kk, remain, err = ti.info[0].mapDump(&ti.info[0], remain[TSSD_TYPE_LENGTH:])
			if err != nil {
				return src, err
			}
			key.Set(kk.Convert(ktype))

			if remain[0] != byte(Tdictv) {
				return src, fmt.Errorf("%w [field type mismatch: expect %d but %d", ErrorInvalidTSSDData, Tdictv, remain[0])
			}

			vv, remain, err = ti.info[1].mapDump(&ti.info[1], remain[TSSD_TYPE_LENGTH:])
			if err != nil {
				return src, err
			}
			value.Set(vv.Convert(value.Type()))

			mvalue.SetMapIndex(key, value)
		}
		reflect.NewAt(ti.rtype, dest).Elem().Set(mvalue)
		return remain, nil
	case -ti.Type:
		//skip this field
		return src[1:], nil
	default:
		return src, fmt.Errorf("%w [field type mismatch %d %d]", ErrorInvalidTSSDData, src[0], ti.Type)
	}
}

func (ti *typeInfo) marshal(src any, dest []byte) ([]byte, error) {
	value := reflect.ValueOf(src)
	obj := value.Pointer()

	return ti.save(ti, Ptr(obj), dest)
}

func (ti *typeInfo) unmarshal(src []byte, dest any) ([]byte, error) {
	return ti.unmarshalTo(src, dest)
}

func (ti *typeInfo) unmarshalTo(src []byte, dest any) ([]byte, error) {
	value := reflect.ValueOf(dest)
	obj := value.Pointer()
	return ti.dump(ti, src, Ptr(obj))
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

func (ti *typeInfo) setType(typ int8) {
	ti.Type = typ
	ti.root.stype = append(ti.root.stype, byte(ti.Type))
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
		ti.setType(Tobject)
		ti.save = (*typeInfo).objSave
		ti.dump = (*typeInfo).objDump
		ti.mapSave, ti.mapDump = (*typeInfo).mapStructSave, (*typeInfo).mapStructDump

		fields := reflect.TypeOf(intf)
		num := fields.NumField()

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
		//we append struct's fields to validate
		ti.root.stype = appendSize2(ti.root.stype, len(ti.info))

	case reflect.Array, reflect.Slice: //for array, the memorry is continus:  &array==&array[0]
		ti.setType(Tarray)
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
			ti.setType(Tarraym)
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
