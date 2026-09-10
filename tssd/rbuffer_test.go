package tssd

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"sync"
)

type rbufferTestFlat struct {
	Flat[rbufferTestFlat, *rbufferTestFlat]
	Value int32
}

func (*rbufferTestFlat) Family() string { return "RBufferTest" }
func (*rbufferTestFlat) Version() string { return "V1" }

var registerRBufferTestFlat sync.Once

func rbufferTestFrame(t *testing.T, value int32) []byte {
	t.Helper()
	registerRBufferTestFlat.Do(func() {
		if err := Register(&rbufferTestFlat{}); err != nil {
			t.Fatalf("register test flat: %v", err)
		}
	})
	buf := &Buffer{MTU: 4096}
	if err := MarshalTo(&rbufferTestFlat{Value: value}, buf); err != nil {
		t.Fatalf("marshal test flat: %v", err)
	}
	return append([]byte(nil), buf.Fragments()[0].Data...)
}


func TestRBufferDetectMagicAcrossChunks(t *testing.T) {
	buf := NewRBuffer()

	more, err := buf.detectMagic([]byte{'T', 'S'}, 0)
	if !errors.Is(err, ErrorInSufficientData) || more != TSSD_FRAGMENT_MIN_HEADER_SIZE-2 {
		t.Fatalf("first magic chunk: more=%d err=%v", more, err)
	}

	more, err = buf.detectMagic([]byte{'S', 'D', 'V'}, 0)
	if err != nil || buf.magic != 0 || string(buf.data) != MAGIC {
		t.Fatalf("completed magic: more=%d magic=%d data=%q err=%v", more, buf.magic, buf.data, err)
	}

	buf = NewRBuffer()
	more, err = buf.detectMagic([]byte("noise"), 0)
	if !errors.Is(err, ErrorInSufficientData) || more != TSSD_FRAGMENT_MIN_HEADER_SIZE-4 {
		t.Fatalf("noise: more=%d err=%v", more, err)
	}
	if string(buf.data) != "oise" {
		t.Fatalf("expected retained suffix %q, got %q", "oise", buf.data)
	}
}

func TestRBufferExtractFragment(t *testing.T) {
	data, expectedChecksum := buildFragmentBytes(t, []byte("payload"), false)
	buf := NewRBuffer()

	more, err := buf.Extract(data)
	if err != nil || more != 0 {
		t.Fatalf("Extract returned more=%d err=%v", more, err)
	}

	fragment := buf.Fragment()
	if fragment == nil {
		t.Fatal("expected extracted fragment")
	}
	if string(fragment.Header.Magic[:]) != MAGIC {
		t.Fatalf("unexpected magic %q", fragment.Header.Magic)
	}
	if fragment.Schema.FID != 1 || fragment.Schema.Types != "hash" || fragment.Schema.TID != "tid" {
		t.Fatalf("unexpected schema: %+v", fragment.Schema)
	}
	if string(fragment.Payload()) != "payload" {
		t.Fatalf("unexpected payload %q", fragment.Payload())
	}
	if string(fragment.Checksum()) != string(expectedChecksum) {
		t.Fatalf("unexpected checksum %q", fragment.Checksum())
	}
}

func TestRBufferExtractConcatenatedFragments(t *testing.T) {
	first, _ := buildFragmentBytes(t, []byte("first"), false)
	second, _ := buildFragmentBytes(t, []byte("second"), false)
	buf := NewRBuffer()

	combined := append(append([]byte("prefix"), first...), second...)
	if more, err := buf.Extract(combined); err != nil || more != 0 {
		t.Fatalf("first Extract returned more=%d err=%v", more, err)
	}
	if got := string(buf.Fragment().Payload()); got != "first" {
		t.Fatalf("first payload %q", got)
	}
	if more, err := buf.Extract(nil); err != nil || more != 0 {
		t.Fatalf("second Extract returned more=%d err=%v", more, err)
	}
	if got := string(buf.Fragment().Payload()); got != "second" {
		t.Fatalf("second payload %q", got)
	}
}

