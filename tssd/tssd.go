package tssd

import (
	"errors"
	"fmt"
)

// data type define
const (
	MAGIC               = "TSSD"
	TSSD_VERSION        = 1
	TSSD_FLAT_KIND      = "tssd.Flat"
	TSSD_TIME_KIND      = "time.Time"
	Tbase          int8 = iota + 10
	Tbool               //fix-length-data
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
	Tschema //schema meta data string
	Tarray
	TmergeArray   //merged array, elements including 1 simple fixed length data only
	Tobject //struct	
	Tdict    //map, pairs of (key, value)
	Traw   //raw binary data

	Tuser = 0xEF //user define data
)

var ErrorInvalidTSSDData = errors.New("TSSD data invalid format error or damaged")
var ErrorInSufficientData = errors.New("Need more data to process")
var ErrorTSSDDataSchemaReject = errors.New("TSSD data schema not match")

type Header struct {
	Magic       [4]byte
	Version     int16
	Schema      string // string content
}

func appendHeader(buf []byte, schema string) []byte {

	buf = append(buf, MAGIC...)
	buf = appendSize(buf, TSSD_VERSION)

	buf = append(buf, byte(Tschema))
	return appendString(buf, schema)
}

func isMagic(buf []byte) bool {
	return buf[0] == MAGIC[0] && buf[1] == MAGIC[1] && buf[2] == MAGIC[2] && buf[3] == MAGIC[3]
}

// [TSSD][TSSD_VERSION][Tschema][string-size][xxxxxxx]
func dumpHeader(buf []byte) (header *Header, remain []byte, err error) {
	if len(buf) < 9 {
		return nil, buf, fmt.Errorf("%w [header magic]", ErrorInSufficientData)
	}
	if !isMagic(buf) || buf[6] != byte(Tschema) {
		return nil, buf, fmt.Errorf("%w [magic header not 'TSSD' or schema %d invalid]", ErrorInvalidTSSDData, buf[6])
	}

	header = &Header{
		Magic: [4]byte{'T', 'S', 'S', 'D'},
	}

	header.Version = dumpSize(buf[4:])

	dsize := dumpSize(buf[7:])
	if dsize <= 0 {
		return nil, buf, fmt.Errorf("%w [data version size invalid:%d]", ErrorInvalidTSSDData, dsize)
	}
	if len(buf[9:]) < int(dsize) {
		return nil, buf, fmt.Errorf("%w [data version]", ErrorInSufficientData)
	}
	header.Schema = string(buf[9: 9+dsize])
	return header, buf[9+dsize:], nil
}








