package tssd_test

import (
	"errors"
	"testing"

	tssd "github.com/tssdorg/tssdgo/tssd"
)

func TestBufferAppendAndReadRoundTrip(t *testing.T) {
	buf := &tssd.Buffer{MTU: 32}
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
	buf := &tssd.Buffer{MTU: 16}
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
	buf := &tssd.Buffer{}

	if got := buf.Wanted(); got != 1 {
		t.Fatalf("expected Wanted to report no missing fragments, got %d", got)
	}

	first := &tssd.Fragment{Data: []byte("hello"), Schema: tssd.Schema{Hash: "hash", TID: "tid", Fragment: 1}}
	second := &tssd.Fragment{Data: []byte("world"), Schema: tssd.Schema{Hash: "hash", TID: "tid", Fragment: 2}}

	miss, err := buf.Push(first)
	if !errors.Is(err, tssd.ErrorInSufficientData) {
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

	third := &tssd.Fragment{Data: []byte("!"), Schema: tssd.Schema{Hash: "hash", TID: "tid", Fragment: -3}}
	miss, err = buf.Push(third)
	if err != nil || miss != 0 {
		t.Fatalf("pushing third fragment failed: %v", err)
	}

	if got := buf.Wanted(); got != 0 {
		t.Fatalf("expected Wanted to report no missing fragments, got %d", got)
	}
}