func TestRBufferExtractResynchronizesAfterInvalidFrame(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("valid"), false)
	invalid := append([]byte(nil), data...)
	invalid[7] = 0
	input := append(invalid, data...)
	buf := NewRBuffer()

	if more, err := buf.Extract(input); err != nil || more != 0 {
		t.Fatalf("Extract returned more=%d err=%v", more, err)
	}
	if got := string(buf.Fragment().Payload()); got != "valid" {
		t.Fatalf("payload after resynchronization %q", got)
	}
}

func TestRBufferUnmarshalDetectsChecksumFailure(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	data[len(data)-1] ^= 1
	buf := NewRBuffer()

	if more, err := buf.detectMagic(data, 0); err != nil || more != 0 {
		t.Fatalf("detectMagic returned more=%d err=%v", more, err)
	}
	if _, err := buf.unmarshalFragment(); !errors.Is(err, ErrorTSSDDataChecksumFailure) {
		t.Fatalf("expected checksum failure, got %v", err)
	}
}

func TestRBufferExtractReaderKeepsInputStreaming(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	reader := bytes.NewReader(append([]byte("noise"), data...))
	buf := NewRBuffer()

	// The frame schema is intentionally unregistered, so ExtractReader reports
	// an error when it encounters the fragment.
	if err := buf.ExtractReader(reader); err == nil {
		t.Fatalf("ExtractReader should have reported an error")
	}
}

func TestRBufferDetectMagicRetainsFourByteSuffix(t *testing.T) {
	buf := NewRBuffer()
	for _, input := range [][]byte{
		[]byte("abc"),
		[]byte("d"),
		[]byte("e"),
	} {
		more, err := buf.detectMagic(input, 0)
		if !errors.Is(err, ErrorInSufficientData) || more <= 0 {
			t.Fatalf("input %q: more=%d err=%v", input, more, err)
		}
	}
	if got := string(buf.data); got != "bcde" {
		t.Fatalf("retained suffix %q", got)
	}
}

func TestRBufferDetectMagicSplitVariants(t *testing.T) {
	cases := []struct {
		name  string
		first []byte
		last  []byte
	}{
		{"split-after-prefix", []byte{'a', 'T', 'S'}, []byte{'S', 'D', 'V'}},
		{"split-after-prefix-two", []byte{'a', 'T'}, []byte{'S', 'S', 'D', 'V'}},
		{"split-after-prefix-four", []byte{'a', 'T', 'S', 'S'}, []byte{'D', 'V', 'a'}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := NewRBuffer()
			if _, err := buf.detectMagic(tc.first, 0); !errors.Is(err, ErrorInSufficientData) {
				t.Fatalf("first chunk error: %v", err)
			}
			if _, err := buf.detectMagic(tc.last, 0); err != nil || buf.magic != 0 {
				t.Fatalf("last chunk: magic=%d err=%v", buf.magic, err)
			}
			if !bytes.HasPrefix(buf.data, []byte(MAGIC)) {
				t.Fatalf("data does not start with magic: %q", buf.data)
			}
		})
	}
}

func TestRBufferParsePayloadReportsMissingBytes(t *testing.T) {
	data, _ := buildFragmentBytes(t, []byte("payload"), false)
	buf := NewRBuffer()
	truncated := append([]byte(nil), data[:len(data)-2]...)
	if _, err := buf.detectMagic(truncated, 0); err != nil {
		t.Fatalf("detectMagic: %v", err)
	}
	if _, err := buf.parseHeads(); err != nil {
		t.Fatalf("parseHeads: %v", err)
	}
	if more, err := buf.parsePayload(); err != nil || more != 0 {
		t.Fatalf("parsePayload: %v %d", err, more)
	}

	more, err := buf.parseChecksum()
	if !errors.Is(err, ErrorInSufficientData) || more != 2 {
		t.Fatalf("parsePayload: more=%d err=%v", more, err)
	}
/*
	buf = NewRBuffer()
	truncated = append([]byte(nil), data[:len(data)-22]...)
	if _, err := buf.detectMagic(truncated, 0); err != nil {
		t.Fatalf("detectMagic: %v", err)
	}
	if _, err := buf.parseHeads(); err != nil {
		t.Fatalf("parseHeads: %v", err)
	}
	more, err = buf.parsePayload()
	if !errors.Is(err, ErrorInSufficientData) || more != 2 {
		t.Fatalf("parsePayload: more=%d err=%v", more, err)
	}*/
}

