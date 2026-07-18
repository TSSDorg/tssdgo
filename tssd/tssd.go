package tssd

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"
)

const (
	MAGIC                = "TSSDV"
	TSSD_VERSION_MINOR   = 1
	TSSD_VERSION_MAJOR   = 0
	TSSD_FLAT_KIND       = "tssd.Flat"
	TSSD_TIME_KIND       = "time.Time"
	TSSD_TYPE_LENGTH     = 1
	TSSD_SIZET_LENGTH    = 4
	TSSD_SIZEA_LENGTH    = 2
	TSSD_BUFFER_MTU      = 3072
	TSSD_HASH_HALF_SIZE  = 6 //actually size means *2
	TSSD_CHECKSUM_LENGTH = TSSD_TYPE_LENGTH + TSSD_TYPE_LENGTH + TSSD_SIZET_LENGTH + TSSD_SIZEA_LENGTH + TSSD_HASH_HALF_SIZE + TSSD_HASH_HALF_SIZE
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
	Tdata           //TSSD data, it can be unmarshaled into Flatable object
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

var schemaTypeInfo *typeInfo

type Header struct {
	Magic   [5]byte
	Version [2]byte
}

// [Tobject][sizet/4bytes][sizea/2bytes][Tuint16][Fragments/2bytes][Tuint16][Current/2bytes][...]
type Schema struct {
	Fragment int16 //Fragment ID: [1,2,-3], < 0 means a ending fragment
	Hash     string
	TID      string
	Extent   string
}

type Patch struct {
	Fragment int16 //Fragment ID
	Off      int16 //position
	Value    int64
}

type Fragment struct {
	Header
	Schema
	Data []byte //TSSD content only
	//Patches  []Patch
	//checksum format: [Tarraym][Tuint8][sizet/4B][sizea/2B][checksum/12B]
	Checksum []byte //disgest of all the Fragment bytes
	Raw      []byte //raw data including fragment header, TSSD content, Checksum
}

func hash(types []byte) []byte {
	hasher := md5.New()
	hasher.Write(types)                         // Write the data to the hasher
	hashBytes := hasher.Sum(nil)                // Get the hash sum as a byte slice
	hashString := hex.EncodeToString(hashBytes) // Convert to a hex string
	l := len(hashString)
	return []byte(hashString[:TSSD_HASH_HALF_SIZE] + hashString[l-TSSD_HASH_HALF_SIZE:l])
}

// we need unmarshal fragment manualy
// @desc
// input: data input raw data, make sure it begin with magic "TSSD"
// return
//
//	[]byte: remain bytes after consume
//	error:  ErrorInSufficientData means need more data to unmarshal
//	        ErrorInvalidTSSDData is invalid data, you need drop all of them
func (frag *Fragment) Unmarshal(data []byte) ([]byte, error) {
	if len(data) < 8 {
		return data, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}

	if !isMagic(data) || data[7] != byte(Tschema) {
		return data, fmt.Errorf("%w [magic header not 'TSSD' or schema %d invalid]", ErrorInvalidTSSDData, data[7])
	}

	buf := &Buffer{
		Size: len(data),
		Data: [][]byte{
			data,
		},
	}

	buf.Read(frag.Header.Magic[:])
	buf.Read(frag.Header.Version[:])

	//skip Tschema
	if _, err := buf.ReadByte(); err != nil {
		return data, ErrorInSufficientData
	}

	err := (&frag.Schema).Unmarshal(buf)
	if err != nil {
		return data, err
	}

	posData := buf.pos + 8
	frag.Data, err = mergeByteSliceDump(data[buf.pos:])
	if err != nil {
		return data, err
	}
	//data before Checksum need hash to validate
	needCheck := data[0 : posData+len(frag.Data)]

	posChecksum := len(needCheck) + 8
	frag.Checksum, err = mergeByteSliceDump(data[len(needCheck):])
	if err != nil {
		return data, err
	}

	if err = frag.Validate(needCheck); err != nil {
		return data, err
	}
	frag.Raw = make([]byte, posChecksum+len(frag.Checksum))
	copy(frag.Raw, data)
	frag.Data = frag.Raw[posData : posData+len(frag.Data)]
	frag.Checksum = frag.Raw[posChecksum : posChecksum+len(frag.Checksum)]

	return data[posChecksum+len(frag.Checksum):], nil
}

func (frag *Fragment) Validate(input []byte) error {
	if string(hash(input)) != string(frag.Checksum) {
		return ErrorTSSDDataChecksumFailure
	}
	return nil
}

// [Tarraym][Tbyte][sizet][sizea][...]
func mergeByteSliceDump(input []byte) ([]byte, error) {
	if len(input) < 8 {
		return nil, ErrorInSufficientData
	}

	if string(input[:2]) != string([]byte{byte(Tarraym), byte(Tuint8)}) {
		return nil, ErrorInvalidTSSDData
	}
	var size4 int32
	copy(Slice(Ptr(&size4), unsafe.Sizeof(size4)), input[2:])
	var arrayN int16
	copy(Slice(Ptr(&arrayN), unsafe.Sizeof(arrayN)), input[6:])

	if size4 != int32(arrayN) {
		return nil, ErrorInvalidTSSDData
	}
	if len(input[8:]) < int(arrayN) {
		return nil, ErrorInSufficientData
	}

	return input[8 : 8+int(arrayN)], nil
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

func isMagic(buf []byte) bool {
	return string(buf[:len(MAGIC)]) == MAGIC
}
