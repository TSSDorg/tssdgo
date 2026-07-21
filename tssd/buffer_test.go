package tssd

import (
	"errors"
	"testing"
	//tssd "github.com/tssdorg/tssdgo/tssd"
)

func TestBuffer(t *testing.T) {
	buf := &Buffer{}
	buf.Append(nil)

	buf.Append([]byte(MAGIC))

	if !isMagic(buf.Fragments[0].tdata) || buf.Size != len(MAGIC) {
		t.Error("Buffer Append MAGIC err")
	}
}

func TestBuffer2(t *testing.T) {
	buf := &Buffer{
		MTU: 2,
	}

	buf.Append([]byte(MAGIC))
	if buf.Size != len(MAGIC) || len(buf.Fragments) != 3 || cap(buf.Fragments[0].tdata) != buf.MTU {
		t.Error("Buffer Append magic err")
	}

	if string(buf.Fragments[0].tdata) != string([]byte(MAGIC)[:buf.MTU]) ||
		string(buf.Fragments[1].tdata) != string([]byte(MAGIC)[buf.MTU:buf.MTU*2]) ||
		string(buf.Fragments[2].tdata) != string([]byte(MAGIC)[buf.MTU*2:]) {
		t.Error("Buffer Append magic content err")
	}

	if d, err := buf.Read(nil); err != nil || len(d) != 0 {
		t.Error("Buffer read 0 should return ok")
	}

	dest := make([]byte, 10)
	if _, err := buf.Read(dest[:0]); err != nil {
		t.Error("Buffer read empty should return ok")
	}

	if _, err := buf.Read(dest); err == nil {
		t.Error("Buffer read oversize should return err")
	}

	d, err := buf.Read(dest[:1])
	if err != nil || len(d) != 1 || d[0] != MAGIC[0] {
		t.Error("Buffer read 1 byte err:", err, d)
	}

	d, err = buf.Read(dest[:4])
	if err != nil || len(d) != 4 || string(d) != string(MAGIC[1:]) || buf.Size != 0 || buf.index != 2 || buf.pos != 1 {
		t.Error("Buffer read 4 bytes err:", err, d, buf)
	}

	if d, err = buf.Read(dest[:1]); err == nil {
		t.Error("Buffer read oversize should return err")
	}
}

func getData(buf *Buffer) [][]byte {

	ret := make([][]byte, len(buf.Fragments))
	for i := 0; i < len(buf.Fragments); i++ {
		ret[i] = buf.Fragments[i].tdata
	}
	return ret
}

func appendBuffer3(t *testing.T, first, second int, r1, r2 [][]byte) {
	buf := &Buffer{
		MTU: 3,
	}

	dest := make([]byte, 11)
	for i := range dest {
		dest[i] = byte(100 + i)
	}

	buf.Append(dest[:first])
	if !SliceSliceEqual(getData(buf), r1) {
		t.Error("Buffer append r1 e rr:", getData(buf), r1)
	}

	buf.Append(dest[:second])
	if !SliceSliceEqual(getData(buf), r2) {
		t.Error("Buffer append r2 err:", getData(buf), r2)
	}
}

