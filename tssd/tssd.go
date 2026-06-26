package tssd

import (
	"errors"
	"fmt"
)

const (
	MAGIC          = "SSD"
	TSSD_VERSION   = 1
	TSSD_FLAT_KIND = "tssd.Flat"
	TSSD_TIME_KIND = "time.Time"
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
	Tschema  = 83   //'S' schema meta data string
	Theader  = 84   //'T' tssd header
	Tversion = 86   //'V' tssd format version
	Tuser    = 0xEF //user define data
)

var ErrorInvalidTSSDVersion = errors.New("TSSD version invalid or too new to process")
var ErrorInvalidTSSDData = errors.New("TSSD data invalid format error or damaged")
var ErrorInSufficientData = errors.New("Need more data to process")
var ErrorTSSDDataSchemaReject = errors.New("TSSD data schema not match")

var schemaTypeInfo *typeInfo

type Header struct {
	Magic   [4]byte
	Version int16
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

func (this *Schema) Marshal(to []byte) (ret []byte) {
	ret, _ = schemaTypeInfo.marshal(this, to)
	return ret
}

func (this *Schema) Unmarshal(from []byte) (remain []byte, err error) {
	return schemaTypeInfo.unmarshal(from, this)
}

func appendHeader(buf []byte, schema Schema) []byte {

	buf = append(buf, Theader)
	buf = append(buf, MAGIC...)
	buf = append(buf, Tversion)
	buf = appendSize(buf, TSSD_VERSION)

	buf = append(buf, byte(Tschema))
	return schema.Marshal(buf)
}

func isMagic(buf []byte) bool {
	return buf[0] == Theader && buf[1] == MAGIC[0] && buf[2] == MAGIC[1] && buf[3] == MAGIC[2]
}

// [TSSD][Tversion][TSSD_VERSION][Tschema][string-size][xxxxxxx]
func dumpHeader(buf []byte) (header *Header, remain []byte, err error) {
	if len(buf) < 10 {
		return nil, buf, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}
	if !isMagic(buf) || buf[4] != byte(Tversion) || buf[7] != byte(Tschema) {
		return nil, buf, fmt.Errorf("%w [magic header not 'TSSD' or version: %d schema %d invalid]", ErrorInvalidTSSDData, buf[4], buf[7])
	}

	header = &Header{
		Magic: [4]byte{'T', 'S', 'S', 'D'},
		//Schema: &schema,
	}

	header.Version = dumpSize(buf[5:])

	if remain, err = (&header.Schema).Unmarshal(buf[8:]); err != nil {
		return nil, buf, err
	}

	return
}
