package tssd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// RBuffer incrementally receives TSSD fragments and groups them by schema.
type RBuffer struct {
	data         []byte
	magic        int
	header       Header
	schema       Schema
	headsLen     int
	payload      int
	payloadLen   int
	checksum     int
	checksumLen  int
	results      map[string]map[string]map[string]*Buffer  // family -> version -> tid -> Buffer
	unregistered map[string]map[string]*Buffer  // types -> tid -> Buffer
}

func NewRBuffer() *RBuffer {
	buf := &RBuffer{}
	buf.reset()
	return buf
}


func (buf *RBuffer) append(data []byte) {
	buf.data = append(buf.data, data...)
}

func (buf *RBuffer) reset() {
	buf.magic = -1
	buf.headsLen = -1
	buf.payload = -1
	buf.payloadLen = -1
	buf.checksum = -1
	buf.checksumLen = -1
}

func (buf *RBuffer) dumpMergeArrayHeader(pos int) (length, more int, err error) {
	if pos < 0 || pos+TSSD_TARRAYM_HEAD_LENGTH > len(buf.data) {
		return 0, pos + TSSD_TARRAYM_HEAD_LENGTH - len(buf.data), ErrorInSufficientData
	}
	if buf.data[pos] != byte(Tarraym) || buf.data[pos+1] != byte(Tuint8) {
		return 0, 0, fmt.Errorf("%w [Fragment Tarraym type invalid]", ErrorInvalidTSSDData)
	}
	size := int32(binary.LittleEndian.Uint32(buf.data[pos+2 : pos+6]))
	arrayLength := int16(binary.LittleEndian.Uint16(buf.data[pos+6 : pos+8]))
	if size != int32(arrayLength)+TSSD_SIZEA_LENGTH || arrayLength < 0 {
		return 0, 0, fmt.Errorf("%w [Fragment Tarraym size invalid]", ErrorInvalidTSSDData)
	}
	return int(arrayLength), 0, nil
}

func (buf *RBuffer) parseHeads() (int, error) {
	if buf.headsLen >= 0 {
		return 0, nil
	}
	if len(buf.data) < TSSD_FRAGMENT_MIN_HEADER_SIZE {
		return TSSD_FRAGMENT_MIN_HEADER_SIZE - len(buf.data), ErrorInSufficientData
	}
	if buf.data[buf.magic+7] != byte(Tschema) {
		return 0, ErrorInvalidTSSDData
	}
	copy(buf.header.Magic[:], buf.data[buf.magic:buf.magic+5])
	copy(buf.header.Version[:], buf.data[buf.magic+5:buf.magic+7])

	cursor := buf.magic + 8
	input := buf.data[cursor:]
	schemaBuffer := &Buffer{
		MTU:  len(input),
		Size: len(input),
		fragments: map[int]*Fragment{
			0: {payload: input, Data: input},
		},
	}
	if err := buf.schema.unmarshal(schemaBuffer); err != nil {
		if errors.Is(err, ErrorInSufficientData) {
			return TSSD_FRAGMENT_MIN_HEADER_SIZE, err
		}
		return 0, err
	}
	cursor += len(input) - schemaBuffer.Size
	payloadLength, more, err := buf.dumpMergeArrayHeader(cursor)
	if err != nil {
		return more, err
	}
	buf.payloadLen = payloadLength
	buf.headsLen = cursor + TSSD_TARRAYM_HEAD_LENGTH - buf.magic
	return 0, nil
}

func (buf *RBuffer) parsePayload() (int, error) {
	if buf.payload >= 0 {
		return 0, nil
	}
	needed := buf.headsLen + buf.payloadLen
	if needed > len(buf.data)-buf.magic {
		return needed - (len(buf.data) - buf.magic), ErrorInSufficientData
	}
	buf.payload = buf.magic + buf.headsLen
	return 0, nil
}