func TestAppendBuffer3(t *testing.T) {
	appendBuffer3(t, 0, 0, [][]byte{}, [][]byte{})
	appendBuffer3(t, 0, 1, [][]byte{}, [][]byte{[]byte{100}})
	appendBuffer3(t, 1, 0, [][]byte{[]byte{100}}, [][]byte{[]byte{100}})
	appendBuffer3(t, 1, 1, [][]byte{[]byte{100}}, [][]byte{[]byte{100, 100}})
	appendBuffer3(t, 1, 2, [][]byte{[]byte{100}}, [][]byte{[]byte{100, 100, 101}})
	appendBuffer3(t, 1, 3, [][]byte{[]byte{100}}, [][]byte{[]byte{100, 100, 101}, []byte{102}})
	appendBuffer3(t, 1, 4, [][]byte{[]byte{100}}, [][]byte{[]byte{100, 100, 101}, []byte{102, 103}})
	appendBuffer3(t, 1, 5, [][]byte{[]byte{100}}, [][]byte{[]byte{100, 100, 101}, []byte{102, 103, 104}})

	appendBuffer3(t, 2, 1, [][]byte{[]byte{100, 101}}, [][]byte{[]byte{100, 101, 100}})
	appendBuffer3(t, 2, 2, [][]byte{[]byte{100, 101}}, [][]byte{[]byte{100, 101, 100}, []byte{101}})
	appendBuffer3(t, 2, 3, [][]byte{[]byte{100, 101}}, [][]byte{[]byte{100, 101, 100}, []byte{101, 102}})
	appendBuffer3(t, 2, 4, [][]byte{[]byte{100, 101}}, [][]byte{[]byte{100, 101, 100}, []byte{101, 102, 103}})
	appendBuffer3(t, 2, 5, [][]byte{[]byte{100, 101}}, [][]byte{[]byte{100, 101, 100}, []byte{101, 102, 103}, []byte{104}})
}
/*
func TestAppendBufferWithUserBuffer(t *testing.T) {
	d1 := make([]byte, 2, 2)
	d2 := make([]byte, 1, 1)
	d3 := make([]byte, 3, 4)
	buf := &Buffer{
		MTU: 3,
		Fragments: []Fragment{
			Fragment{Raw: d1},
			Fragment{Raw: d2},
			Fragment{Raw: d3},
		},
	}

	dest := make([]byte, 11)
	for i := range dest {
		dest[i] = byte(100 + i)
	}

	buf.Append(dest[:])
	if !SliceSliceEqual(getData(buf), [][]byte{[]byte{100, 101}, []byte{102}, []byte{103, 104, 105, 106}, []byte{107, 108, 109}, []byte{110}}) {
		t.Error("Buffer append r1 err:", getData(buf))
	}
	buf.Append(dest[:3])
	if !SliceSliceEqual(getData(buf), [][]byte{[]byte{100, 101}, []byte{102}, []byte{103, 104, 105, 106}, []byte{107, 108, 109}, []byte{110, 100, 101}, []byte{102}}) {
		t.Error("Buffer append r1 err:", getData(buf))
	}

	readBuffer3(t, 1, 10, []byte{100}, []byte{101, 102, 103, 104, 105, 106, 107, 108, 109, 110})
	readBuffer3(t, 2, 9, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108, 109, 110})
	readBuffer3(t, 3, 8, []byte{100, 101, 102}, []byte{103, 104, 105, 106, 107, 108, 109, 110})
	readBuffer3(t, 4, 7, []byte{100, 101, 102, 103}, []byte{104, 105, 106, 107, 108, 109, 110})
}*/

func readBuffer3(t *testing.T, first, second int, r1, r2 []byte) {
	buf := &Buffer{
		MTU: 3,
	}

	dest := make([]byte, 11)
	for i := range dest {
		dest[i] = byte(100 + i)
	}

	buf.Append(dest)

	for i := range dest {
		dest[i] = byte(0)
	}
	d, err := buf.Read(dest[:first])
	if err != nil || !SliceEqual(d, r1) || !SliceEqual(dest[:first], r1) {
		t.Error("Buffer read r1 err:", err, d, r1)
	}

	for i := range dest {
		dest[i] = byte(0)
	}
	d, err = buf.Read(dest[:second])
	if err != nil || !SliceEqual(d, r2) || !SliceEqual(dest[:second], r2) {
		t.Error("Buffer read 4 bytes err:", err, d, r2)
	}
}

func TestReadBuffer3(t *testing.T) {
	readBuffer3(t, 0, 1, []byte{}, []byte{100})
	readBuffer3(t, 1, 0, []byte{100}, []byte{})
	readBuffer3(t, 1, 1, []byte{100}, []byte{101})
	readBuffer3(t, 1, 2, []byte{100}, []byte{101, 102})
	readBuffer3(t, 1, 3, []byte{100}, []byte{101, 102, 103})
	readBuffer3(t, 1, 4, []byte{100}, []byte{101, 102, 103, 104})
	readBuffer3(t, 1, 5, []byte{100}, []byte{101, 102, 103, 104, 105})
	readBuffer3(t, 1, 6, []byte{100}, []byte{101, 102, 103, 104, 105, 106})
	readBuffer3(t, 1, 7, []byte{100}, []byte{101, 102, 103, 104, 105, 106, 107})
	readBuffer3(t, 2, 1, []byte{100, 101}, []byte{102})
	readBuffer3(t, 2, 2, []byte{100, 101}, []byte{102, 103})
	readBuffer3(t, 2, 3, []byte{100, 101}, []byte{102, 103, 104})
	readBuffer3(t, 2, 4, []byte{100, 101}, []byte{102, 103, 104, 105})
	readBuffer3(t, 2, 5, []byte{100, 101}, []byte{102, 103, 104, 105, 106})
	readBuffer3(t, 2, 6, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107})
	readBuffer3(t, 2, 7, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108})
	readBuffer3(t, 2, 8, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108, 109})
	readBuffer3(t, 2, 9, []byte{100, 101}, []byte{102, 103, 104, 105, 106, 107, 108, 109, 110})
}