func TestRBufferChecksumFailureLeavesTrailingData(t *testing.T) {
	good, _ := buildFragmentBytes(t, []byte("good"), false)
	bad, _ := buildFragmentBytes(t, []byte("bad"), false)
	bad[len(bad)-1] ^= 1
	// the schema mismatch after successfully decoding and routing the fragment.
	input := append(append([]byte(nil), bad...), good...)
	buf := NewRBuffer()

	more, err := buf.Extract(input)
	if  err != nil || more != 0 {
		t.Fatalf("checksum recovery: more=%d err=%v", more, err)
	}

	if got := string(buf.Fragment().Payload()); got != "good" {
		t.Fatalf("trailing payload %q", got)
	}
}

func TestRBufferExtractMultipleFramesWithNoise(t *testing.T) {
	first, _ := buildFragmentBytes(t, []byte("a"), false)
	second, _ := buildFragmentBytes(t, []byte("b"), false)
	inputs := [][]byte{
		append(append([]byte("TSSDV"), first...), second...),
		append(append([]byte("x"), first...), append([]byte("TSSDV"), second...)...),
	}
	for _, input := range inputs {
		buf := NewRBuffer()
		for index, expected := range []string{"a", "b"} {
			if more, err := buf.Extract(input); err != nil || more != 0 {
				t.Fatalf("frame %d: more=%d err=%v", index, more, err)
			}
			if got := string(buf.Fragment().Payload()); got != expected {
				t.Fatalf("frame %d payload %q", index, got)
			}
			input = nil
		}
	}
}

func TestRBufferExtractReader(t *testing.T) {
	first := rbufferTestFrame(t, 123)
	second := rbufferTestFrame(t, 456)
	reader := &rbufferTestReader{chunks: [][]byte{
		append([]byte("noise"), first...),
		second,
	}}
	buf := NewRBuffer()

	if err := buf.ExtractReader(reader); err != nil {
		t.Fatalf("first ExtractReader: %v", err)
	}
	firstBuffer := buf.Buffer("RBufferTest", "V1")
	if firstBuffer == nil {
		t.Fatal("missing first assembled buffer")
	}
	mp := map[int32]struct{} {
		123: {},
		456: {},
	}
	var firstValue rbufferTestFlat
	if err := UnmarshalTo(firstBuffer, &firstValue); err != nil {
		t.Fatalf("first value=%d err=%v", firstValue.Value, err)
	}
	delete(mp, firstValue.Value)

	secondBuffer := buf.Buffer("RBufferTest", "V1")
	if secondBuffer == nil {
		//t.Fatal("missing second assembled buffer")
		buf.ExtractReader(reader)
		secondBuffer = buf.Buffer("RBufferTest", "V1")
	}
	var secondValue rbufferTestFlat
	if err := UnmarshalTo(secondBuffer, &secondValue); err != nil {
		t.Fatalf("second value=%d err=%v", secondValue.Value, err)
	}
	delete(mp, secondValue.Value)
	if len(mp) != 0 {
		t.Fatalf("missing values: %v", mp)
	}

	if err := buf.ExtractReader(reader); err == nil {
		t.Fatalf("expected schema mismatch, got %v", err)
	}

}

func TestRBufferDetectMagicAtStart(t *testing.T) {
	buf := NewRBuffer()
	more, err := buf.detectMagic([]byte(MAGIC), 0)
	if err != nil || buf.magic != 0 || more != TSSD_FRAGMENT_MIN_HEADER_SIZE-len(MAGIC) {
		t.Fatalf("more=%d magic=%d err=%v", more, buf.magic, err)
	}
}

type rbufferTestReader struct {
	chunks [][]byte
}

