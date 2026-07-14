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
var ErrorTSSDDataSchemaUnmatch = errors.New("TSSD data schema not match or unregistered")
var ErrorTSSDHeadOverSizeFragment = errors.New("TSSD Head large than fragment size limitation")

var schemaTypeInfo *typeInfo

type Header struct {
	Magic   [5]byte
	Version [2]byte
	Schema  Schema
}

// [Tobject][sizet/4bytes][sizea/2bytes][Tuint16][Fragments/2bytes][Tuint16][Current/2bytes][...]
type Schema struct {
	Hash     string
	TID      string
	Fragment int16 //Fragment ID: [1,2,-3], < 0 means a ending fragment
	Extent   string
}

func init() {
	schemaTypeInfo = parse(Schema{})
}

func (this *Schema) Marshal(buf *Buffer) error {
	return schemaTypeInfo.marshalTo(this, buf)
}

func (this *Schema) Unmarshal(buf *Buffer) error {
	return schemaTypeInfo.unmarshal(buf, this)
}

func appendHeader(buf *Buffer, schema Schema) error {
	buf.schema = &schema
	buf.Append([]byte(MAGIC))
	buf.Append([]byte{TSSD_VERSION_MINOR, TSSD_VERSION_MAJOR, Tschema})
	err := buf.schema.Marshal(buf)
	if err != nil {
		return err
	}
	buf.AppendByte(byte(Traw))
	buf.appendSize4(0) //reserve sizet for raw data
	//we will keep a copy of schema in buf.heads
	if buf.Size >= cap(buf.Data[0]) { //TSSD Heads too large than the cap(fragment limitation)
		return ErrorTSSDHeadOverSizeFragment
	}
	buf.heads = buf.Data[0][:buf.Size]
	return nil
}

func isMagic(buf []byte) bool {
	return string(buf[:len(MAGIC)]) == MAGIC
}

// [TSSD][Tversion][TSSD_VERSION_MINOR][TSSD_VERSION_MAJOR][Tschema][Tobject][sizet][sizea=3][xxxxxxx]
func dumpHeader(buf *Buffer) (*Header, error) {
	bs, err := buf.Read(make([]byte, 8))
	if err != nil {
		return nil, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}
	if !isMagic(bs) || bs[7] != byte(Tschema) {
		return nil, fmt.Errorf("%w [magic header not 'TSSD' or schema %d invalid]", ErrorInvalidTSSDData, bs[7])
	}

	header := &Header{
		Magic: [5]byte{'T', 'S', 'S', 'D', 'V'},
	}

	copy(header.Version[:], bs[5:])
	if err = (&header.Schema).Unmarshal(buf); err != nil {
		return nil, err
	}

	if b, err := buf.ReadByte(); err != nil || int8(b) != Traw {
		return nil, ErrorInvalidTSSDData
	}

	if _, err := dumpSize4(buf); err != nil {
		return nil, err
	}

	return header, err
}

type Buffer struct {
	schema *Schema
	heads  []byte
	Cap    int
	Size   int //total size
	index  int //read index
	pos    int //read position
	windex int //write index
	Data   [][]byte
}

// reset buf's read info, let user read from begin
func (buf *Buffer) Rewind() *Buffer {
	buf.index, buf.pos, buf.Size = 0, 0, 0
	for i := 0; i <= buf.windex; i++ {
		buf.Size += len(buf.Data[i])
	}
	return buf
}

func (buf *Buffer) writePos() (int, int) {
	idx := len(buf.Data) - 1
	pos := len(buf.Data[idx])
	if pos >= cap(buf.Data[idx]) {
		pos = 0
		idx++
	}
	return idx, pos
}

func (buf *Buffer) Append(bs []byte) *Buffer {
	for len(bs) > 0 {
		if buf.windex == len(buf.Data) {
			if buf.Cap == 0 {
				buf.Cap = TSSD_BUFFER_CAP
			}
			buf.Data = append(buf.Data, make([]byte, 0, buf.Cap))
		}

		if len(buf.Data[buf.windex])+len(bs) <= cap(buf.Data[buf.windex]) {
			buf.Data[buf.windex] = append(buf.Data[buf.windex], bs...)
			buf.Size += len(bs)
			return buf
		}

		fill := cap(buf.Data[buf.windex]) - len(buf.Data[buf.windex])
		buf.Data[buf.windex] = append(buf.Data[buf.windex], bs[:fill]...)
		buf.Size += fill
		bs = bs[fill:]
		buf.windex++
	}
	//return self let us call in chain
	return buf
}

func (buf *Buffer) AppendByte(b byte) *Buffer {
	return buf.Append([]byte{b})
}

func (buf *Buffer) ReadByte() (b byte, err error) {
	_, err = buf.Read(Slice(Ptr(&b), TSSD_TYPE_LENGTH))
	return b, err
}

func (buf *Buffer) Read(dest []byte) (result []byte, err error) {
	if len(dest) == 0 {
		return nil, nil
	}
	if buf.Size < len(dest) {
		//should we return partial content ?
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