func (buf *RBuffer) parseChecksum() (int, error) {
	if buf.checksum >= 0 {
		return 0, nil
	}
	checksumLength, more, err := buf.dumpMergeArrayHeader(buf.payload + buf.payloadLen)
	if err != nil {
		return more, err
	}
	needed := buf.headsLen + buf.payloadLen + TSSD_TARRAYM_HEAD_LENGTH + checksumLength
	if needed > len(buf.data)-buf.magic {
		return needed - (len(buf.data) - buf.magic), ErrorInSufficientData
	}
	buf.checksumLen = checksumLength + TSSD_TARRAYM_HEAD_LENGTH
	buf.checksum = buf.payload + buf.payloadLen
	return 0, nil
}

func (buf *RBuffer) unmarshalFragment() (int, error) {
	if more, err := buf.parseHeads(); err != nil {
		return more, err
	}
	if more, err := buf.parsePayload(); err != nil {
		return more, err
	}
	if more, err := buf.parseChecksum(); err != nil {
		return more, err
	}

	frameLength := buf.headsLen + buf.payloadLen + buf.checksumLen
	frame :=  buf.data[buf.magic:buf.magic+frameLength]
	fragment := &Fragment{
		Header:   buf.header,
		Schema:   buf.schema,
		Data:     frame,
		heads:    frame[:buf.headsLen],
		payload:  frame[buf.headsLen : buf.headsLen+buf.payloadLen],
		checksum: frame[buf.headsLen+buf.payloadLen:],
	}
	if err := fragment.Validate(frame[:buf.headsLen+buf.payloadLen]); err != nil {
		return 0, err
	}
	return 0, nil
}

func (buf *RBuffer) detectMagic(data []byte, skip int) (more int, err error) {
	if buf.magic >= 0 {
		buf.append(data)
		if (len(buf.data) < TSSD_FRAGMENT_MIN_HEADER_SIZE) {
        	more = TSSD_FRAGMENT_MIN_HEADER_SIZE - len(buf.data);
		}
		return more, nil
	}
	preSize := len(buf.data)
	cpsize := min(len(data), 4)
	if preSize >= len(MAGIC) {
		buf.magic = bytes.Index(buf.data[skip:], []byte(MAGIC))
		if buf.magic >= 0 {
			buf.data = buf.data[skip+buf.magic:]
			buf.append(data)
			goto found
		}
		buf.data = buf.data[preSize-4:]
	}
	preSize = len(buf.data)
	if preSize+len(data) < len(MAGIC) {
		buf.append(data)
		more = TSSD_FRAGMENT_MIN_HEADER_SIZE - len(buf.data)
		return more, ErrorInSufficientData
	}

	buf.append(data[:cpsize])
	buf.magic = bytes.Index(buf.data, []byte(MAGIC))
	if buf.magic < 0 {
		pos := bytes.Index(data, []byte(MAGIC))
		if pos < 0 {
			buf.data = buf.data[preSize-(4-cpsize):preSize]
			buf.append(data[len(data)-cpsize:])
			more = TSSD_FRAGMENT_MIN_HEADER_SIZE - len(buf.data)
			return more, ErrorInSufficientData
		}
		buf.data = append(make([]byte, 0, len(data[pos:])), data[pos:]...)
		goto found
	}

	buf.data = buf.data[buf.magic:preSize]
	buf.append(data)
found:
	if len(buf.data) < TSSD_FRAGMENT_MIN_HEADER_SIZE {
		more = TSSD_FRAGMENT_MIN_HEADER_SIZE - len(buf.data)
	}
	buf.magic = bytes.Index(buf.data, []byte(MAGIC))
	return more, nil
}

// Extract adds encoded data and returns the number of bytes still needed for a
// complete fragment. The returned error is nil when a fragment is available.
func (buf *RBuffer) Extract(data []byte) (more int, err error) {
	skip := 0
	for {
		if more, err = buf.detectMagic(data, skip); err != nil {
			return more, err
		}
		more, err = buf.unmarshalFragment()
		if err == nil {
			return 0, nil
		}
		if errors.Is(err, ErrorInSufficientData) {
			return more, err
		}

		buf.reset()
		skip = 5
		data = nil
	}
}