func TestBufferAppendAndReadRoundTrip(t *testing.T) {
	buf := &Buffer{MTU: 32}
	payload := []byte("hello world")

	buf.Append(payload)

	if got := buf.Size; got != len(payload) {
		t.Fatalf("expected buffer size %d, got %d", len(payload), got)
	}

	firstChunk, err := buf.Read(make([]byte, 5))
	if err != nil {
		t.Fatalf("reading first chunk failed: %v", err)
	}
	if string(firstChunk) != "hello" {
		t.Fatalf("expected first chunk %q, got %q", "hello", firstChunk)
	}

	secondChunk, err := buf.Read(make([]byte, len(payload)-5))
	if err != nil {
		t.Fatalf("reading second chunk failed: %v", err)
	}
	if string(secondChunk) != " world" {
		t.Fatalf("expected second chunk %q, got %q", " world", secondChunk)
	}

	if buf.Size != 0 {
		t.Fatalf("expected buffer to be drained, got size %d", buf.Size)
	}
}

func TestBufferPeekByteAndRewind(t *testing.T) {
	buf := &Buffer{MTU: 16}
	buf.Append([]byte("abc"))

	first, err := buf.PeekByte()
	if err != nil {
		t.Fatalf("peek failed: %v", err)
	}
	if first != 'a' {
		t.Fatalf("expected first byte to be 'a', got %q", first)
	}

	got, err := buf.ReadByte()
	if err != nil {
		t.Fatalf("read byte failed: %v", err)
	}
	if got != 'a' {
		t.Fatalf("expected first byte from read to be 'a', got %q", got)
	}

	buf.Rewind()

	rewound, err := buf.ReadByte()
	if err != nil {
		t.Fatalf("rewound read byte failed: %v", err)
	}
	if rewound != 'a' {
		t.Fatalf("expected rewound byte to be 'a', got %q", rewound)
	}
}

func TestBufferPushAndWanted(t *testing.T) {
	buf := &Buffer{}

	if got := buf.Wanted(); got != 1 {
		t.Fatalf("expected Wanted to report no missing fragments, got %d", got)
	}

	first := &Fragment{Raw: []byte("hello"), Schema: Schema{Hash: "hash", TID: "tid", Fragment: 1}}
	second := &Fragment{Raw: []byte("world"), Schema: Schema{Hash: "hash", TID: "tid", Fragment: 2}}

	miss, err := buf.Push(first)
	if !errors.Is(err, ErrorInSufficientData) {
		t.Fatalf("expected insufficient data error, got %v", err)
	}
	if miss != 2 {
		t.Fatalf("expected missing fragment 2 after first push, got %d", miss)
	}

	miss, err = buf.Push(second)
	if err == nil {
		t.Fatalf("pushing second fragment failed: %v", err)
	}
	if miss != 3 {
		t.Fatalf("expected no missing fragments after second push, got %d", miss)
	}

	if got := buf.Wanted(); got != 3 {
		t.Fatalf("expected Wanted to report no missing fragments, got %d", got)
	}

	third := &Fragment{Raw: []byte("!"), Schema: Schema{Hash: "hash", TID: "tid", Fragment: -3}}
	miss, err = buf.Push(third)
	if err != nil || miss != 0 {
		t.Fatalf("pushing third fragment failed: %v", err)
	}

	if got := buf.Wanted(); got != 0 {
		t.Fatalf("expected Wanted to report no missing fragments, got %d", got)
	}
}
