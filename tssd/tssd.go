package tssd

import (
	//"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
)

const (
	MAGIC                         = "TSSDV"
	TSSD_VERSION_MINOR            = 1
	TSSD_VERSION_MAJOR            = 0
	TSSD_FLAT_KIND                = "tssd.Flat"
	TSSD_TIME_KIND                = "time.Time"
	TSSD_FIELD_TAG_KEY            = "tssd"
	TSSD_FIELD_TAG_IGNORE         = "-"
	TSSD_FIELD_TAG_SPLITER        = ","
	TSSD_TYPE_LENGTH              = 1
	TSSD_SIZET_LENGTH             = 4
	TSSD_SIZEA_LENGTH             = 2
	TSSD_TARRAYM_HEAD_LENGTH      = 8 //8 bytes for [Tarraym][Tuint8][sizet/4B][sizea/2B]
	TSSD_BUFFER_MIN_MTU           = 128
	TSSD_BUFFER_MTU               = 1440
	TSSD_FRAGMENT_MIN_HEADER_SIZE = 64
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
var ErrorTSSDDataChecksumFailure = errors.New("TSSD fragment data checksum failure")
var ErrorTSSDDataFragmentIDUnmatch = errors.New("TSSD fragments TID unmatch")
var ErrorRegisterFlatFailure = errors.New("register Flatable falure: Family and Version should not empty")

var schemaTypeInfo *typeInfo

type Header struct {
	Magic   [5]byte
	Version [2]byte
}

// [Tobject][sizet/4bytes][sizea/2bytes][Tuint16][Fragments/2bytes][Tuint16][Current/2bytes][...]
type Schema struct {
	FID    int16  // Fragment ID: [1,2,...,(N-1), -N], < 0 means an ending fragment
	Types  string // Types
	TID    string // object ID
	Info   string // reserve for other user info
}

type Patch struct {
	Fragment int16 //Fragment ID
	Off      int16 //position
	Value    int64
}

type Fragment struct {
	Header
	Schema
	heads    []byte // bytes before payload(including payload's Tarraym head)
	payload  []byte // TSSD content only
	checksum []byte // disgest of all the Fragment bytes(including checksum's Tarram head)
	Data     []byte // raw data including fragment header, TSSD content, Checksum
}

var HashFunc func([]byte) []byte = hash
var ChecksumFunc func([]byte) []byte = hash

func hash(types []byte) []byte {
	const TSSD_HASH_HALF_SIZE = 6
	hasher := md5.New()
	hasher.Write(types)                         // Write the data to the hasher
	hashBytes := hasher.Sum(nil)                // Get the hash sum as a byte slice
	hashString := hex.EncodeToString(hashBytes) // Convert to a hex string
	l := len(hashString)
	return []byte(hashString[:TSSD_HASH_HALF_SIZE] + hashString[l-TSSD_HASH_HALF_SIZE:l])
}

func (frag *Fragment) Write(wr io.Writer) (nn int, err error) {
	return wr.Write(frag.Data)
}

func (frag *Fragment) Validate(input []byte) error {
	// if frag.Checksum() is empty, we skip checksum validation
	if len(frag.Checksum()) > 0 && string(ChecksumFunc(input)) != string(frag.Checksum()) {
		return ErrorTSSDDataChecksumFailure
	}
	return nil
}

func (frag *Fragment) Heads() []byte {
	return frag.heads[:len(frag.heads)-TSSD_TARRAYM_HEAD_LENGTH]
}

func (frag *Fragment) Payload() []byte {
	return frag.payload
}

func (frag *Fragment) Checksum() []byte {
	return frag.checksum[TSSD_TARRAYM_HEAD_LENGTH:]
}

func init() {
	schemaTypeInfo = parse(Schema{})
}

func (this *Schema) marshal(buf *Buffer) error {
	if err := schemaTypeInfo.marshalTo(this, buf); err != nil {
		return nil
	}
	if len(buf.fragments) > 1 {
		return ErrorTSSDHeadOverSizeFragment
	}
	buf.fragments[0].Data = buf.fragments[0].Data[:buf.Size]
	return nil
}

func (this *Schema) unmarshal(buf *Buffer) error {
	return schemaTypeInfo.unmarshal(buf, this)
}

func isMagic(buf []byte) bool {
	return string(buf[:len(MAGIC)]) == MAGIC
}
