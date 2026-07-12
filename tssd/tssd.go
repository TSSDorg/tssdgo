package tssd

import (
	"errors"
	"fmt"
	"unsafe"
)

const (
	MAGIC              = "TSSDV"
	TSSD_VERSION_MINOR = 1
	TSSD_VERSION_MAJOR = 0
	TSSD_FLAT_KIND     = "tssd.Flat"
	TSSD_TIME_KIND     = "time.Time"
	TSSD_TYPE_LENGTH   = 1
	TSSD_SIZET_LENGTH  = 4
	TSSD_SIZEA_LENGTH  = 2
	TSSD_BUFFER_CAP    = 3072
)

type Ttype int8

// data type define
const (
	Tbase int8 = 10 + iota
	Tbool      //fix-length-data
	Tint8
	Tuint8
	Tint16
	Tuint16
	Tint32
	Tuint32
	Tint64
	Tuint64
	Tfloat32
	Tfloat64
	Tstring //dynamic length data
	Ttime   //RFC3339Nano string
	Tenum
	Tarray
	Tarraym         //merged array, elements including 1 simple fixed length data only
	Tobject         //struct
	Tdict           //map, pairs of (key, value)
	Tdictk          //key of a map node
	Tdictv          //value of a map node
	Traw            //raw binary data
	Tschema  = 77   //'M' schema meta data string
	Theader  = 84   //'T' tssd header
	Tversion = 86   //'V' tssd format version
	Tuser    = 0xEF //user define data
)

var ErrorInvalidTSSDVersion = errors.New("TSSD version invalid or too new to process")
var ErrorInvalidTSSDData = errors.New("TSSD data invalid format error or damaged")
var ErrorInSufficientData = errors.New("Need more data to process")
var ErrorTSSDDataSchemaReject = errors.New("TSSD data schema not match")
var ErrorTSSDDataUnregister = errors.New("TSSD data schema not found or not register")

var schemaTypeInfo *typeInfo

type Header struct {
	Magic   [5]byte
	Version [2]byte
	Schema  Schema
}

type Schema struct {
	Hash    string
	Type    string
	Content string
}

func init() {
	schemaTypeInfo = parse(Schema{})
}

func (this *Schema) Marshal(buf *Buffer) error {
	return schemaTypeInfo.marshalTo(this, buf)
}

func (this *Schema) Unmarshal(from []byte) (remain []byte, err error) {
	return schemaTypeInfo.unmarshal(from, this)
}

func appendHeader(buf *Buffer, schema Schema) error {
	buf.Append([]byte(MAGIC))
	buf.Append([]byte{TSSD_VERSION_MINOR, TSSD_VERSION_MAJOR, Tschema})
	return (&schema).Marshal(buf)
}

func isMagic(buf []byte) bool {
	return string(buf[:len(MAGIC)]) == MAGIC
}

// [TSSD][Tversion][TSSD_VERSION_MINOR][TSSD_VERSION_MAJOR][Tschema][Tobject][sizet][sizea=3][xxxxxxx]
func dumpHeader(buf []byte) (header *Header, remain []byte, err error) {
	if len(buf) < 15 {
		return nil, buf, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}
	if !isMagic(buf) || buf[7] != byte(Tschema) {
		return nil, buf, fmt.Errorf("%w [magic header not 'TSSD' or version: %d schema %d invalid]", ErrorInvalidTSSDData, buf[4], buf[7])
	}

	header = &Header{
		Magic: [5]byte{'T', 'S', 'S', 'D', 'V'},
	}

	copy(header.Version[:], buf[5:])
	if remain, err = (&header.Schema).Unmarshal(buf[8:]); err != nil {
		return nil, buf, err
	}

	return
}

type Buffer struct {
	Cap   int
	Size  int //total size
	index int
	pos   int
	Data  [][]byte
}

func (buf *Buffer) writePos() (int, int) {
	idx := len(buf.Data) - 1
	return idx, len(buf.Data[idx])
}

func (buf *Buffer) Append(bs []byte) *Buffer {
	if len(buf.Data) == 0 {
		if buf.Cap == 0 {
			buf.Cap = TSSD_BUFFER_CAP
		}
		buf.Data = append(buf.Data, make([]byte, 0, buf.Cap))
	}
	for len(bs) > 0 {
		w := len(buf.Data) - 1
		if len(buf.Data[w])+len(bs) <= cap(buf.Data[w]) {
			buf.Data[w] = append(buf.Data[w], bs...)
			buf.Size += len(bs)
			return buf
		}

		fill := cap(buf.Data[w]) - len(buf.Data[w])
		buf.Data[w] = append(buf.Data[w], bs[:fill]...)
		buf.Size += fill
		//TODO, how can we let user supply Data ?
		buf.Data = append(buf.Data, make([]byte, 0, buf.Cap))
		bs = bs[fill:]
	}
	//return self let us call in chain
	return buf
}

func (buf *Buffer) AppendByte(b byte) *Buffer {
	return buf.Append([]byte{b})
}

func (buf *Buffer) Read(dest []byte) (result []byte, err error) {
	if len(dest) == 0 {
		return nil, nil
	}
	if buf.Size < len(dest) {
		return nil, ErrorInSufficientData
	}

	result = dest
	wanted := len(dest)
	n := copy(dest[:wanted], buf.Data[buf.index][buf.pos:])
	buf.Size -= n
	buf.pos += n
	if buf.pos >= buf.Cap {
		buf.pos = 0
		buf.index++
	}
	if n >= wanted {
		return result, nil
	}

	dest = dest[n:]
	for {
		n = copy(dest, buf.Data[buf.index])
		buf.Size -= n
		dest = dest[n:]
		if len(dest) == 0 {
			break
		}
		buf.index++
	}
	buf.pos += n
	if buf.pos >= buf.Cap {
		buf.pos = 0
		buf.index++
	}
	return result, nil
}

func (buf *Buffer) appendSize2(le int) *Buffer {
	l := uint16(le)
	return buf.Append(Slice(Ptr(&l), unsafe.Sizeof(l)))
}

func (buf *Buffer) appendSize4(le int) *Buffer {
	l := uint32(le)
	return buf.Append(Slice(Ptr(&l), unsafe.Sizeof(l)))
}

func (buf *Buffer) appendString(s string) *Buffer {
	return buf.appendSize4(len(s)).Append([]byte(s))
}

func (buf *Buffer) updateSize(index, pos, value int) {
	l := uint32(value)
	s := Slice(Ptr(&l), unsafe.Sizeof(l))
	for i := 0; i < len(s); i++ {
		buf.Data[index][pos] = s[i]
		pos++
		if pos >= buf.Cap {
			pos = 0
			index++
		}
	}
}
