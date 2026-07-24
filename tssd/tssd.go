package tssd

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"
)

const (
	MAGIC               = "TSSDV"
	TSSD_VERSION_MINOR  = 1
	TSSD_VERSION_MAJOR  = 0
	TSSD_FLAT_KIND      = "tssd.Flat"
	TSSD_TIME_KIND      = "time.Time"
	TSSD_TYPE_LENGTH    = 1
	TSSD_SIZET_LENGTH   = 4
	TSSD_SIZEA_LENGTH   = 2
	TSSD_BUFFER_MIN_MTU = 256
	TSSD_BUFFER_MTU     = 3072
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
	tdata    []byte //TSSD content only
	Checksum []byte //disgest of all the Fragment bytes
	Data     []byte //raw data including fragment header, TSSD content, Checksum
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

// we need unmarshal fragment manualy
// @desc
// input: data input raw data, make sure it contains magic "TSSDV"
// return
//
//	[]byte: remain bytes after consume
//	error:  ErrorInSufficientData means need more data to unmarshal
//	        ErrorInvalidTSSDData is invalid data, you need drop all of them
func (frag *Fragment) Unmarshal(input []byte) ([]byte, error) {
	i := bytes.Index(input, []byte(MAGIC))
	if i < 0 {
		return nil, ErrorInvalidTSSDData
	}
	data := input[i:]
	if len(data) < 8 {
		return data, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}

	if !isMagic(data) || data[7] != byte(Tschema) {
		return data, fmt.Errorf("%w [magic header not 'TSSD' or schema %d invalid]", ErrorInvalidTSSDData, data[7])
	}

	buf := &Buffer{
		Size: len(data),
		fragments: map[int]*Fragment{
			0: &Fragment{
				tdata: data,
				Data:  data,
			},
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
	frag.tdata, err = mergeByteSliceDump(data[buf.pos:])
	if err != nil {
		return data, err
	}
	//data before Checksum need hash to validate
	needCheck := data[0 : posData+len(frag.tdata)]

	posChecksum := len(needCheck) + 8
	frag.Checksum, err = mergeByteSliceDump(data[len(needCheck):])
	if err != nil {
		return data, err
	}

	if err = frag.Validate(needCheck); err != nil {
		return data, err
	}
	frag.Data = make([]byte, posChecksum+len(frag.Checksum))
	copy(frag.Data, data)
	frag.tdata = frag.Data[posData : posData+len(frag.tdata)]
	frag.Checksum = frag.Data[posChecksum : posChecksum+len(frag.Checksum)]

	return data[posChecksum+len(frag.Checksum):], nil
}

func (frag *Fragment) Validate(input []byte) error {
	// if frag.Checksum is empty, we skip checksum validation
	if len(frag.Checksum) > 0 && string(ChecksumFunc(input)) != string(frag.Checksum) {
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

	if size4 != int32(arrayN)+TSSD_SIZEA_LENGTH {
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
	//buf.Clear()
	err := schemaTypeInfo.marshalTo(this, buf)
	if err == nil && buf.Size > 0 {
		buf.fragments[0].Data = buf.fragments[0].Data[:buf.Size]
	}
	return err
}

func (this *Schema) Unmarshal(buf *Buffer) error {
	return schemaTypeInfo.unmarshal(buf, this)
}

func isMagic(buf []byte) bool {
	return string(buf[:len(MAGIC)]) == MAGIC
}