// ExtractReader reads and extracts fragments until one complete Buffer is ready.
func (buf *RBuffer) ExtractReader(reader io.Reader) (err error) {
	chunk := make([]byte, TSSD_BUFFER_MTU)
	more := TSSD_FRAGMENT_MIN_HEADER_SIZE
	got := false
	exit := false
	for !exit {
		if more > 0 {
			readSize := min(more, TSSD_BUFFER_MTU)
			if cap(chunk) < readSize {
				chunk = make([]byte, readSize)
			}
			chunk = chunk[:readSize]
			var n int
			n, err = reader.Read(chunk)
			if n <= 0 {
				exit = true
				//err = io.UnexpectedEOF
				n = 0
			}
			chunk = chunk[:n]
		}

		for {
			var extractErr error
			more, extractErr = buf.Extract(chunk)
			chunk = chunk[:0]
			if extractErr != nil {
				if errors.Is(extractErr, ErrorInSufficientData) {
					break
				}
				return extractErr
			}

			fragment := buf.Fragment()
			if buf.Push(fragment) == nil && !got {
				got = true
			}
		}
	}
	if !got {
		return err
	}
	return nil
}

// Fragment returns compost fragment and clean cache.
func (buf *RBuffer) Fragment() *Fragment {
	frameLength := buf.headsLen + buf.payloadLen + buf.checksumLen
	frame :=  append(make([]byte, 0, frameLength), buf.data[buf.magic:buf.magic+frameLength]...)
	fragment := &Fragment{
		Header:   buf.header,
		Schema:   buf.schema,
		Data:     frame,
		heads:    frame[:buf.headsLen],
		payload:  frame[buf.headsLen : buf.headsLen+buf.payloadLen],
		checksum: frame[buf.headsLen+buf.payloadLen:],
	}
	buf.data = buf.data[buf.magic+frameLength:]
	buf.reset()
	return fragment
}

// Buffer returns the assembled buffer for a family and version when complete.
func (buf *RBuffer) Buffer(family, version string) *Buffer {
	if buf.results == nil {
		return nil
	}
	versions := buf.results[family]
	if versions == nil {
		return nil
	}
	objects := versions[version]
	for tid, result := range objects {
		if result.Wanted() == 0 {
			delete(objects, tid)
			return result
		}
	}
	return nil
}

// Push routes the most recently decoded fragment into its assembled buffer.
// It returns true when all fragments for that object have arrived.
func (buf *RBuffer) Push(fragment *Fragment) (error) {
	location, ok := registers.types[fragment.Types]
	if !ok {
		if buf.unregistered == nil {
			buf.unregistered = make(map[string]map[string]*Buffer)
		}
		result := buf.unregistered[fragment.Types]
		if result == nil {
			result = make(map[string]*Buffer)
			buf.unregistered[fragment.Types] = result
		}
		tid := result[fragment.TID]
		if tid == nil {
			tid = &Buffer{}
			result[fragment.TID] = tid
		}
		tid.Push(fragment)
		return ErrorTSSDDataSchemaUnmatch
	}
	if buf.results == nil {
		buf.results = make(map[string]map[string]map[string]*Buffer)
	}
	if buf.results[location.family] == nil {
		buf.results[location.family] = make(map[string]map[string]*Buffer)
	}
	if buf.results[location.family][location.version] == nil {
		buf.results[location.family][location.version] = make(map[string]*Buffer)
	}
	objects := buf.results[location.family][location.version]
	result := objects[fragment.TID]
	if result == nil {
		result = &Buffer{}
		objects[fragment.TID] = result
	}
	missing, err := result.Push(fragment);
	if  err != nil {
		return err
	}
	if missing != 0 {
		return ErrorInSufficientData
	}
	return nil
}
