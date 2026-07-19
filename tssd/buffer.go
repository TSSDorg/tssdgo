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
	buf.MTU = max(buf.MTU, TSSD_BUFFER_MIN_MTU)
	buf.schema = &schema
	buf.heads = make([]byte, 0, buf.MTU/3)
	//create a new buffer to receive
	nbuf := &Buffer{
		MTU: buf.MTU,
		Fragments: []Fragment{
			Fragment{
				Data: buf.heads,
				Raw:  buf.heads,
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
	avail := nbuf.MTU - nbuf.Size - TSSD_SIZET_LENGTH - TSSD_SIZEA_LENGTH - TSSD_CHECKSUM_LENGTH
	nbuf.appendSize4(avail) //reserve sizet
	nbuf.appendSize2(avail)
	//we will keep a copy of schema in buf.heads
	if nbuf.Size >= avail { //TSSD Heads too large than the MTU(fragment limitation)
		return ErrorTSSDHeadOverSizeFragment
	}
	buf.heads = nbuf.Fragments[0].Data[:nbuf.Size]
	buf.lenChecksum = TSSD_CHECKSUM_LENGTH
	return nil
}

// reset buf's read info, let user read from begin
func (buf *Buffer) Rewind() *Buffer {
	buf.index, buf.pos, buf.Size = 0, 0, 0
	for i := 0; i <= buf.windex; i++ {
		buf.Size += len(buf.Fragments[i].Data)
	}
	return buf
}

func (buf *Buffer) writePos() (int, int) {
	idx := len(buf.Fragments) - 1
	pos := len(buf.Fragments[idx].Data)
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
	length := len(buf.Fragments[buf.windex].Data)
	appendSize4(buf.Fragments[buf.windex].Raw[:pos-TSSD_SIZET_LENGTH-TSSD_SIZEA_LENGTH], length)
	appendSize2(buf.Fragments[buf.windex].Raw[:pos-TSSD_SIZEA_LENGTH], length)

	//at last we need append checksum when finish(many sizet will update at finish)
	for i := 0; i <= buf.windex; i++ {
		buf.appendChecksum(i)
	}
}

func (buf *Buffer) appendChecksum(index int) {
	checksum := hash(buf.Fragments[index].Raw)
	//exclude all info about checksum
	buf.Fragments[index].Raw = append(buf.Fragments[index].Raw, byte(Tarraym))
	buf.Fragments[index].Raw = append(buf.Fragments[index].Raw, byte(Tuint8))
	buf.Fragments[index].Raw = appendSize4(buf.Fragments[index].Raw, len(checksum))
	buf.Fragments[index].Raw = appendSize2(buf.Fragments[index].Raw, len(checksum))
	buf.Fragments[index].Raw = append(buf.Fragments[index].Raw, checksum...)
}

func (buf *Buffer) copyAndUpdate(bs []byte) {
	buf.Fragments[buf.windex].Data = append(buf.Fragments[buf.windex].Data, bs...)
	l := len(buf.Fragments[buf.windex].Raw)
	buf.Fragments[buf.windex].Raw = buf.Fragments[buf.windex].Raw[:l+len(bs)]
	buf.Size += len(bs)
}

func (buf *Buffer) Append(bs []byte) *Buffer {
	for len(bs) > 0 {
		if buf.windex == len(buf.Fragments) {
			if buf.MTU == 0 {
				buf.MTU = TSSD_BUFFER_MTU
			}
			b := make([]byte, len(buf.heads), buf.MTU)
			copy(b, buf.heads)

			buf.Fragments = append(buf.Fragments,
				Fragment{
					Data: b[len(buf.heads):],
					Raw:  b,
				})
			if buf.schema != nil {
				buf.Fragments[buf.windex].Schema = *buf.schema
			}
			buf.updateFragmentID(buf.windex, buf.windex+1)
		}

		if len(buf.Fragments[buf.windex].Data)+len(bs) <= buf.avail(buf.windex) {
			buf.copyAndUpdate(bs)
			return buf
		}

		fill := buf.avail(buf.windex) - len(buf.Fragments[buf.windex].Data)
		buf.copyAndUpdate(bs[:fill])
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
	return buf.Fragments[buf.index].Data[buf.pos], nil
}

func (buf *Buffer) avail(index int) int {
	return cap(buf.Fragments[buf.index].Data) - buf.lenChecksum
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

	n := copy(dest[:wanted], buf.Fragments[buf.index].Data[buf.pos:buf.avail(buf.index)])
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
		n = copy(dest, buf.Fragments[buf.index].Data[:buf.avail(buf.index)])
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

func (buf *Buffer) appendSize2(le int) *Buffer {
	l := uint16(le)
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
	if len(buf.Fragments[index].Raw) < 17 {
		return
	}
	l := int16(n)
	s := Slice(Ptr(&l), unsafe.Sizeof(l))
	copy(buf.Fragments[index].Raw[16:], s)
}

func (buf *Buffer) updateSize(index, pos, value int) {
	l := int32(value)
	s := Slice(Ptr(&l), unsafe.Sizeof(l))
	for i := 0; i < len(s); i++ {
		buf.Fragments[index].Data[pos] = s[i]
		pos++
		if pos >= buf.avail(index) {
			pos = 0
			index++
		}
	}
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
		buf.Size -= len(buf.Fragments[fid-1].Data)
	}

	//we always copy it, even repeat push
	buf.Fragments[fid-1] = *frag
	buf.Size += len(buf.Fragments[fid-1].Data)
	if len(buf.heads) == 0 && len(buf.Fragments[0].Raw) > len(buf.Fragments[0].Data) {
		buf.heads = buf.Fragments[0].Raw[0 : len(buf.Fragments[0].Raw)-len(buf.Fragments[0].Data)]
		buf.lenChecksum = TSSD_CHECKSUM_LENGTH
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
			Raw:    make([]byte, 0, len(buf.Fragments[0].Raw)-len(buf.Fragments[0].Data)+buf.Size),
		},
	}
	frag := &buf.Fragments[0] //new one
	headLen := len(frags[0].Raw) - TSSD_CHECKSUM_LENGTH - len(frags[0].Data)
	frag.Raw = append(frag.Raw, frags[0].Raw[:headLen]...)
	frag.Data = frag.Raw[headLen:headLen]
	buf.Size = 0
	buf.windex = 0
	buf.MTU = cap(frag.Raw)
	if len(buf.heads) == 0 {
		buf.heads = frag.Raw[0:headLen]
	}

	for i := 0; i < len(frags); i++ {
		buf.Append(frags[i].Data)
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
	return buf.Append(frag.Data)
}