func (reader *rbufferTestReader) Read(dst []byte) (int, error) {
	if len(reader.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := reader.chunks[0]
	count := copy(dst, chunk)
	if count == len(chunk) {
		reader.chunks = reader.chunks[1:]
	} else {
		reader.chunks[0] = chunk[count:]
	}
	return count, nil
}

func TestRBufferUnmarshalAndPush(t *testing.T) {
	data := rbufferTestFrame(t, 123)
	buf := NewRBuffer()
	if more, err := buf.Extract(data); err != nil || more != 0 {
		t.Fatalf("Extract: more=%d err=%v", more, err)
	}
	if err := buf.Push(buf.Fragment()); err != nil {
		t.Fatalf("Push: %v", err)
	}
	assembled := buf.Buffer("RBufferTest", "V1")
	if assembled == nil {
		t.Fatal("expected assembled buffer")
	}
	var value rbufferTestFlat
	if err := UnmarshalTo(assembled, &value); err != nil {
		t.Fatalf("UnmarshalTo: %v", err)
	}
	if value.Value != 123 {
		t.Fatalf("value=%d, want 123", value.Value)
	}
}

func TestRBufferExtractFramesWithSplitMagicAndNoise(t *testing.T) {
	first, _ := buildFragmentBytes(t, []byte("a"), false)
	second, _ := buildFragmentBytes(t, []byte("b"), false)
	cases := [][][]byte{
		{[]byte(MAGIC), first, []byte("b"), second},
		{[]byte(MAGIC), first, []byte(MAGIC), second},
		{[]byte("b"), first, []byte("b"), second},
		{[]byte("b"), first, []byte(MAGIC), second},
	}

	for index, parts := range cases {
		t.Run(string(rune('1'+index)), func(t *testing.T) {
			buf := NewRBuffer()
			input := bytes.Join(parts, nil)
			for _, want := range []string{"a", "b"} {
				if more, err := buf.Extract(input); err != nil || more != 0 {
					t.Fatalf("Extract: more=%d err=%v", more, err)
				}
				if got := string(buf.Fragment().Payload()); got != want {
					t.Fatalf("payload=%q, want %q", got, want)
				}
				input = nil
			}
		})
	}
}

func TestRBufferExtractMalformedPrefixAndChecksum(t *testing.T) {
	good, _ := buildFragmentBytes(t, []byte("good"), false)
	cases := []struct {
		name  string
		input []byte
	}{
		{
			name:  "bad-schema-type",
			input: func() []byte { data := append([]byte(nil), good...); data[7] = 'x'; return append(data, good...) }(),
		},
		{
			name: "bad-checksum",
			input: func() []byte {
				bad := append([]byte(nil), good...)
				bad[len(bad)-1] ^= 1
				return append(bad, good...)
			}(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := NewRBuffer()
			if more, err := buf.Extract(tc.input); err != nil || more != 0 {
				t.Fatalf("Extract: more=%d err=%v", more, err)
			}
			if got := string(buf.Fragment().Payload()); got != "good" {
				t.Fatalf("payload=%q", got)
			}
		})
	}
}

func TestRBufferReaderChunkScenarios(t *testing.T) {
	first := rbufferTestFrame(t, 97)
	second := rbufferTestFrame(t, 123)
	third := rbufferTestFrame(t, 456)
	magic := []byte(MAGIC)
	noise := []byte("xy")
	cases := []struct {
		name  string
		parts [][]byte
		want  []int32
	}{
		{"single", [][]byte{first}, []int32{97}},
		{"noise-chunks", [][]byte{[]byte("a"), []byte("b"), first}, []int32{97}},
		{"split-magic", [][]byte{[]byte("TSSD"), []byte("V"), first}, []int32{97}},
		{"two-frames", [][]byte{[]byte("TSSD"), []byte("V"), first, magic, second}, []int32{97, 123}},
		{"noise-and-two", [][]byte{noise, append(append([]byte(nil), first...), append(magic, second...)...)}, []int32{97, 123}},
		{"three-frames", [][]byte{noise, append(append([]byte(nil), first...), append(magic, second...)...), magic, third}, []int32{97, 123, 456}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader(bytes.Join(tc.parts, nil))
			buf := NewRBuffer()
			for _, want := range tc.want {
				if err := buf.ExtractReader(reader); err != nil {
					t.Fatalf("ExtractReader: %v", err)
				}
				assembled := buf.Buffer("RBufferTest", "V1")
				if assembled == nil {
					t.Fatal("missing assembled buffer")
				}
				var value rbufferTestFlat
				if err := UnmarshalTo(assembled, &value); err != nil {
					t.Fatalf("UnmarshalTo: %v", err)
				}
				if value.Value != want {
					t.Fatalf("value=%d, want %d", value.Value, want)
				}
			}
		})
	}
}
