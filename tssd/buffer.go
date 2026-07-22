package tssd

import (
	//"fmt"
	"unsafe"
)

type Buffer struct {
	schema      *Schema
	heads       []byte
	lenChecksum int
	MTU         int
	Size        int        //total size
	index       int        //read index
	pos         int        //read position
	windex      int        //write index
	Fragments   []Fragment //framents list sending/received
}

func (buf *Buffer) prepare(schema Schema) error {
	if buf.MTU == 0 {
		buf.MTU = TSSD_BUFFER_MTU
	}
	//buf.Clear()
	buf.MTU = max(buf.MTU, TSSD_BUFFER_MIN_MTU)
	buf.schema = &schema
	buf.heads = make([]byte, 0, buf.MTU/3)
	//create a new buffer to receive
	nbuf := &Buffer{
		MTU: buf.MTU,
		Fragments: []Fragment{
			Fragment{
				tdata: buf.heads,
				Data:  buf.heads,
			},
		},
	}

	nbuf.Append([]byte(MAGIC))
	nbuf.Append([]byte{TSSD_VERSION_MINOR, TSSD_VERSION_MAJOR, Tschema})

	err := buf.schema.Marshal(nbuf)
	if err != nil {
		return err
	}
	nbuf.Append([]byte{byte(Tarraym), byte(Tuint8)})
	//we will try to calc the real size of Checksum
	buf.lenChecksum = 8 + len(ChecksumFunc(buf.heads)) //8 bytes for [Tarraym][Tuint8][sizet/4B][sizea/2B]
	avail := nbuf.MTU - nbuf.Size - TSSD_SIZET_LENGTH - TSSD_SIZEA_LENGTH - buf.lenChecksum
	nbuf.appendSize4(avail) //reserve sizet
	nbuf.appendSize2(avail)
	//we will keep a copy of schema in buf.heads
	if nbuf.Size >= avail { //TSSD Heads too large than the MTU(fragment limitation)
		return ErrorTSSDHeadOverSizeFragment
	}
	buf.heads = nbuf.Fragments[0].tdata[:nbuf.Size]
	return nil
}

// reset buf's read info, let user read from begin
func (buf *Buffer) Rewind() *Buffer {
	buf.index, buf.pos, buf.Size = 0, 0, 0
	for i := 0; i < len(buf.Fragments); i++ {
		buf.Size += len(buf.Fragments[i].tdata)
	}
	return buf
}

func (buf *Buffer) Clear() *Buffer {
	buf.index, buf.pos, buf.Size = 0, 0, 0
	buf.heads = buf.heads[:0]
	buf.lenChecksum = 0

	buf.Fragments = buf.Fragments[:0]
	return buf
}

func (buf *Buffer) writePos() (int, int) {
	idx := len(buf.Fragments) - 1
	pos := len(buf.Fragments[idx].tdata)
	if pos >= buf.avail(idx) {
		pos = 0
		idx++
	}
	return idx, pos
}

func (buf *Buffer) finish() {
	//we need update fragment id
	buf.updateFragmentID(buf.windex, -(buf.windex + 1)) //mark ending fragment

	//update the real fragment data size for the last fragment
	pos := len(buf.heads)
	length := len(buf.Fragments[buf.windex].tdata)
	appendSize4(buf.Fragments[buf.windex].Data[:pos-TSSD_SIZET_LENGTH-TSSD_SIZEA_LENGTH], length)
	appendSize2(buf.Fragments[buf.windex].Data[:pos-TSSD_SIZEA_LENGTH], length)

	//reset Data to the real size, we will use Data to send out
	buf.Fragments[buf.windex].Data = buf.Fragments[buf.windex].Data[:pos+length]

	//at last we need append checksum when finish(many sizet will update at finish)
	for i := 0; i <= buf.windex; i++ {
		buf.appendChecksum(i)
	}
}

func (buf *Buffer) appendChecksum(index int) {
	checksum := ChecksumFunc(buf.Fragments[index].Data)
	//exclude all info about checksum
	buf.Fragments[index].Data = append(buf.Fragments[index].Data, byte(Tarraym))
	buf.Fragments[index].Data = append(buf.Fragments[index].Data, byte(Tuint8))
	buf.Fragments[index].Data = appendSize4(buf.Fragments[index].Data, len(checksum))
	buf.Fragments[index].Data = appendSize2(buf.Fragments[index].Data, len(checksum))
	buf.Fragments[index].Data = append(buf.Fragments[index].Data, checksum...)
}

