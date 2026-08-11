package tssd

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"unsafe"
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
	TSSD_BUFFER_MIN_MTU           = 256
	TSSD_BUFFER_MTU               = 2048
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

var schemaTypeInfo *typeInfo

type Header struct {
	Magic   [5]byte
	Version [2]byte
}

// [Tobject][sizet/4bytes][sizea/2bytes][Tuint16][Fragments/2bytes][Tuint16][Current/2bytes][...]
type Schema struct {
	FID      int16  // Fragment ID: [1,2,...,(N-1), -N], < 0 means an ending fragment
	TID      string // object ID
	Types    string // Types
	Group    string // Group
	Info     string // reserve for other user info
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

// unmarshal fragment manualy from bytes
// @desc  public api, copy Data from user's space until validate success
// input: data should contains magic "TSSDV", length should > TSSD_FRAGMENT_MIN_HEADER_SIZE
//
// return
//
//	 more:   need more data if we meet ErrorInSufficientData
//		[]byte: remain bytes after consume when unmarshal success
//		error:  ErrorInSufficientData means need more data to unmarshal
//		        ErrorInvalidTSSDData is invalid data
func (frag *Fragment) Unmarshal(input []byte) (more int, remain []byte, err error) {
	more, remain, err = frag.unmarshal(input)
	if err != nil {
		return more, remain, err
	}
	frag.Data = append(make([]byte, 0, len(frag.Data)), frag.Data...)
	headLen := len(frag.heads)
	frag.heads = frag.Data[:headLen]
	frag.payload = frag.Data[headLen : headLen+len(frag.payload)]
	frag.checksum = frag.Data[headLen+len(frag.payload):]
	return more, remain, err
}

// we need unmarshal fragment manualy
// @desc  internal api, we don't copy data from user space
// input: data should contains magic "TSSDV", length should > TSSD_FRAGMENT_MIN_HEADER_SIZE
//
// return
//
//	 more:   need more data if we meet ErrorInSufficientData
//		[]byte: remain bytes after consume when unmarshal success
//		error:  ErrorInSufficientData means need more data to unmarshal
//		        ErrorInvalidTSSDData is invalid data
func (frag *Fragment) unmarshal(input []byte) (more int, remain []byte, err error) {
	if len(input) < TSSD_FRAGMENT_MIN_HEADER_SIZE {
		return TSSD_FRAGMENT_MIN_HEADER_SIZE, nil, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}

	i := bytes.Index(input, []byte(MAGIC))
	if i < 0 {
		return 0, nil, fmt.Errorf("%w [TSSD MAGIC head invalid]", ErrorInvalidTSSDData)
	}
	data := input[i:]
	buf := &Buffer{
		Size: len(data),
		fragments: map[int]*Fragment{
			0: &Fragment{
				payload: data,
				Data:    data,
			},
		},
	}

	buf.Read(frag.Header.Magic[:])
	buf.Read(frag.Header.Version[:])

	// Tschema
	if b, _ := buf.ReadByte(); b != byte(Tschema) {
		return 0, nil, fmt.Errorf("%w [schema type %d invalid]", ErrorInvalidTSSDData, b)
	}

	err = (&frag.Schema).unmarshal(buf)
	if err != nil {
		if errors.Is(err, ErrorInSufficientData) {
			return TSSD_FRAGMENT_MIN_HEADER_SIZE, nil, err
		}
		return 0, nil, err
	}

	posData := buf.pos + TSSD_TARRAYM_HEAD_LENGTH
	more, frag.payload, err = mergeByteSliceDump(data[buf.pos:])
	if err != nil {
		return more, nil, err
	}
	frag.heads = data[:posData]
	//data before Checksum need hash to validate
	needCheck := data[0 : posData+len(frag.payload)]
	posChecksum := len(needCheck)
	more, checksum, err := mergeByteSliceDump(data[len(needCheck):])
	if err != nil {
		return more, nil, err
	}
	// keep checksum including the Tarraym header
	frag.checksum = data[len(needCheck) : len(needCheck)+TSSD_TARRAYM_HEAD_LENGTH+len(checksum)]

	if err = frag.Validate(needCheck); err != nil {
		return 0, nil, err
	}
	frag.Data = data[:posChecksum+len(frag.checksum)]
	return 0, data[posChecksum+len(frag.checksum):], nil
}

// read a fragment
func (frag *Fragment) Read(rd io.Reader) (err error) {
	more := TSSD_FRAGMENT_MIN_HEADER_SIZE
	bs := make([]byte, TSSD_BUFFER_MTU, TSSD_BUFFER_MTU)
	size := 0
	var remain []byte
	for {
		if size+more > len(bs) {
			bs = append(make([]byte, 0, more+TSSD_BUFFER_MTU), bs...)
		}
		n, err := rd.Read(bs[size : size+more])
		if n == 0 && err != nil {
			return err
		}
		size += n
		more, remain, err = frag.unmarshal(bs[:size]) // call internal api, no need copy
		if err == nil {
			fmt.Println("Received fragment:", frag.FID, " with length:", len(frag.Data), " remain:", len(remain))
			// need drop the data from bufio to prepare the next fragment
			return nil
		}
		if !errors.Is(err, ErrorInSufficientData) {
			fmt.Println("Error occurred while unmarshalling:", err)
			return err
		}
	}
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

// [Tarraym][Tbyte][sizet][sizea][...]
// return
//
//	 more:   need more data if we meet ErrorInSufficientData
//		[]byte: remain bytes after consume, nil if meet err
//		error:  ErrorInSufficientData means need more data to unmarshal
//		        ErrorInvalidTSSDData is invalid data, you need drop all of them
func mergeByteSliceDump(input []byte) (more int, remain []byte, err error) {
	if len(input) < TSSD_TARRAYM_HEAD_LENGTH {
		return TSSD_TARRAYM_HEAD_LENGTH - len(input), nil, ErrorInSufficientData
	}

	if string(input[:2]) != string([]byte{byte(Tarraym), byte(Tuint8)}) {
		return 0, nil, fmt.Errorf("%w [Fragment Tarraym type %d %d invalid]", ErrorInvalidTSSDData, input[0], input[1])
	}
	var size4 int32
	copy(Slice(Ptr(&size4), unsafe.Sizeof(size4)), input[2:])
	var arrayN int16
	copy(Slice(Ptr(&arrayN), unsafe.Sizeof(arrayN)), input[6:])

	if size4 != int32(arrayN)+TSSD_SIZEA_LENGTH {
		return 0, nil, fmt.Errorf("%w [Fragment Tarraym size %d %d invalid]", ErrorInvalidTSSDData, size4, arrayN)
	}
	if len(input[TSSD_TARRAYM_HEAD_LENGTH:]) < int(arrayN) {
		return int(arrayN) - len(input[TSSD_TARRAYM_HEAD_LENGTH:]), nil, ErrorInSufficientData
	}

	return 0, input[TSSD_TARRAYM_HEAD_LENGTH : TSSD_TARRAYM_HEAD_LENGTH+int(arrayN)], nil
}

func init() {
	schemaTypeInfo = parse(Schema{})
}

func (this *Schema) marshal(buf *Buffer) error {
	//buf.Clear()
	err := schemaTypeInfo.marshalTo(this, buf)
	if err == nil && buf.Size > 0 {
		buf.fragments[0].Data = buf.fragments[0].Data[:buf.Size]
	}
	return err
}

func (this *Schema) unmarshal(buf *Buffer) error {
	return schemaTypeInfo.unmarshal(buf, this)
}

func isMagic(buf []byte) bool {
	return string(buf[:len(MAGIC)]) == MAGIC
}