func (buf *Buffer) Append(bs []byte) *Buffer {
	for len(bs) > 0 {
		if buf.windex == len(buf.Fragments) {
			if buf.MTU == 0 {
				buf.MTU = TSSD_BUFFER_MTU
			}
			b := make([]byte, buf.MTU, buf.MTU)
			copy(b, buf.heads)

			buf.Fragments = append(buf.Fragments,
				Fragment{
					tdata: b[len(buf.heads):len(buf.heads)],
					Data:  b[:buf.MTU-buf.lenChecksum],
				})
			if buf.schema != nil {
				buf.Fragments[buf.windex].Schema = *buf.schema
			}
			buf.updateFragmentID(buf.windex, buf.windex+1)
		}

		if len(buf.Fragments[buf.windex].tdata)+len(bs) <= buf.avail(buf.windex) {
			buf.Fragments[buf.windex].tdata = append(buf.Fragments[buf.windex].tdata, bs...)
			buf.Size += len(bs)
			return buf
		}

		fill := buf.avail(buf.windex) - len(buf.Fragments[buf.windex].tdata)
		buf.Fragments[buf.windex].tdata = append(buf.Fragments[buf.windex].tdata, bs[:fill]...)
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

// peak a byte from buffer without change read off
func (buf *Buffer) PeekByte() (b byte, err error) {
	if buf.Size == 0 {
		return 0, ErrorInSufficientData
	}
	return buf.Fragments[buf.index].tdata[buf.pos], nil
}

func (buf *Buffer) avail(index int) int {
	return cap(buf.Fragments[index].tdata) - buf.lenChecksum
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

	n := copy(dest[:wanted], buf.Fragments[buf.index].tdata[buf.pos:buf.avail(buf.index)])
	buf.Size -= n
	buf.pos += n
	if buf.pos >= buf.avail(buf.index) {
		buf.pos = 0
		buf.index++
	}
	if n >= wanted {
		return result, nil
	}

	dest = dest[n:]
	for {
		n = copy(dest, buf.Fragments[buf.index].tdata[:buf.avail(buf.index)])
		buf.Size -= n
		dest = dest[n:]
		if len(dest) == 0 {
			break
		}
		buf.index++
	}
	buf.pos += n
	if buf.pos >= buf.avail(buf.index) {
		buf.pos = 0
		buf.index++
	}
	return result, nil
}

// common version append diretly without check data cap
// call when you need update heads or checksum.
func appendSize2(dest []byte, le int) []byte {
	l := int16(le)
	return append(dest, Slice(Ptr(&l), unsafe.Sizeof(l))...)
}

func appendSize4(dest []byte, le int) []byte {
	l := int32(le)
	return append(dest, Slice(Ptr(&l), unsafe.Sizeof(l))...)
}

// Buffer version append Buffer.tdata only with check data cap
// call when you add new data to the buffer.
func (buf *Buffer) appendSize2(le int) *Buffer {
	l := int16(le)
	return buf.Append(Slice(Ptr(&l), unsafe.Sizeof(l)))
}

func (buf *Buffer) appendSize4(le int) *Buffer {
	l := int32(le)
	return buf.Append(Slice(Ptr(&l), unsafe.Sizeof(l)))
}

func (buf *Buffer) appendString(s string) *Buffer {
	return buf.appendSize4(len(s)).Append([]byte(s))
}

// [TSSD][Tversion][TSSD_VERSION_MINOR][TSSD_VERSION_MAJOR][Tschema][Tobject][sizet/4B][sizea/2B][FID][...]
func (buf *Buffer) updateFragmentID(index, n int) {
	if len(buf.heads) == 0 || len(buf.Fragments[index].Data) < 17 {
		return
	}
	l := int16(n)
	s := Slice(Ptr(&l), unsafe.Sizeof(l))
	copy(buf.Fragments[index].Data[16:], s)
}

func (buf *Buffer) updateSize(index, pos, value int) {
	l := int32(value)
	s := Slice(Ptr(&l), unsafe.Sizeof(l))
	for i := 0; i < len(s); i++ {
		buf.Fragments[index].tdata[pos] = s[i]
		pos++
		if pos >= buf.avail(index) {
			pos = 0
			index++
		}
	}
}

func (buf *Buffer) dumpSize2() (int, error) {
	var size int16
	_, err := buf.Read(Slice(Ptr(&size), TSSD_SIZEA_LENGTH))
	return int(size), err
}

func (buf *Buffer) dumpSize4() (int, error) {
	var size int32
	_, err := buf.Read(Slice(Ptr(&size), TSSD_SIZET_LENGTH))
	return int(size), err
}

// check and dump sizet
func (buf *Buffer) checkDumpSizet() (sizet int, err error) {
	if sizet, err = buf.dumpSize4(); err != nil {
		return 0, err
	}
	if buf.Size < sizet {
		//TODO, add field name info
		return 0, ErrorInSufficientData
	}
	return sizet, nil
}

// check and dump sizet, sizea
func (buf *Buffer) checkDumpSize() (sizet int, sizea int, err error) {
	if sizet, err = buf.checkDumpSizet(); err != nil {
		return
	}
	//we have check total size in checkDumpSizet, so dump sizea directly
	sizea, err = buf.dumpSize2()
	return sizet, sizea, err
}

// return fragment id with error if lost
func (buf *Buffer) Push(frag *Fragment) (miss int, err error) {
	if buf.schema == nil {
		buf.schema = &frag.Schema
	} else {
		if buf.schema.Hash != frag.Schema.Hash {
			err = ErrorTSSDDataSchemaUnmatch
		}
		if buf.schema.TID != frag.Schema.TID {
			err = ErrorTSSDDataFragmentIDUnmatch
		}
		if err != nil {
			if miss = buf.Wanted(); miss != 0 {
				return miss, err
			}
			return 0, nil
		}
	}
	fid := int(frag.Schema.Fragment) //fid: [1, 2, 3.. -n], the last < 0
	if fid < 0 {
		fid = -fid
	}
	if cap(buf.Fragments) < fid {
		frags := buf.Fragments
		buf.Fragments = make([]Fragment, cap(frags)+32)
		for i := 0; i < cap(frags); i++ {
			buf.Fragments[i] = frags[i]
		}
	}

	if buf.Fragments[fid-1].Schema.Fragment != 0 {
		buf.Size -= len(buf.Fragments[fid-1].tdata)
	}

	//we always copy it, even repeat push
	buf.Fragments[fid-1] = *frag
	buf.Size += len(buf.Fragments[fid-1].tdata)
	if len(buf.heads) == 0 && len(buf.Fragments[0].Data) > len(buf.Fragments[0].tdata) {
		buf.heads = buf.Fragments[0].Data[0 : len(buf.Fragments[0].Data)-len(buf.Fragments[0].tdata)]
		buf.lenChecksum = 8 + len(HashFunc(buf.heads)) //8 bytes for [Tarraym][Tuint8][sizet/4B][sizea/2B]
	}

	if miss = buf.Wanted(); miss != 0 {
		return miss, ErrorInSufficientData
	}
	return 0, nil
}

// return the n-th Fragment that missing
// return 0 if all fragments are present
func (buf *Buffer) Wanted() int {
	//find if we miss one
	for i := 0; i < cap(buf.Fragments); i++ {
		switch {
		case buf.Fragments[i].Schema.Fragment == 0:
			return i + 1
		case buf.Fragments[i].Schema.Fragment < 0:
			//we hit last one, reset the size
			buf.Fragments = buf.Fragments[0 : i+1]
			return 0
		default:
		}
	}
	return 1
}

// merge all Fragments into one
func (buf *Buffer) Merge() *Buffer {
	if len(buf.Fragments) < 2 {
		return buf
	}
	frags := buf.Fragments //old frags
	buf.Fragments = []Fragment{
		Fragment{
			Header: frags[0].Header,
			Schema: frags[0].Schema,
			Data:   make([]byte, 0, len(buf.Fragments[0].Data)-len(buf.Fragments[0].tdata)+buf.Size),
		},
	}
	frag := &buf.Fragments[0] //new one
	headLen := len(frags[0].Data) - buf.lenChecksum - len(frags[0].tdata)
	frag.Data = append(frag.Data, frags[0].Data[:headLen]...)
	frag.tdata = frag.Data[headLen:headLen]
	buf.Size = 0
	buf.windex = 0
	buf.MTU = cap(frag.Data)
	if len(buf.heads) == 0 {
		buf.heads = frag.Data[0:headLen]
	}

	for i := 0; i < len(frags); i++ {
		buf.Append(frags[i].tdata)
	}
	buf.finish()
	return buf
}

// split large fragments into small ones
func (buf *Buffer) Split(mtu int) *Buffer {
	nMTU := max(mtu, TSSD_BUFFER_MIN_MTU)
	if buf.MTU < nMTU {
		return buf
	}
	// merge first
	buf.Merge()
	frag := &buf.Fragments[0]

	// init buf by the new mtu
	buf.MTU = nMTU
	buf.Size = 0
	buf.windex = 0
	buf.index = 0
	buf.pos = 0
	buf.Fragments = []Fragment{}

	// append all data back
	return buf.Append(frag.tdata)
}
